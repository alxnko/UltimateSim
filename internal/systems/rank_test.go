package systems

import (
	"errors"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: Rank & Leadership Claims E2E Tests
// Deterministic, headless tests covering rank tier resolution, leadership claim
// transfer with legitimacy seeding, and claim denial against a stronger ruler.

// spawnRankNPC creates an NPC+Identity+Affiliation entity for rank tests.
func spawnRankNPC(world *ecs.World, identityID uint64, cityID uint32, countryID uint32) ecs.Entity {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)

	e := world.NewEntity(npcID, identID, affilID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = identityID
	affil := (*components.Affiliation)(world.Get(e, affilID))
	affil.CityID = cityID
	affil.CountryID = countryID
	return e
}

// TestRankTiers verifies Citizen -> Ruler -> Sovereign resolution via the capital village.
func TestRankTiers(t *testing.T) {
	world := ecs.NewWorld()

	affilID := ecs.ComponentID[components.Affiliation](&world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)

	// RankName mapping
	if RankName(RankCitizen) != "Citizen" {
		t.Errorf("RankName(RankCitizen) = %q, want Citizen", RankName(RankCitizen))
	}
	if RankName(RankRuler) != "Ruler" {
		t.Errorf("RankName(RankRuler) = %q, want Ruler", RankName(RankRuler))
	}
	if RankName(RankSovereign) != "Sovereign" {
		t.Errorf("RankName(RankSovereign) = %q, want Sovereign", RankName(RankSovereign))
	}

	// 1. Homeless NPC (CityID == 0) is a Citizen even with the AdministrationMarker.
	wanderer := spawnRankNPC(&world, 1, 0, 0)
	world.Add(wanderer, adminMarkerID)
	if rank, _, _ := GetRank(&world, wanderer); rank != RankCitizen {
		t.Errorf("homeless marker-holder rank = %d, want RankCitizen", rank)
	}

	// 2. Plain settled NPC without the marker is a Citizen; IDs flow through regardless.
	npc := spawnRankNPC(&world, 2, 5, 77)
	rank, cityID, countryID := GetRank(&world, npc)
	if rank != RankCitizen || cityID != 5 || countryID != 77 {
		t.Errorf("settled citizen = (%d, %d, %d), want (RankCitizen, 5, 77)", rank, cityID, countryID)
	}

	// 3. Marker + city = Ruler (its village is not a capital yet).
	world.Add(npc, adminMarkerID)
	village := world.NewEntity(villageID, affilID)
	vAffil := (*components.Affiliation)(world.Get(village, affilID))
	vAffil.CityID = 5
	vAffil.CountryID = 77

	rank, cityID, countryID = GetRank(&world, npc)
	if rank != RankRuler || cityID != 5 || countryID != 77 {
		t.Errorf("ruler of plain village = (%d, %d, %d), want (RankRuler, 5, 77)", rank, cityID, countryID)
	}

	// 4. Promote the village to a Capital: the ruler becomes a Sovereign.
	world.Add(village, capitalID)
	rank, cityID, countryID = GetRank(&world, npc)
	if rank != RankSovereign || cityID != 5 || countryID != 77 {
		t.Errorf("ruler of capital village = (%d, %d, %d), want (RankSovereign, 5, 77)", rank, cityID, countryID)
	}

	// 5. A capital village in ANOTHER city must not promote this ruler.
	otherRuler := spawnRankNPC(&world, 3, 6, 77)
	world.Add(otherRuler, adminMarkerID)
	if rank, _, _ := GetRank(&world, otherRuler); rank != RankRuler {
		t.Errorf("ruler of non-capital city rank = %d, want RankRuler", rank)
	}
}

// TestRankClaimLeadershipTransfer verifies claim transfer from a weaker ruler,
// legitimacy seeding, and claiming an empty throne.
func TestRankClaimLeadershipTransfer(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)

	// City 1: a weak sitting ruler (score 5) and a popular player (score 50).
	oldRuler := spawnRankNPC(&world, 200, 1, 0)
	world.Add(oldRuler, adminMarkerID)
	player := spawnRankNPC(&world, 100, 1, 0)

	hooks.AddHook(900, 200, 5)    // ruler influence
	hooks.AddHook(901, 100, 50)   // player influence
	hooks.AddHook(902, 100, -100) // negative hooks must NOT count toward the score

	// Sanity: CityRuler resolves the sitting ruler.
	rulerEnt, rulerIdentityID, found := CityRuler(&world, 1)
	if !found || rulerEnt != oldRuler || rulerIdentityID != 200 {
		t.Fatalf("CityRuler(1) = (%v, %d, %v), want (oldRuler, 200, true)", rulerEnt, rulerIdentityID, found)
	}

	if !CanClaimLeadership(&world, hooks, player) {
		t.Fatalf("player (score 50) should be able to claim against ruler (score 5)")
	}

	if err := ClaimLeadership(&world, hooks, player); err != nil {
		t.Fatalf("ClaimLeadership returned error: %v", err)
	}

	// Old ruler dethroned, player crowned.
	if world.Has(oldRuler, adminMarkerID) {
		t.Errorf("old ruler should have lost the AdministrationMarker")
	}
	if !world.Has(player, adminMarkerID) {
		t.Errorf("player should have gained the AdministrationMarker")
	}

	// Legitimacy seeded at 50.
	if !world.Has(player, legitID) {
		t.Fatalf("player should have a LegitimacyComponent after claiming")
	}
	legit := (*components.LegitimacyComponent)(world.Get(player, legitID))
	if legit.Score != 50 {
		t.Errorf("seeded legitimacy = %d, want 50", legit.Score)
	}

	// CityRuler now resolves the player; GetRank reports Ruler.
	rulerEnt, rulerIdentityID, found = CityRuler(&world, 1)
	if !found || rulerEnt != player || rulerIdentityID != 100 {
		t.Errorf("CityRuler(1) after claim = (%v, %d, %v), want (player, 100, true)", rulerEnt, rulerIdentityID, found)
	}
	if rank, cityID, _ := GetRank(&world, player); rank != RankRuler || cityID != 1 {
		t.Errorf("player rank after claim = (%d, %d), want (RankRuler, 1)", rank, cityID)
	}

	// Re-claiming an already-held throne is denied.
	if CanClaimLeadership(&world, hooks, player) {
		t.Errorf("sitting ruler should not be able to re-claim its own throne")
	}

	// Empty throne: a zero-score citizen of a rulerless city may claim it. A pre-existing
	// legitimacy score must be preserved (seeding only happens when missing).
	player2 := spawnRankNPC(&world, 300, 2, 0)
	world.Add(player2, legitID)
	preLegit := (*components.LegitimacyComponent)(world.Get(player2, legitID))
	preLegit.Score = 80

	if !CanClaimLeadership(&world, hooks, player2) {
		t.Fatalf("citizen should be able to claim an empty throne")
	}
	if err := ClaimLeadership(&world, hooks, player2); err != nil {
		t.Fatalf("ClaimLeadership on empty throne returned error: %v", err)
	}
	if !world.Has(player2, adminMarkerID) {
		t.Errorf("player2 should have gained the AdministrationMarker")
	}
	postLegit := (*components.LegitimacyComponent)(world.Get(player2, legitID))
	if postLegit.Score != 80 {
		t.Errorf("pre-existing legitimacy = %d after claim, want 80 (must not be reseeded)", postLegit.Score)
	}
}

// TestRankClaimLeadershipDenied verifies denial against a stronger ruler, the
// strict incumbency-bias boundary, and denial for homeless claimants.
func TestRankClaimLeadershipDenied(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)

	// City 3: a strong ruler (score 100) and a weaker player (score 50).
	ruler := spawnRankNPC(&world, 400, 3, 0)
	world.Add(ruler, adminMarkerID)
	player := spawnRankNPC(&world, 500, 3, 0)

	hooks.AddHook(900, 400, 100)
	hooks.AddHook(901, 500, 50)

	if CanClaimLeadership(&world, hooks, player) {
		t.Errorf("player (score 50) should NOT be able to claim against ruler (score 100)")
	}
	if err := ClaimLeadership(&world, hooks, player); !errors.Is(err, ErrClaimDenied) {
		t.Errorf("ClaimLeadership error = %v, want ErrClaimDenied", err)
	}

	// The throne is untouched and no legitimacy was seeded on the failed claimant.
	if !world.Has(ruler, adminMarkerID) {
		t.Errorf("ruler should retain the AdministrationMarker after a denied claim")
	}
	if world.Has(player, adminMarkerID) {
		t.Errorf("player should not have the AdministrationMarker after a denied claim")
	}
	if world.Has(player, legitID) {
		t.Errorf("player should not have a LegitimacyComponent after a denied claim")
	}

	// Boundary: score must STRICTLY exceed rulerScore+1 (incumbency bias).
	hooks.AddHook(902, 500, 51) // player total = 101 = rulerScore+1 -> still denied
	if CanClaimLeadership(&world, hooks, player) {
		t.Errorf("player at exactly rulerScore+1 (101) should still be denied")
	}
	hooks.AddHook(903, 500, 1) // player total = 102 > 101 -> allowed
	if !CanClaimLeadership(&world, hooks, player) {
		t.Errorf("player at rulerScore+2 (102) should be able to claim")
	}

	// Homeless claimant (CityID == 0) is always denied.
	wanderer := spawnRankNPC(&world, 600, 0, 0)
	if CanClaimLeadership(&world, hooks, wanderer) {
		t.Errorf("homeless entity should never be able to claim leadership")
	}
	if err := ClaimLeadership(&world, hooks, wanderer); !errors.Is(err, ErrClaimDenied) {
		t.Errorf("homeless ClaimLeadership error = %v, want ErrClaimDenied", err)
	}
}
