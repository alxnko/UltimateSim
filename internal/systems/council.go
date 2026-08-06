package systems

import (
	"errors"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Grand Strategy Phase (spec P2.5): the sovereign's council.
// A CouncilComponent hangs on the country capital entity (the same Village
// carrying CapitalComponent + CountryComponent that taxation and diplomacy use
// for country state). Each filled seat yields a periodic national bonus;
// GetCouncilBonus is the read API other systems (diplomacy, economy, plots)
// consume.

const (
	// CouncilTickRate throttles CouncilSystem to every 400 ticks.
	CouncilTickRate = 400

	// StewardIncomePct: treasury grows by this percentage per council cycle.
	StewardIncomePct = 5
	// MarshalWarScoreBonus: flat war-score edge for diplomacy/war resolution.
	MarshalWarScoreBonus = 10
	// DiplomatOpinionDrift: passive opinion gain per diplomacy cycle.
	DiplomatOpinionDrift = 1
	// SpymasterDiscoveryBonus: added plot-discovery chance (percent points).
	SpymasterDiscoveryBonus = 25
)

var (
	ErrNoCapital         = errors.New("appointment failed: country has no capital")
	ErrInvalidSeat       = errors.New("appointment failed: unknown council seat")
	ErrForeignCouncilor  = errors.New("appointment failed: NPC does not belong to this country")
	ErrCouncilorNotFound = errors.New("appointment failed: no living NPC with that identity")
)

// councilSeat returns a pointer to the seat field for the given seat constant.
func councilSeat(c *components.CouncilComponent, seat uint8) *uint64 {
	switch seat {
	case components.SeatSteward:
		return &c.Steward
	case components.SeatMarshal:
		return &c.Marshal
	case components.SeatDiplomat:
		return &c.Diplomat
	case components.SeatSpymaster:
		return &c.Spymaster
	default:
		return nil
	}
}

// Appoint seats the NPC with Identity.ID npcID on the given council seat of
// the country. The NPC must be a living citizen of the country
// (Affiliation.CountryID matches). Holding a second seat is not allowed: any
// prior seat of the same NPC on this council is vacated. Sovereign gating is
// the caller's responsibility (UI checks GetRank). Must be called outside any
// open query loop (structural mutation).
func Appoint(world *ecs.World, country uint32, seat uint8, npcID uint64) error {
	if seat < components.SeatSteward || seat > components.SeatSpymaster {
		return ErrInvalidSeat
	}

	capital, ok := FindCapitalOf(world, country)
	if !ok {
		return ErrNoCapital
	}

	// Validate citizenship: scan NPCs to completion (keeps the world unlocked).
	npcTagID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	found := false
	sameCountry := false
	q := world.Query(ecs.All(npcTagID, identID, affID))
	for q.Next() {
		if found {
			continue
		}
		ident := (*components.Identity)(q.Get(identID))
		if ident.ID != npcID {
			continue
		}
		found = true
		aff := (*components.Affiliation)(q.Get(affID))
		sameCountry = aff.CountryID == country
	}
	if !found {
		return ErrCouncilorNotFound
	}
	if !sameCountry {
		return ErrForeignCouncilor
	}

	councilID := ecs.ComponentID[components.CouncilComponent](world)
	if !world.Has(capital, councilID) {
		world.Add(capital, councilID)
	}
	council := (*components.CouncilComponent)(world.Get(capital, councilID))

	// One NPC, one seat: vacate any seat the appointee already holds.
	for s := components.SeatSteward; s <= components.SeatSpymaster; s++ {
		if p := councilSeat(council, s); p != nil && *p == npcID {
			*p = 0
		}
	}
	*councilSeat(council, seat) = npcID
	return nil
}

// GetCouncilBonus reports the national bonus granted by the given seat of the
// country's council: StewardIncomePct, MarshalWarScoreBonus,
// DiplomatOpinionDrift, or SpymasterDiscoveryBonus when the seat is filled,
// and 0 when vacant or the country has no capital/council. Read-only; safe to
// call from any system outside its own open query loops.
func GetCouncilBonus(world *ecs.World, country uint32, seat uint8) int {
	capital, ok := FindCapitalOf(world, country)
	if !ok {
		return 0
	}
	councilID := ecs.ComponentID[components.CouncilComponent](world)
	if !world.Has(capital, councilID) {
		return 0
	}
	council := (*components.CouncilComponent)(world.Get(capital, councilID))
	p := councilSeat(council, seat)
	if p == nil || *p == 0 {
		return 0
	}
	switch seat {
	case components.SeatSteward:
		return StewardIncomePct
	case components.SeatMarshal:
		return MarshalWarScoreBonus
	case components.SeatDiplomat:
		return DiplomatOpinionDrift
	case components.SeatSpymaster:
		return SpymasterDiscoveryBonus
	}
	return 0
}

// CouncilSystem validates council seats and applies the Steward's periodic
// treasury income bonus every CouncilTickRate ticks. The Marshal, Diplomat,
// and Spymaster bonuses are passive: diplomacy, economy, and plot systems
// read them through GetCouncilBonus.
type CouncilSystem struct {
	tick uint64
}

func NewCouncilSystem(world *ecs.World) *CouncilSystem {
	// Pre-register component IDs used by Update.
	ecs.ComponentID[components.CouncilComponent](world)
	ecs.ComponentID[components.TreasuryComponent](world)
	return &CouncilSystem{}
}

// IsExpensive throttles the system during fast-forward.
func (s *CouncilSystem) IsExpensive() bool { return true }

func (s *CouncilSystem) Update(world *ecs.World) {
	s.tick++
	if s.tick%CouncilTickRate != 0 {
		return
	}

	npcTagID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	councilID := ecs.ComponentID[components.CouncilComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	// Flat cache of living NPC countries (avoids nested queries below).
	npcCountry := make(map[uint64]uint32)
	nq := world.Query(ecs.All(npcTagID, identID, affID))
	for nq.Next() {
		ident := (*components.Identity)(nq.Get(identID))
		aff := (*components.Affiliation)(nq.Get(affID))
		npcCountry[ident.ID] = aff.CountryID
	}

	cq := world.Query(filter.All(capitalID, countryCompID, affID, councilID))
	for cq.Next() {
		aff := (*components.Affiliation)(cq.Get(affID))
		council := (*components.CouncilComponent)(cq.Get(councilID))

		// Vacate seats whose holder died or left the country.
		for seat := components.SeatSteward; seat <= components.SeatSpymaster; seat++ {
			p := councilSeat(council, seat)
			if *p == 0 {
				continue
			}
			holderCountry, alive := npcCountry[*p]
			if !alive || holderCountry != aff.CountryID {
				*p = 0
			}
		}

		// Steward: treasury income bonus (+StewardIncomePct% per cycle).
		if council.Steward != 0 && cq.Has(treasuryID) {
			treasury := (*components.TreasuryComponent)(cq.Get(treasuryID))
			if treasury.Wealth > 0 {
				treasury.Wealth += treasury.Wealth * StewardIncomePct / 100.0
			}
		}
	}
}
