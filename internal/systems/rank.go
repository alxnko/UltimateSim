package systems

import (
	"errors"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Shell Phase: Rank & Leadership Claims
// Pure helper functions resolving the political rank of any entity (Citizen, Ruler,
// Sovereign) and letting the player claim leadership of their city by out-hooking the
// current AdministrationMarker holder. The influence score uses the exact same formula
// as the organic election in leadership_emergence.go (sum of positive incoming hooks),
// so a player claim and an organic emergence are always judged by the same physics.
//
// All functions here run complete (never broken-out) queries and apply structural
// mutations strictly after every query loop has finished. They must be called from
// outside any open query.Next() loop.

// Rank tier constants.
const (
	RankCitizen   uint8 = 0
	RankRuler     uint8 = 1
	RankSovereign uint8 = 2
)

// ErrClaimDenied is returned by ClaimLeadership when the claimant has no city or the
// sitting ruler's influence score is too strong to be displaced.
var ErrClaimDenied = errors.New("leadership claim denied: claimant has no city or the current ruler is stronger")

// RankName maps a rank tier constant to its display name.
func RankName(r uint8) string {
	switch r {
	case RankRuler:
		return "Ruler"
	case RankSovereign:
		return "Sovereign"
	default:
		return "Citizen"
	}
}

// GetRank resolves the political rank of an entity along with its city and country IDs
// (both taken from the entity's Affiliation, returned regardless of rank).
//   - Ruler: the entity carries the AdministrationMarker and belongs to a city (CityID != 0).
//   - Sovereign: a Ruler whose ruled settlement (the Village whose Affiliation.CityID
//     matches) is a Capital (carries CapitalComponent).
//   - Citizen: everything else.
func GetRank(world *ecs.World, ent ecs.Entity) (rank uint8, cityID uint32, countryID uint32) {
	affilID := ecs.ComponentID[components.Affiliation](world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](world)
	villageID := ecs.ComponentID[components.Village](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)

	if !world.Alive(ent) || !world.Has(ent, affilID) {
		return RankCitizen, 0, 0
	}

	affil := (*components.Affiliation)(world.Get(ent, affilID))
	cityID = affil.CityID
	countryID = affil.CountryID

	if cityID == 0 || !world.Has(ent, adminMarkerID) {
		return RankCitizen, cityID, countryID
	}

	// Ruler confirmed. Sovereign additionally requires the ruled settlement to be a Capital.
	// Scan the Village+Affiliation archetypes fully (no early break, keeps the world unlocked).
	isSovereign := false
	query := world.Query(filter.All(villageID, affilID))
	for query.Next() {
		vAffil := (*components.Affiliation)(query.Get(affilID))
		if vAffil.CityID == cityID && query.Has(capitalID) {
			isSovereign = true
		}
	}

	if isSovereign {
		return RankSovereign, cityID, countryID
	}
	return RankRuler, cityID, countryID
}

// CityRuler scans NPC+Identity+Affiliation+AdministrationMarker entities and returns the
// current ruler of the given city: its entity, its Identity.ID, and whether one exists.
func CityRuler(world *ecs.World, cityID uint32) (ecs.Entity, uint64, bool) {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](world)

	var rulerEnt ecs.Entity
	var rulerIdentityID uint64
	found := false

	// Iterate the query to completion (no early break) so the world lock is released.
	query := world.Query(filter.All(npcID, identID, affilID, adminMarkerID))
	for query.Next() {
		if found {
			continue
		}
		affil := (*components.Affiliation)(query.Get(affilID))
		if affil.CityID != cityID {
			continue
		}
		ident := (*components.Identity)(query.Get(identID))
		rulerEnt = query.Entity()
		rulerIdentityID = ident.ID
		found = true
	}

	return rulerEnt, rulerIdentityID, found
}

// positiveIncomingScore sums all positive incoming hooks for an identity — the same
// influence formula used by LeadershipEmergenceSystem for organic elections.
func positiveIncomingScore(hooks *engine.SparseHookGraph, identityID uint64) int {
	if hooks == nil {
		return 0
	}
	score := 0
	for _, val := range hooks.GetAllIncomingHooks(identityID) {
		if val > 0 {
			score += val
		}
	}
	return score
}

// CanClaimLeadership reports whether the player entity may seize rulership of its city.
// The player must belong to a city (CityID != 0) and either the city has no current
// ruler, or the player's influence score strictly exceeds the ruler's score plus the
// incumbency bias of 1 (mirroring the anti-flip bias in leadership_emergence.go).
func CanClaimLeadership(world *ecs.World, hooks *engine.SparseHookGraph, player ecs.Entity) bool {
	affilID := ecs.ComponentID[components.Affiliation](world)
	identID := ecs.ComponentID[components.Identity](world)

	if !world.Alive(player) || !world.Has(player, affilID) {
		return false
	}
	affil := (*components.Affiliation)(world.Get(player, affilID))
	if affil.CityID == 0 {
		return false // Homeless / wandering entities cannot claim anything.
	}

	rulerEnt, rulerIdentityID, hasRuler := CityRuler(world, affil.CityID)
	if !hasRuler {
		return true // Empty throne: any citizen of the city may take it.
	}
	if rulerEnt == player {
		return false // Already ruling this city; nothing to claim.
	}

	playerScore := 0
	if world.Has(player, identID) {
		playerIdent := (*components.Identity)(world.Get(player, identID))
		playerScore = positiveIncomingScore(hooks, playerIdent.ID)
	}
	rulerScore := positiveIncomingScore(hooks, rulerIdentityID)

	return playerScore > rulerScore+1
}

// ClaimLeadership executes a leadership claim: it removes the AdministrationMarker from
// the displaced ruler (if any), crowns the player with the marker, and seeds a
// LegitimacyComponent{Score: 50} on the player if one is missing (an existing score is
// preserved). Returns ErrClaimDenied when CanClaimLeadership rejects the claim.
// All structural mutations happen outside query loops.
func ClaimLeadership(world *ecs.World, hooks *engine.SparseHookGraph, player ecs.Entity) error {
	if !CanClaimLeadership(world, hooks, player) {
		return ErrClaimDenied
	}

	affilID := ecs.ComponentID[components.Affiliation](world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)

	affil := (*components.Affiliation)(world.Get(player, affilID))

	// Locate the displaced ruler before mutating. CityRuler runs a fully-consumed query,
	// so the world is unlocked by the time we apply structural changes below.
	oldRuler, _, hadRuler := CityRuler(world, affil.CityID)

	if hadRuler && oldRuler != player && world.Has(oldRuler, adminMarkerID) {
		world.Remove(oldRuler, adminMarkerID)
	}

	if !world.Has(player, adminMarkerID) {
		world.Add(player, adminMarkerID)
	}

	// Phase 35.1 bridge: a fresh ruler starts with a baseline legitimacy of 50.
	if !world.Has(player, legitID) {
		world.Add(player, legitID)
		legit := (*components.LegitimacyComponent)(world.Get(player, legitID))
		legit.Score = 50
	}

	return nil
}
