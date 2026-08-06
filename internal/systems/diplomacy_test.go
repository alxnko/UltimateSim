package systems

import (
	"reflect"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase P2.2: deterministic E2E tests for DiplomacySystem.
// Run headless: go test ./internal/systems/ -run TestDiplomacy -count=2

// newDiploTestCountry spawns a capital in the exact shape the macro country
// model uses (Capital + Country + Affiliation + Village + Treasury) plus pop
// NPC citizens, and returns the capital entity.
func newDiploTestCountry(world *ecs.World, countryID uint32, wealth float32, pop int) ecs.Entity {
	capID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	villageID := ecs.ComponentID[components.Village](world)
	treasID := ecs.ComponentID[components.TreasuryComponent](world)

	entity := world.NewEntity(capID, countryCompID, affID, villageID, treasID)
	aff := (*components.Affiliation)(world.Get(entity, affID))
	aff.CountryID = countryID
	treas := (*components.TreasuryComponent)(world.Get(entity, treasID))
	treas.Wealth = wealth

	npcID := ecs.ComponentID[components.NPC](world)
	for i := 0; i < pop; i++ {
		npc := world.NewEntity(npcID, affID)
		npcAff := (*components.Affiliation)(world.Get(npc, affID))
		npcAff.CountryID = countryID
	}
	return entity
}

// placeCapital adds a Position so border friction applies.
func placeCapital(world *ecs.World, capital ecs.Entity, x, y float32) {
	posID := ecs.ComponentID[components.Position](world)
	world.Add(capital, posID)
	pos := (*components.Position)(world.Get(capital, posID))
	pos.X, pos.Y = x, y
}

// runPulses advances the system by whole diplomacy pulses.
func runPulses(sys *DiplomacySystem, world *ecs.World, pulses int) {
	for i := 0; i < pulses*int(DiplomacyPulseTicks); i++ {
		sys.Update(world)
	}
}

// setMutualOpinion overwrites the symmetric opinion between two countries.
func setMutualOpinion(t *testing.T, world *ecs.World, c1, c2 uint32, opinion int16) {
	t.Helper()
	cap1, ok := FindCapitalOf(world, c1)
	if !ok {
		t.Fatalf("no capital for country %d", c1)
	}
	cap2, ok := FindCapitalOf(world, c2)
	if !ok {
		t.Fatalf("no capital for country %d", c2)
	}
	relationOn(world, cap1, c2).Opinion = opinion
	relationOn(world, cap2, c1).Opinion = opinion
}

// relationSnapshot fetches the single relation row c1 -> c2 via the UI helper.
func relationSnapshot(t *testing.T, world *ecs.World, c1, c2 uint32) components.CountryRelation {
	t.Helper()
	for _, rel := range GetRelations(world, c1) {
		if rel.TargetCountry == c2 {
			return rel
		}
	}
	t.Fatalf("no relation %d -> %d", c1, c2)
	return components.CountryRelation{}
}

// TestDiplomacy_WarDeclarationAtLowOpinion verifies the AI declares war through
// the existing macro DeclareWar path once opinion sinks below the threshold.
func TestDiplomacy_WarDeclarationAtLowOpinion(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewDiplomacySystem(&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)

	cap1 := newDiploTestCountry(&world, 1, 100, 0)
	newDiploTestCountry(&world, 2, 100, 0)

	// First pulse initializes symmetric neutral relations; no war at opinion 0.
	runPulses(sys, &world, 1)
	rel := relationSnapshot(t, &world, 1, 2)
	if rel.AtWar || rel.Alliance || rel.Opinion != 0 {
		t.Fatalf("expected neutral initial relation, got %+v", rel)
	}
	if world.Has(cap1, warCompID) {
		t.Fatalf("neutral countries must not gain a war tracker")
	}

	// Drop mutual opinion below the war threshold: next pulse declares war.
	setMutualOpinion(t, &world, 1, 2, -70)
	runPulses(sys, &world, 1)

	rel12 := relationSnapshot(t, &world, 1, 2)
	rel21 := relationSnapshot(t, &world, 2, 1)
	if !rel12.AtWar || !rel21.AtWar {
		t.Fatalf("expected symmetric war, got %+v / %+v", rel12, rel21)
	}
	if rel12.Alliance || rel21.Alliance {
		t.Fatalf("war must clear alliances")
	}
	if !world.Has(cap1, warCompID) {
		t.Fatalf("AI war must go through DeclareWar (macro tracker missing)")
	}
	war := (*components.WarTrackerComponent)(world.Get(cap1, warCompID))
	if !war.Active || war.TargetCountryID != 2 {
		t.Fatalf("expected active macro war on country 2, got %+v", war)
	}
}

// TestDiplomacy_AllianceAtHighOpinion verifies symmetric auto-accepted
// alliances above the opinion threshold.
func TestDiplomacy_AllianceAtHighOpinion(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewDiplomacySystem(&world)

	newDiploTestCountry(&world, 1, 100, 0)
	newDiploTestCountry(&world, 2, 100, 0)
	runPulses(sys, &world, 1)

	setMutualOpinion(t, &world, 1, 2, 70)
	runPulses(sys, &world, 1)

	rel12 := relationSnapshot(t, &world, 1, 2)
	rel21 := relationSnapshot(t, &world, 2, 1)
	if !rel12.Alliance || !rel21.Alliance {
		t.Fatalf("expected symmetric alliance, got %+v / %+v", rel12, rel21)
	}
	if rel12.AtWar || rel21.AtWar {
		t.Fatalf("allies must not be at war")
	}
	// Opinion drifted toward zero exactly once before the alliance check.
	if rel12.Opinion != 69 {
		t.Fatalf("expected drifted opinion 69, got %d", rel12.Opinion)
	}
}

// TestDiplomacy_WarScoreSettlementTruceAndTribute drives a full war to the
// WarScore cap and verifies the settlement: truce, tribute transfer, macro
// peace, and the truce blocking a new declaration until it expires.
func TestDiplomacy_WarScoreSettlementTruceAndTribute(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewDiplomacySystem(&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	treasID := ecs.ComponentID[components.TreasuryComponent](&world)

	cap1 := newDiploTestCountry(&world, 1, 2000, 0) // strong side
	cap2 := newDiploTestCountry(&world, 2, 400, 0)  // weak side

	if err := DeclareWarAction(&world, 1, 2, 1); err != nil {
		t.Fatalf("unexpected declare error: %v", err)
	}
	rel := relationSnapshot(t, &world, 1, 2)
	if !rel.AtWar || rel.Opinion > WarDeclareOpinion {
		t.Fatalf("expected fresh war at hostile opinion, got %+v", rel)
	}

	// Strengths 2000 vs 400: step = 1600*25/2401 = 16 per pulse.
	runPulses(sys, &world, 1)
	rel = relationSnapshot(t, &world, 1, 2)
	if rel.WarScore != 16 {
		t.Fatalf("expected WarScore 16 after one pulse, got %d", rel.WarScore)
	}
	if mirror := relationSnapshot(t, &world, 2, 1); mirror.WarScore != -16 {
		t.Fatalf("expected mirrored WarScore -16, got %d", mirror.WarScore)
	}

	// Pulse 7 pushes 96+16 past the cap and settles the war (tick 2100).
	runPulses(sys, &world, 6)
	rel12 := relationSnapshot(t, &world, 1, 2)
	rel21 := relationSnapshot(t, &world, 2, 1)
	if rel12.AtWar || rel21.AtWar {
		t.Fatalf("expected settled war, got %+v / %+v", rel12, rel21)
	}
	if rel12.WarScore != 0 || rel21.WarScore != 0 {
		t.Fatalf("expected WarScore reset, got %d / %d", rel12.WarScore, rel21.WarScore)
	}
	wantTruce := 7*DiplomacyPulseTicks + TruceDurationTicks // 2100 + 2000
	if rel12.TruceUntil != wantTruce || rel21.TruceUntil != wantTruce {
		t.Fatalf("expected truce until %d, got %d / %d", wantTruce, rel12.TruceUntil, rel21.TruceUntil)
	}
	if rel12.Opinion != PostWarOpinion || rel21.Opinion != PostWarOpinion {
		t.Fatalf("expected post-war opinion %d, got %d / %d", PostWarOpinion, rel12.Opinion, rel21.Opinion)
	}

	// Loser paid 25% of its 400 treasury to the winner.
	winnerT := (*components.TreasuryComponent)(world.Get(cap1, treasID))
	loserT := (*components.TreasuryComponent)(world.Get(cap2, treasID))
	if winnerT.Wealth != 2100 || loserT.Wealth != 300 {
		t.Fatalf("expected tribute 2100/300, got %f/%f", winnerT.Wealth, loserT.Wealth)
	}

	// Macro model is back in sync: the tracker exists but is inactive.
	war := (*components.WarTrackerComponent)(world.Get(cap1, warCompID))
	if war.Active {
		t.Fatalf("settlement must deactivate the macro war tracker")
	}

	// The truce blocks a re-declaration until it expires.
	if err := DeclareWarAction(&world, 1, 2, wantTruce-1); err == nil {
		t.Fatalf("expected truce to block war declaration")
	}
	if err := DeclareWarAction(&world, 1, 2, wantTruce); err != nil {
		t.Fatalf("expected declaration after truce expiry, got %v", err)
	}
}

// TestDiplomacy_MacroParitySync verifies wars started/ended through the
// pre-existing law-action path (ruler UI) are mirrored into the ledger.
func TestDiplomacy_MacroParitySync(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewDiplomacySystem(&world)

	cap1 := newDiploTestCountry(&world, 1, 100, 0)
	newDiploTestCountry(&world, 2, 100, 0)

	// War declared via the macro path only (as state_playing.go does today).
	if err := DeclareWar(&world, cap1, 2); err != nil {
		t.Fatalf("unexpected macro declare error: %v", err)
	}
	runPulses(sys, &world, 1)
	rel := relationSnapshot(t, &world, 1, 2)
	if !rel.AtWar || rel.Opinion > WarDeclareOpinion {
		t.Fatalf("macro war must sync into the ledger, got %+v", rel)
	}

	// Macro peace (e.g. WarEconomySystem state default) yields a white peace.
	MakePeace(&world, cap1)
	runPulses(sys, &world, 1)
	rel12 := relationSnapshot(t, &world, 1, 2)
	rel21 := relationSnapshot(t, &world, 2, 1)
	if rel12.AtWar || rel21.AtWar {
		t.Fatalf("macro peace must sync into the ledger, got %+v / %+v", rel12, rel21)
	}
	if rel12.TruceUntil == 0 {
		t.Fatalf("white peace must still start a truce")
	}
}

// TestDiplomacy_PlayerActions exercises ImproveRelations (cost + cooldown),
// FormAlliance/BreakAlliance and SueForPeace acceptance rules.
func TestDiplomacy_PlayerActions(t *testing.T) {
	world := ecs.NewWorld()
	treasID := ecs.ComponentID[components.TreasuryComponent](&world)

	cap1 := newDiploTestCountry(&world, 1, 100, 0)
	cap2 := newDiploTestCountry(&world, 2, 100, 0)

	// Envoy missions cost gold and raise mutual opinion.
	if err := ImproveRelations(&world, 1, 2, 100); err != nil {
		t.Fatalf("unexpected improve error: %v", err)
	}
	if rel := relationSnapshot(t, &world, 1, 2); rel.Opinion != ImproveRelationsOpinionGain {
		t.Fatalf("expected opinion %d, got %d", ImproveRelationsOpinionGain, rel.Opinion)
	}
	if rel := relationSnapshot(t, &world, 2, 1); rel.Opinion != ImproveRelationsOpinionGain {
		t.Fatalf("expected mirrored opinion gain, got %d", rel.Opinion)
	}
	if tr := (*components.TreasuryComponent)(world.Get(cap1, treasID)); tr.Wealth != 100-ImproveRelationsCost {
		t.Fatalf("expected treasury %f, got %f", 100-ImproveRelationsCost, tr.Wealth)
	}

	// Cooldown blocks immediate reuse; expiry allows it again.
	if err := ImproveRelations(&world, 1, 2, 200); err == nil {
		t.Fatalf("expected cooldown rejection")
	}
	if err := ImproveRelations(&world, 1, 2, 100+ImproveRelationsCooldown); err != nil {
		t.Fatalf("expected improve after cooldown, got %v", err)
	}

	// Treasury exhausted (100 - 2*50 = 0): third mission is refused.
	if err := ImproveRelations(&world, 1, 2, 100+2*ImproveRelationsCooldown); err == nil {
		t.Fatalf("expected poverty rejection")
	}

	// Alliance requires enough opinion (currently 20 < 40).
	if err := FormAlliance(&world, 1, 2); err == nil {
		t.Fatalf("expected alliance refusal at low opinion")
	}
	setMutualOpinion(t, &world, 1, 2, AllianceOfferMinOpinion)
	if err := FormAlliance(&world, 1, 2); err != nil {
		t.Fatalf("unexpected alliance error: %v", err)
	}
	if rel := relationSnapshot(t, &world, 2, 1); !rel.Alliance {
		t.Fatalf("expected symmetric alliance")
	}

	// Breaking it costs opinion on both sides.
	if err := BreakAlliance(&world, 1, 2); err != nil {
		t.Fatalf("unexpected break error: %v", err)
	}
	rel := relationSnapshot(t, &world, 1, 2)
	if rel.Alliance || rel.Opinion != AllianceOfferMinOpinion-BreakAllianceOpinionPenalty {
		t.Fatalf("expected broken alliance at opinion %d, got %+v", AllianceOfferMinOpinion-BreakAllianceOpinionPenalty, rel)
	}

	// At war: alliances are refused and peace is only granted to the loser.
	if err := DeclareWarAction(&world, 1, 2, 1000); err != nil {
		t.Fatalf("unexpected declare error: %v", err)
	}
	if err := FormAlliance(&world, 1, 2); err == nil {
		t.Fatalf("expected alliance refusal while at war")
	}
	if err := SueForPeace(&world, 1, 2, 1100); err == nil {
		t.Fatalf("expected peace refusal while WarScore does not favor the target")
	}

	// Once the actor is losing, the target accepts and collects tribute.
	relationOn(&world, cap1, 2).WarScore = -50
	relationOn(&world, cap2, 1).WarScore = 50
	if err := SueForPeace(&world, 1, 2, 1200); err != nil {
		t.Fatalf("unexpected sue-for-peace error: %v", err)
	}
	rel12 := relationSnapshot(t, &world, 1, 2)
	rel21 := relationSnapshot(t, &world, 2, 1)
	if rel12.AtWar || rel21.AtWar || rel12.TruceUntil != 1200+TruceDurationTicks {
		t.Fatalf("expected settled war with truce, got %+v / %+v", rel12, rel21)
	}
	actorT := (*components.TreasuryComponent)(world.Get(cap1, treasID))
	targetT := (*components.TreasuryComponent)(world.Get(cap2, treasID))
	if actorT.Wealth != 0 || targetT.Wealth != 100 {
		t.Fatalf("expected tribute 0/100, got %f/%f", actorT.Wealth, targetT.Wealth)
	}
}

// buildDeterminismWorld fabricates three bordering countries with unequal
// strength so friction, wars, settlements and truces all fire organically.
func buildDeterminismWorld() (*ecs.World, *DiplomacySystem) {
	world := ecs.NewWorld()
	sys := NewDiplomacySystem(&world)

	capA := newDiploTestCountry(&world, 1, 2000, 3)
	capB := newDiploTestCountry(&world, 2, 500, 2)
	capC := newDiploTestCountry(&world, 3, 500, 1)
	placeCapital(&world, capA, 0, 0)
	placeCapital(&world, capB, 30, 0)
	placeCapital(&world, capC, 0, 30)
	return &world, sys
}

// TestDiplomacy_Determinism runs the identical scenario in two separately
// built worlds through 75 pulses (wars, settlements, truces included) and
// requires bit-identical ledgers. -count=2 additionally proves run-to-run
// stability of a single world.
func TestDiplomacy_Determinism(t *testing.T) {
	worldA, sysA := buildDeterminismWorld()
	worldB, sysB := buildDeterminismWorld()

	runPulses(sysA, worldA, 75)
	runPulses(sysB, worldB, 75)

	sawWar := false
	for _, country := range []uint32{1, 2, 3} {
		relsA := GetRelations(worldA, country)
		relsB := GetRelations(worldB, country)
		if len(relsA) != 2 {
			t.Fatalf("country %d: expected 2 relations, got %d", country, len(relsA))
		}
		if !reflect.DeepEqual(relsA, relsB) {
			t.Fatalf("country %d diverged:\nA: %+v\nB: %+v", country, relsA, relsB)
		}
		for _, rel := range relsA {
			if rel.AtWar || rel.TruceUntil > 0 {
				sawWar = true
			}
		}
	}
	// Border friction must have produced at least one organic war by pulse 75.
	if !sawWar {
		t.Fatalf("expected organic wars in the determinism scenario")
	}
}
