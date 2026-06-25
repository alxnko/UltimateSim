package systems

import (
	"fmt"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase (Playable Game Shell): ruler law actions.
// Pure helpers and world mutations backing the player's law/edict UI. A ruling
// player toggles jurisdiction laws, contraband flags, currency debasement and
// declares wars on the capital it controls. All world mutations happen outside
// open query loops per DOD rules.

// MaxDebasement caps currency debasement so a country can never fully zero out
// its own coinage (clamp range is 0..MaxDebasement).
const MaxDebasement float32 = 0.9

// IsIllegal reports whether the given interaction type (e.g.
// components.InteractionTheft) is flagged as a crime in the jurisdiction's
// IllegalActionIDs bitmask.
func IsIllegal(jur *components.JurisdictionComponent, interaction uint8) bool {
	if jur == nil {
		return false
	}
	return jur.IllegalActionIDs&(uint32(1)<<interaction) != 0
}

// ToggleLaw flips the legality bit for the given interaction type in the
// jurisdiction's IllegalActionIDs bitmask.
func ToggleLaw(jur *components.JurisdictionComponent, interaction uint8) {
	if jur == nil {
		return
	}
	jur.IllegalActionIDs ^= uint32(1) << interaction
}

// HasContraband reports whether the given item ID (e.g. components.ItemIron)
// is flagged as illegal in the contraband bitmask.
func HasContraband(c *components.ContrabandComponent, item uint8) bool {
	if c == nil {
		return false
	}
	return c.Contraband&(uint32(1)<<item) != 0
}

// ToggleContraband flips the contraband bit for the given item ID.
func ToggleContraband(c *components.ContrabandComponent, item uint8) {
	if c == nil {
		return
	}
	c.Contraband ^= uint32(1) << item
}

// SetDebasement applies a debasement delta to a country's standard currency,
// clamping the result to the 0..MaxDebasement range.
func SetDebasement(c *components.CountryComponent, delta float32) {
	if c == nil {
		return
	}
	c.Debasement += delta
	if c.Debasement < 0 {
		c.Debasement = 0
	}
	if c.Debasement > MaxDebasement {
		c.Debasement = MaxDebasement
	}
}

// FindCapitalOf locates the capital village governing the given country:
// an entity carrying CapitalComponent and CountryComponent whose
// Affiliation.CountryID matches countryID. Returns false if no such capital
// exists (e.g. the country collapsed).
func FindCapitalOf(world *ecs.World, countryID uint32) (ecs.Entity, bool) {
	capID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	filter := ecs.All(capID, countryCompID, affID)
	query := world.Query(&filter)
	for query.Next() {
		aff := (*components.Affiliation)(query.Get(affID))
		if aff.CountryID == countryID {
			entity := query.Entity()
			query.Close()
			return entity, true
		}
	}
	return ecs.Entity{}, false
}

// ListCountries returns the distinct CountryIDs of all living capitals,
// sorted ascending for deterministic UI ordering. CountryID 0 (unaffiliated)
// is skipped because it does not denote a real country.
func ListCountries(world *ecs.World) []uint32 {
	capID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	seen := make(map[uint32]struct{})
	countries := make([]uint32, 0, 8)

	filter := ecs.All(capID, countryCompID, affID)
	query := world.Query(&filter)
	for query.Next() {
		aff := (*components.Affiliation)(query.Get(affID))
		if aff.CountryID == 0 {
			continue
		}
		if _, ok := seen[aff.CountryID]; ok {
			continue
		}
		seen[aff.CountryID] = struct{}{}
		countries = append(countries, aff.CountryID)
	}

	slices.Sort(countries)
	return countries
}

// DeclareWar adds (or re-targets) a WarTrackerComponent on the given capital,
// marking an active war against targetCountryID. The component is added
// outside any open query loop. Returns an error if the target is invalid:
// CountryID 0 or the capital's own country.
func DeclareWar(world *ecs.World, capital ecs.Entity, targetCountryID uint32) error {
	if targetCountryID == 0 {
		return fmt.Errorf("cannot declare war on country 0 (no country)")
	}
	if !world.Alive(capital) {
		return fmt.Errorf("cannot declare war: capital entity is dead")
	}

	affID := ecs.ComponentID[components.Affiliation](world)
	if !world.Has(capital, affID) {
		return fmt.Errorf("cannot declare war: capital has no Affiliation")
	}
	aff := (*components.Affiliation)(world.Get(capital, affID))
	if aff.CountryID == targetCountryID {
		return fmt.Errorf("country %d cannot declare war on itself", targetCountryID)
	}

	warCompID := ecs.ComponentID[components.WarTrackerComponent](world)
	if !world.Has(capital, warCompID) {
		world.Add(capital, warCompID)
	}
	warComp := (*components.WarTrackerComponent)(world.Get(capital, warCompID))
	warComp.TargetCountryID = targetCountryID
	warComp.Active = true
	return nil
}

// MakePeace deactivates the capital's war tracker if one is present. The
// component is kept (Active=false) so downstream systems see the war ended
// rather than a missing component archetype shift.
func MakePeace(world *ecs.World, capital ecs.Entity) {
	if !world.Alive(capital) {
		return
	}
	warCompID := ecs.ComponentID[components.WarTrackerComponent](world)
	if !world.Has(capital, warCompID) {
		return
	}
	warComp := (*components.WarTrackerComponent)(world.Get(capital, warCompID))
	warComp.Active = false
}
