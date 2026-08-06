package systems

import (
	"fmt"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase P2.2: Diplomacy
// DiplomacySystem maintains the country-to-country diplomatic ledger stored in
// DiplomacyComponent on each capital (the country-representing entity, see
// law_actions.go). Every pulse it drifts opinions toward 0, applies border
// friction and war weariness, syncs relation state with the existing macro war
// model (WarTrackerComponent via DeclareWar/MakePeace so both models never
// diverge), forms alliances at high opinion, declares AI wars at low opinion,
// accumulates deterministic WarScore from relative treasury+population
// strength, and settles wars at |WarScore| >= WarScoreCap with a truce and a
// tribute payment.
//
// Determinism: no RNG, countries processed in sorted CountryID order, integer
// strength arithmetic only. Register under engine.PhaseResolution.

// Diplomacy tuning constants (exported so the UI can surface the rules).
const (
	DiplomacyPulseTicks uint64 = 300 // System evaluates once per pulse

	OpinionMin int16 = -100
	OpinionMax int16 = 100

	OpinionDriftStep   int16 = 1 // Per-pulse decay of opinion toward 0
	WarWearinessStep   int16 = 2 // Per-pulse opinion drop while at war
	BorderFrictionStep int16 = 2 // Per-pulse opinion drop between non-allied neighbors

	// BorderRadiusSq matches the geopolitical distance threshold used by
	// ResourceWarSystem (50 tiles squared): capitals closer than this are
	// considered bordering countries.
	BorderRadiusSq float32 = 2500.0

	WarDeclareOpinion int16 = -60 // AI declares war below this opinion
	AllianceOpinion   int16 = 60  // AI alliance offers auto-accept above this opinion
	PostWarOpinion    int16 = -30 // Opinion both sides settle at after a peace

	WarScoreCap       int64 = 100 // |WarScore| that forces a settlement
	WarScoreStepScale int64 = 25  // Max per-pulse WarScore swing at total dominance

	// PopulationStrengthWeight converts one citizen into strength points; the
	// other strength component is raw treasury wealth.
	PopulationStrengthWeight int64 = 10

	TruceDurationTicks uint64  = 2000 // Peace treaty length after a settled war
	TributeFraction    float32 = 0.25 // Fraction of the loser's treasury paid to the winner

	ImproveRelationsCost        float32 = 50.0 // Treasury gold burned per ImproveRelations
	ImproveRelationsOpinionGain int16   = 10
	ImproveRelationsCooldown    uint64  = 600 // Ticks between ImproveRelations uses per target

	AllianceOfferMinOpinion     int16 = 40 // Player alliance offers accepted at/above this opinion
	BreakAllianceOpinionPenalty int16 = 20
)

// diploCountryData is the flat per-pulse snapshot of one country's capital.
type diploCountryData struct {
	entity        ecs.Entity
	countryID     uint32
	x, y          float32
	hasPos        bool
	strength      int64
	trackerActive bool
	trackerTarget uint32
	hasDiplomacy  bool
}

// DiplomacySystem runs the pulse described in the package comment.
type DiplomacySystem struct {
	tickCounter uint64

	// Component IDs
	capID         ecs.ID
	countryCompID ecs.ID
	affID         ecs.ID
	posID         ecs.ID
	treasID       ecs.ID
	warCompID     ecs.ID
	npcID         ecs.ID
	diploID       ecs.ID
}

// NewDiplomacySystem creates a DiplomacySystem bound to the world's component IDs.
func NewDiplomacySystem(world *ecs.World) *DiplomacySystem {
	return &DiplomacySystem{
		capID:         ecs.ComponentID[components.CapitalComponent](world),
		countryCompID: ecs.ComponentID[components.CountryComponent](world),
		affID:         ecs.ComponentID[components.Affiliation](world),
		posID:         ecs.ComponentID[components.Position](world),
		treasID:       ecs.ComponentID[components.TreasuryComponent](world),
		warCompID:     ecs.ComponentID[components.WarTrackerComponent](world),
		npcID:         ecs.ComponentID[components.NPC](world),
		diploID:       ecs.ComponentID[components.DiplomacyComponent](world),
	}
}

// Update evaluates macro diplomacy once every DiplomacyPulseTicks ticks.
func (s *DiplomacySystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%DiplomacyPulseTicks != 0 {
		return
	}
	tick := s.tickCounter

	// 1. Snapshot capitals (query fully consumed; no mutations inside).
	population := make(map[uint32]int64)
	npcFilter := ecs.All(s.npcID, s.affID)
	npcQuery := world.Query(&npcFilter)
	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		if aff.CountryID != 0 {
			population[aff.CountryID]++
		}
	}

	seen := make(map[uint32]struct{})
	var countries []diploCountryData
	capFilter := ecs.All(s.capID, s.countryCompID, s.affID)
	capQuery := world.Query(&capFilter)
	for capQuery.Next() {
		aff := (*components.Affiliation)(capQuery.Get(s.affID))
		if aff.CountryID == 0 {
			continue
		}
		if _, dup := seen[aff.CountryID]; dup {
			continue // One capital represents a country; extras are ignored.
		}
		seen[aff.CountryID] = struct{}{}

		data := diploCountryData{
			entity:    capQuery.Entity(),
			countryID: aff.CountryID,
			strength:  PopulationStrengthWeight * population[aff.CountryID],
		}
		if capQuery.Has(s.posID) {
			pos := (*components.Position)(capQuery.Get(s.posID))
			data.x, data.y = pos.X, pos.Y
			data.hasPos = true
		}
		if capQuery.Has(s.treasID) {
			treas := (*components.TreasuryComponent)(capQuery.Get(s.treasID))
			if treas.Wealth > 0 {
				data.strength += int64(treas.Wealth)
			}
		}
		if capQuery.Has(s.warCompID) {
			war := (*components.WarTrackerComponent)(capQuery.Get(s.warCompID))
			data.trackerActive = war.Active
			data.trackerTarget = war.TargetCountryID
		}
		data.hasDiplomacy = capQuery.Has(s.diploID)
		countries = append(countries, data)
	}

	if len(countries) < 2 {
		return
	}
	slices.SortFunc(countries, func(a, b diploCountryData) int {
		return int(int64(a.countryID) - int64(b.countryID))
	})

	// 2. Deferred structural fix-up: every capital carries a DiplomacyComponent.
	for i := range countries {
		if !countries[i].hasDiplomacy {
			world.Add(countries[i].entity, s.diploID)
			countries[i].hasDiplomacy = true
		}
	}

	// 3. Ensure every pair relation exists BEFORE caching row pointers: later
	// appends would reallocate Relations and orphan pointers taken earlier.
	diplos := make([]*components.DiplomacyComponent, len(countries))
	for i := range countries {
		diplos[i] = (*components.DiplomacyComponent)(world.Get(countries[i].entity, s.diploID))
	}
	for i := 0; i < len(countries); i++ {
		for j := i + 1; j < len(countries); j++ {
			ensureRelation(diplos[i], countries[j].countryID)
			ensureRelation(diplos[j], countries[i].countryID)
		}
	}

	// 4. Value-only pass over sorted pairs. Structural war declarations are
	// collected and applied afterwards (step 5) so no cached pointer is ever
	// read after an archetype move.
	type warDecl struct {
		attacker      ecs.Entity
		targetCountry uint32
	}
	var declarations []warDecl

	for i := 0; i < len(countries); i++ {
		for j := i + 1; j < len(countries); j++ {
			a, b := &countries[i], &countries[j]
			ra := relationRow(diplos[i], b.countryID)
			rb := relationRow(diplos[j], a.countryID)

			// Macro parity sync: WarTrackerComponent is the source of truth
			// for whether a war exists (resource wars, the ruler law UI and
			// this system all funnel through it).
			atWarMacro := (a.trackerActive && a.trackerTarget == b.countryID) ||
				(b.trackerActive && b.trackerTarget == a.countryID)
			if atWarMacro && !ra.AtWar {
				startWarRows(ra, rb)
			} else if !atWarMacro && ra.AtWar {
				// The macro war ended elsewhere (e.g. WarEconomySystem state
				// default): settle as a white peace with a truce.
				endWarRows(ra, rb, tick)
			}

			if ra.AtWar {
				ra.Opinion = clampOpinion(ra.Opinion - WarWearinessStep)
				rb.Opinion = ra.Opinion

				// Deterministic WarScore from relative strength.
				step := (a.strength - b.strength) * WarScoreStepScale /
					(a.strength + b.strength + 1)
				scoreA := int64(ra.WarScore) + step
				if scoreA >= WarScoreCap || scoreA <= -WarScoreCap {
					winIdx, loseIdx := i, j
					winRel, loseRel := ra, rb
					if scoreA < 0 {
						winIdx, loseIdx = j, i
						winRel, loseRel = rb, ra
					}
					s.settleWar(world,
						countries[winIdx].entity, countries[loseIdx].entity,
						countries[winIdx].countryID, countries[loseIdx].countryID,
						winRel, loseRel, tick)
					continue
				}
				ra.WarScore = int16(scoreA)
				rb.WarScore = int16(-scoreA)
				continue
			}

			// Peacetime: drift toward 0, then border friction.
			ra.Opinion = driftTowardZero(ra.Opinion)
			if !ra.Alliance && a.hasPos && b.hasPos {
				dx, dy := a.x-b.x, a.y-b.y
				if dx*dx+dy*dy <= BorderRadiusSq {
					ra.Opinion = clampOpinion(ra.Opinion - BorderFrictionStep)
				}
			}
			rb.Opinion = ra.Opinion

			// AI alliance: mutual admiration auto-accepts symmetrically.
			if ra.Opinion > AllianceOpinion {
				ra.Alliance = true
				rb.Alliance = true
			}

			// AI war: hatred plus no truce. The attacker is the first pair
			// member with a free war tracker (one macro war per country,
			// mirroring ResourceWarSystem); if both are busy no war starts.
			if ra.Opinion < WarDeclareOpinion && tick >= ra.TruceUntil && !ra.Alliance {
				var attacker ecs.Entity
				var target uint32
				switch {
				case !a.trackerActive:
					attacker, target = a.entity, b.countryID
					a.trackerActive, a.trackerTarget = true, b.countryID
				case !b.trackerActive:
					attacker, target = b.entity, a.countryID
					b.trackerActive, b.trackerTarget = true, a.countryID
				default:
					continue
				}
				startWarRows(ra, rb)
				declarations = append(declarations, warDecl{attacker: attacker, targetCountry: target})
			}
		}
	}

	// 5. Apply structural war declarations after all pointer work is done.
	for _, d := range declarations {
		_ = DeclareWar(world, d.attacker, d.targetCountry)
	}
}

// settleWar ends a war pair: rows are reset with a truce and PostWarOpinion,
// the loser pays TributeFraction of its treasury to the winner, and any active
// macro tracker aimed at the other side is deactivated. Only value mutations
// (no structural changes), so cached component pointers stay valid.
func (s *DiplomacySystem) settleWar(world *ecs.World, winnerCap, loserCap ecs.Entity,
	winnerCountry, loserCountry uint32,
	winRel, loseRel *components.CountryRelation, tick uint64) {

	endWarRows(winRel, loseRel, tick)

	if world.Has(winnerCap, s.treasID) && world.Has(loserCap, s.treasID) {
		loserT := (*components.TreasuryComponent)(world.Get(loserCap, s.treasID))
		winnerT := (*components.TreasuryComponent)(world.Get(winnerCap, s.treasID))
		if tribute := loserT.Wealth * TributeFraction; tribute > 0 {
			loserT.Wealth -= tribute
			winnerT.Wealth += tribute
		}
	}

	s.deactivateTracker(world, winnerCap, loserCountry)
	s.deactivateTracker(world, loserCap, winnerCountry)
}

// deactivateTracker flips a capital's war tracker off only when it targets the
// given country, so an unrelated third-front war is never ended by accident.
func (s *DiplomacySystem) deactivateTracker(world *ecs.World, capital ecs.Entity, targetCountry uint32) {
	if !world.Alive(capital) || !world.Has(capital, s.warCompID) {
		return
	}
	war := (*components.WarTrackerComponent)(world.Get(capital, s.warCompID))
	if war.Active && war.TargetCountryID == targetCountry {
		war.Active = false
	}
}

// --- Relation row helpers (pure value logic) ---

func clampOpinion(v int16) int16 {
	if v < OpinionMin {
		return OpinionMin
	}
	if v > OpinionMax {
		return OpinionMax
	}
	return v
}

func driftTowardZero(v int16) int16 {
	switch {
	case v > 0:
		if v < OpinionDriftStep {
			return 0
		}
		return v - OpinionDriftStep
	case v < 0:
		if v > -OpinionDriftStep {
			return 0
		}
		return v + OpinionDriftStep
	default:
		return 0
	}
}

// startWarRows marks both symmetric rows as a fresh war.
func startWarRows(ra, rb *components.CountryRelation) {
	ra.AtWar, rb.AtWar = true, true
	ra.Alliance, rb.Alliance = false, false
	ra.WarScore, rb.WarScore = 0, 0
	if ra.Opinion > WarDeclareOpinion {
		ra.Opinion = WarDeclareOpinion
	}
	rb.Opinion = ra.Opinion
}

// endWarRows resets both symmetric rows to an armistice with a fresh truce.
func endWarRows(ra, rb *components.CountryRelation, tick uint64) {
	ra.AtWar, rb.AtWar = false, false
	ra.WarScore, rb.WarScore = 0, 0
	ra.TruceUntil, rb.TruceUntil = tick+TruceDurationTicks, tick+TruceDurationTicks
	ra.Opinion, rb.Opinion = PostWarOpinion, PostWarOpinion
}

// ensureRelation appends a zero row for targetCountry if none exists.
func ensureRelation(d *components.DiplomacyComponent, targetCountry uint32) {
	for i := range d.Relations {
		if d.Relations[i].TargetCountry == targetCountry {
			return
		}
	}
	d.Relations = append(d.Relations, components.CountryRelation{TargetCountry: targetCountry})
}

// relationRow returns the row for targetCountry; ensureRelation must have run.
func relationRow(d *components.DiplomacyComponent, targetCountry uint32) *components.CountryRelation {
	for i := range d.Relations {
		if d.Relations[i].TargetCountry == targetCountry {
			return &d.Relations[i]
		}
	}
	return nil
}

// --- Player diplomacy actions (called by the UI outside any query loop) ---

// prepareRelation validates both countries, locates their capitals, and makes
// sure both capitals carry a DiplomacyComponent with the symmetric relation
// rows present. Performs deferred-safe structural adds, so callers must invoke
// it (and every action below) strictly outside open query loops.
func prepareRelation(world *ecs.World, actorCountry, targetCountry uint32) (actorCap, targetCap ecs.Entity, err error) {
	if actorCountry == 0 || targetCountry == 0 {
		return ecs.Entity{}, ecs.Entity{}, fmt.Errorf("diplomacy requires two real countries (got %d, %d)", actorCountry, targetCountry)
	}
	if actorCountry == targetCountry {
		return ecs.Entity{}, ecs.Entity{}, fmt.Errorf("country %d cannot run diplomacy with itself", actorCountry)
	}
	actorCap, ok := FindCapitalOf(world, actorCountry)
	if !ok {
		return ecs.Entity{}, ecs.Entity{}, fmt.Errorf("country %d has no capital", actorCountry)
	}
	targetCap, ok = FindCapitalOf(world, targetCountry)
	if !ok {
		return ecs.Entity{}, ecs.Entity{}, fmt.Errorf("country %d has no capital", targetCountry)
	}

	diploID := ecs.ComponentID[components.DiplomacyComponent](world)
	if !world.Has(actorCap, diploID) {
		world.Add(actorCap, diploID)
	}
	if !world.Has(targetCap, diploID) {
		world.Add(targetCap, diploID)
	}
	ensureRelation((*components.DiplomacyComponent)(world.Get(actorCap, diploID)), targetCountry)
	ensureRelation((*components.DiplomacyComponent)(world.Get(targetCap, diploID)), actorCountry)
	return actorCap, targetCap, nil
}

// relationOn fetches the live relation row toward targetCountry on a prepared
// capital. Fetch rows fresh after any structural change; never cache across one.
func relationOn(world *ecs.World, capital ecs.Entity, targetCountry uint32) *components.CountryRelation {
	diploID := ecs.ComponentID[components.DiplomacyComponent](world)
	return relationRow((*components.DiplomacyComponent)(world.Get(capital, diploID)), targetCountry)
}

// ImproveRelations spends ImproveRelationsCost gold from the actor's capital
// treasury to raise mutual opinion by ImproveRelationsOpinionGain. tick drives
// the per-target ImproveRelationsCooldown.
func ImproveRelations(world *ecs.World, actorCountry, targetCountry uint32, tick uint64) error {
	actorCap, targetCap, err := prepareRelation(world, actorCountry, targetCountry)
	if err != nil {
		return err
	}

	ra := relationOn(world, actorCap, targetCountry)
	if ra.LastActionTick > 0 && tick-ra.LastActionTick < ImproveRelationsCooldown {
		return fmt.Errorf("envoys need rest: next mission at tick %d", ra.LastActionTick+ImproveRelationsCooldown)
	}

	treasID := ecs.ComponentID[components.TreasuryComponent](world)
	if !world.Has(actorCap, treasID) {
		return fmt.Errorf("country %d has no treasury to fund envoys", actorCountry)
	}
	treasury := (*components.TreasuryComponent)(world.Get(actorCap, treasID))
	if treasury.Wealth < ImproveRelationsCost {
		return fmt.Errorf("treasury too poor: need %.0f gold", ImproveRelationsCost)
	}
	treasury.Wealth -= ImproveRelationsCost

	rb := relationOn(world, targetCap, actorCountry)
	ra.Opinion = clampOpinion(ra.Opinion + ImproveRelationsOpinionGain)
	rb.Opinion = ra.Opinion
	ra.LastActionTick = tick
	return nil
}

// FormAlliance seals a symmetric alliance. The target accepts when mutual
// opinion is at least AllianceOfferMinOpinion and the pair is not at war.
// Already-allied pairs succeed as a no-op.
func FormAlliance(world *ecs.World, actorCountry, targetCountry uint32) error {
	actorCap, targetCap, err := prepareRelation(world, actorCountry, targetCountry)
	if err != nil {
		return err
	}
	ra := relationOn(world, actorCap, targetCountry)
	rb := relationOn(world, targetCap, actorCountry)
	if ra.Alliance {
		return nil
	}
	if ra.AtWar {
		return fmt.Errorf("cannot ally while at war with country %d", targetCountry)
	}
	if ra.Opinion < AllianceOfferMinOpinion {
		return fmt.Errorf("country %d refuses the alliance (opinion %d < %d)", targetCountry, ra.Opinion, AllianceOfferMinOpinion)
	}
	ra.Alliance, rb.Alliance = true, true
	return nil
}

// BreakAlliance dissolves an alliance symmetrically and costs
// BreakAllianceOpinionPenalty mutual opinion. Breaking a non-existent
// alliance is a no-op.
func BreakAlliance(world *ecs.World, actorCountry, targetCountry uint32) error {
	actorCap, targetCap, err := prepareRelation(world, actorCountry, targetCountry)
	if err != nil {
		return err
	}
	ra := relationOn(world, actorCap, targetCountry)
	rb := relationOn(world, targetCap, actorCountry)
	if !ra.Alliance {
		return nil
	}
	ra.Alliance, rb.Alliance = false, false
	ra.Opinion = clampOpinion(ra.Opinion - BreakAllianceOpinionPenalty)
	rb.Opinion = ra.Opinion
	return nil
}

// DeclareWarAction is the player war declaration. It respects truces, funnels
// through the macro DeclareWar path (WarTrackerComponent) and then marks the
// symmetric relation rows. Already-at-war pairs succeed as a no-op.
func DeclareWarAction(world *ecs.World, actorCountry, targetCountry uint32, tick uint64) error {
	actorCap, targetCap, err := prepareRelation(world, actorCountry, targetCountry)
	if err != nil {
		return err
	}
	if ra := relationOn(world, actorCap, targetCountry); ra.AtWar {
		return nil
	} else if tick < ra.TruceUntil {
		return fmt.Errorf("truce with country %d holds until tick %d", targetCountry, ra.TruceUntil)
	}

	// Structural add happens here, so relation rows are fetched afterwards.
	if err := DeclareWar(world, actorCap, targetCountry); err != nil {
		return err
	}
	startWarRows(relationOn(world, actorCap, targetCountry), relationOn(world, targetCap, actorCountry))
	return nil
}

// SueForPeace asks the target to end the war. The target accepts only when the
// WarScore favors it (the actor is losing); the actor then pays the tribute and
// both sides get the standard truce.
func SueForPeace(world *ecs.World, actorCountry, targetCountry uint32, tick uint64) error {
	actorCap, targetCap, err := prepareRelation(world, actorCountry, targetCountry)
	if err != nil {
		return err
	}
	ra := relationOn(world, actorCap, targetCountry)
	rb := relationOn(world, targetCap, actorCountry)
	if !ra.AtWar {
		return fmt.Errorf("not at war with country %d", targetCountry)
	}
	if ra.WarScore >= 0 {
		return fmt.Errorf("country %d refuses peace: the war favors you (WarScore %d)", targetCountry, ra.WarScore)
	}

	sys := NewDiplomacySystem(world)
	sys.settleWar(world, targetCap, actorCap, targetCountry, actorCountry, rb, ra, tick)
	return nil
}

// GetRelations returns a defensive snapshot of a country's diplomatic ledger,
// sorted by TargetCountry for deterministic UI ordering. Returns nil when the
// country has no capital or no diplomacy yet.
func GetRelations(world *ecs.World, countryID uint32) []components.CountryRelation {
	capital, ok := FindCapitalOf(world, countryID)
	if !ok {
		return nil
	}
	diploID := ecs.ComponentID[components.DiplomacyComponent](world)
	if !world.Has(capital, diploID) {
		return nil
	}
	d := (*components.DiplomacyComponent)(world.Get(capital, diploID))
	out := make([]components.CountryRelation, len(d.Relations))
	copy(out, d.Relations)
	slices.SortFunc(out, func(a, b components.CountryRelation) int {
		return int(int64(a.TargetCountry) - int64(b.TargetCountry))
	})
	return out
}
