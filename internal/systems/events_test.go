package systems

// Grand Strategy Phase G1/G2: deterministic E2E tests for the interactive
// event director and ResolveEvent effects on a real ecs.World.
// Run headless: go test ./internal/systems/ -run TestEvent -count=2
// Every test reseeds the global RNG, so -count=2 replays identical streams.

import (
	"fmt"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// eventFixture bundles a seeded world + hook graph + director.
type eventFixture struct {
	world  *ecs.World
	hooks  *engine.SparseHookGraph
	sys    *EventDirectorSystem
	player ecs.Entity
}

func newEventFixture(seed byte) *eventFixture {
	engine.InitializeRNG([32]byte{seed})
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	return &eventFixture{
		world: &world,
		hooks: hooks,
		sys:   NewEventDirectorSystem(&world, hooks),
	}
}

// makeEventCity plants a living settlement (CityBinder convention:
// CityID == uint32(Identity.ID)).
func makeEventCity(world *ecs.World, cityID, countryID, food uint32) ecs.Entity {
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)

	e := world.NewEntity(villageID, identID, affID, storageID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = uint64(cityID)
	ident.Name = "Testheim"
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.CityID = cityID
	aff.CountryID = countryID
	st := (*components.StorageComponent)(world.Get(e, storageID))
	st.Food = food
	return e
}

// makeEventNPC spawns a full living NPC (safety 100 so nobody menaces by
// accident; rig menace explicitly per test).
func makeEventNPC(world *ecs.World, id uint64, cityID, countryID uint32,
	x, y, wealth float32, strength uint8) ecs.Entity {

	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)

	e := world.NewEntity(npcID, identID, posID, needsID, affID, genID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = id
	ident.Name = fmt.Sprintf("NPC-%d", id)
	ident.Age = 30
	pos := (*components.Position)(world.Get(e, posID))
	pos.X, pos.Y = x, y
	n := (*components.Needs)(world.Get(e, needsID))
	n.Food, n.Rest, n.Safety, n.Wealth = 100, 50, 100, wealth
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.CityID = cityID
	aff.CountryID = countryID
	g := (*components.GenomeComponent)(world.Get(e, genID))
	g.Strength = strength
	return e
}

func giveJob(world *ecs.World, e ecs.Entity, job uint8) {
	jobID := ecs.ComponentID[components.JobComponent](world)
	if !world.Has(e, jobID) {
		world.Add(e, jobID)
	}
	(*components.JobComponent)(world.Get(e, jobID)).JobID = job
}

func makeRuler(world *ecs.World, e ecs.Entity, legitimacy uint32) {
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)
	world.Add(e, adminID)
	world.Add(e, legitID)
	(*components.LegitimacyComponent)(world.Get(e, legitID)).Score = legitimacy
}

func possess(world *ecs.World, e ecs.Entity) {
	world.Add(e, ecs.ComponentID[components.Possessed](world))
}

// runEventPulses advances the director by whole main pulses.
func runEventPulses(f *eventFixture, pulses int) {
	for i := 0; i < pulses*int(EventPulseTicks); i++ {
		f.sys.Update(f.world)
	}
}

// queueKind fetches the single pending event of a kind, failing when absent.
func queueKind(t *testing.T, world *ecs.World, player ecs.Entity, kind uint8) components.GameEvent {
	t.Helper()
	for _, ev := range PendingEventList(world, player) {
		if ev.Kind == kind {
			return ev
		}
	}
	t.Fatalf("no pending event of kind %d; queue: %+v", kind, PendingEventList(world, player))
	return components.GameEvent{}
}

func countKind(world *ecs.World, player ecs.Entity, kind uint8) int {
	n := 0
	for _, ev := range PendingEventList(world, player) {
		if ev.Kind == kind {
			n++
		}
	}
	return n
}

func npcWealth(world *ecs.World, e ecs.Entity) float32 {
	needsID := ecs.ComponentID[components.Needs](world)
	return (*components.Needs)(world.Get(e, needsID)).Wealth
}

// TestEventDirector_TaxDemand_Pay: a poor player under a rich ruler draws a
// tax demand; paying transfers gold and warms the ruler's hook.
func TestEventDirector_TaxDemand_Pay(t *testing.T) {
	f := newEventFixture(1)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 20, 50)
	giveJob(f.world, f.player, components.JobFarmer) // Blocks the job-offer generator
	possess(f.world, f.player)
	ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 500, 60)
	makeRuler(f.world, ruler, 50)

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventTaxDemand)
	if ev.ActorID != 2000 || ev.CityID != 10 {
		t.Fatalf("tax demand actor/city = %d/%d, want 2000/10", ev.ActorID, ev.CityID)
	}
	if ev.Amount < TaxDemandBase {
		t.Fatalf("tax demand amount = %d, want >= %d", ev.Amount, TaxDemandBase)
	}

	expectedPay := float32(ev.Amount)
	if expectedPay > 20 {
		expectedPay = 20 // Player cannot pay more than they own
	}
	line := ResolveEvent(f.world, f.hooks, f.player, &ev, 0)
	if line == "" {
		t.Errorf("ResolveEvent(pay) should return a chronicle line")
	}
	if got := npcWealth(f.world, f.player); got != 20-expectedPay {
		t.Errorf("player wealth after pay = %v, want %v", got, 20-expectedPay)
	}
	if got := npcWealth(f.world, ruler); got != 500+expectedPay {
		t.Errorf("ruler wealth after pay = %v, want %v", got, 500+expectedPay)
	}
	if got := f.hooks.GetHook(2000, 1000); got != TaxPayHookGain {
		t.Errorf("ruler hook toward player = %d, want %d", got, TaxPayHookGain)
	}
}

// TestEventDirector_TaxDemand_Refuse: refusal seeds a ruler feud hook and
// dents the ruler's legitimacy.
func TestEventDirector_TaxDemand_Refuse(t *testing.T) {
	f := newEventFixture(2)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 20, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 500, 60)
	makeRuler(f.world, ruler, 50)

	runEventPulses(f, 1)
	ev := queueKind(t, f.world, f.player, components.EventTaxDemand)

	ResolveEvent(f.world, f.hooks, f.player, &ev, 1)
	if got := f.hooks.GetHook(2000, 1000); got != -TaxRefuseHookLoss {
		t.Errorf("ruler feud hook = %d, want %d", got, -TaxRefuseHookLoss)
	}
	legitID := ecs.ComponentID[components.LegitimacyComponent](f.world)
	lg := (*components.LegitimacyComponent)(f.world.Get(ruler, legitID))
	if lg.Score != 50-TaxRefuseLegitimacyLoss {
		t.Errorf("ruler legitimacy = %d, want %d", lg.Score, 50-TaxRefuseLegitimacyLoss)
	}
	if got := npcWealth(f.world, f.player); got != 20 {
		t.Errorf("refusing must not move gold, player wealth = %v", got)
	}
}

// TestEventDirector_JobOffer: an unemployed player is offered the city's
// scarcest trade; accepting swaps the JobComponent.
func TestEventDirector_JobOffer(t *testing.T) {
	f := newEventFixture(3)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 100, 50)
	possess(f.world, f.player) // Unemployed: job-offer generator eligible
	// One farmer exists, so the scarcest trade rotates to lumberjack.
	farmer := makeEventNPC(f.world, 2000, 10, 1, 200, 200, 50, 40)
	giveJob(f.world, farmer, components.JobFarmer)

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventJobOffer)
	if uint8(ev.Amount) != components.JobLumberjack {
		t.Fatalf("offered job = %d, want %d (scarcest trade)", ev.Amount, components.JobLumberjack)
	}

	ResolveEvent(f.world, f.hooks, f.player, &ev, 0)
	jobID := ecs.ComponentID[components.JobComponent](f.world)
	if !f.world.Has(f.player, jobID) {
		t.Fatalf("accepting the offer must add a JobComponent")
	}
	j := (*components.JobComponent)(f.world.Get(f.player, jobID))
	if j.JobID != components.JobLumberjack {
		t.Errorf("player JobID = %d, want %d", j.JobID, components.JobLumberjack)
	}
}

// TestEventDirector_Insult: the deepest grudge-holder insults the player;
// winning the apology contest raises their hook, striking starts combat.
func TestEventDirector_Insult(t *testing.T) {
	f := newEventFixture(4)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 80)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	insulter := makeEventNPC(f.world, 3000, 10, 1, 150, 150, 50, 10)
	giveJob(f.world, insulter, components.JobGuard)
	f.hooks.AddHook(3000, 1000, -40)

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventInsult)
	if ev.ActorID != 3000 {
		t.Fatalf("insult actor = %d, want 3000", ev.ActorID)
	}

	// Contest: player STR 80 vs insulter STR 10 — player wins the apology.
	ResolveEvent(f.world, f.hooks, f.player, &ev, 1)
	if got := f.hooks.GetHook(3000, 1000); got != -40+InsultContestWinGain {
		t.Errorf("post-contest hook = %d, want %d", got, -40+InsultContestWinGain)
	}

	// Strike: the existing combat path gains a CombatMarker at the insulter.
	ResolveEvent(f.world, f.hooks, f.player, &ev, 2)
	combatID := ecs.ComponentID[components.CombatMarker](f.world)
	if !f.world.Has(f.player, combatID) {
		t.Fatalf("striking must attach a CombatMarker")
	}
	cm := (*components.CombatMarker)(f.world.Get(f.player, combatID))
	if cm.TargetID != 3000 {
		t.Errorf("CombatMarker.TargetID = %d, want 3000", cm.TargetID)
	}
}

// TestEventDirector_BanditShakedown: a bandit near the player demands half
// their gold; pay transfers it, flee nudges away and costs rest, fight marks.
func TestEventDirector_BanditShakedown(t *testing.T) {
	f := newEventFixture(5)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 100, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	bandit := makeEventNPC(f.world, 4000, 10, 1, 104, 100, 0, 60)
	giveJob(f.world, bandit, components.JobBandit)

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventBanditShakedown)
	if ev.ActorID != 4000 || ev.Amount != 50 {
		t.Fatalf("shakedown actor/amount = %d/%d, want 4000/50 (half of 100)", ev.ActorID, ev.Amount)
	}

	// Pay: half the purse crosses over.
	ResolveEvent(f.world, f.hooks, f.player, &ev, 0)
	if got := npcWealth(f.world, f.player); got != 50 {
		t.Errorf("player wealth after paying = %v, want 50", got)
	}
	if got := npcWealth(f.world, bandit); got != 50 {
		t.Errorf("bandit wealth after payoff = %v, want 50", got)
	}

	// Flee: dominant-axis teleport-nudge away (+X here) and rest loss.
	posID := ecs.ComponentID[components.Position](f.world)
	needsID := ecs.ComponentID[components.Needs](f.world)
	ResolveEvent(f.world, f.hooks, f.player, &ev, 2)
	pp := (*components.Position)(f.world.Get(f.player, posID))
	if pp.X != 100-FleeDistance || pp.Y != 100 {
		t.Errorf("flee position = (%v,%v), want (%v,100)", pp.X, pp.Y, 100-FleeDistance)
	}
	n := (*components.Needs)(f.world.Get(f.player, needsID))
	if n.Rest != 50-FleeRestLoss {
		t.Errorf("rest after fleeing = %v, want %v", n.Rest, 50-FleeRestLoss)
	}

	// Fight: combat marker at the bandit.
	ResolveEvent(f.world, f.hooks, f.player, &ev, 1)
	combatID := ecs.ComponentID[components.CombatMarker](f.world)
	if !f.world.Has(f.player, combatID) {
		t.Fatalf("fighting must attach a CombatMarker")
	}
	if cm := (*components.CombatMarker)(f.world.Get(f.player, combatID)); cm.TargetID != 4000 {
		t.Errorf("CombatMarker.TargetID = %d, want 4000", cm.TargetID)
	}
}

// TestEventDirector_WarAndPeaceNews: AtWar flips in the diplomacy ledger
// surface as war/peace news events for the player's country (spec G2).
func TestEventDirector_WarAndPeaceNews(t *testing.T) {
	f := newEventFixture(6)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)

	cap1 := newDiploTestCountry(f.world, 1, 100, 0)
	cap2 := newDiploTestCountry(f.world, 2, 100, 0)
	diploID := ecs.ComponentID[components.DiplomacyComponent](f.world)
	f.world.Add(cap1, diploID)
	f.world.Add(cap2, diploID)
	d1 := (*components.DiplomacyComponent)(f.world.Get(cap1, diploID))
	d1.Relations = append(d1.Relations, components.CountryRelation{TargetCountry: 2})
	d2 := (*components.DiplomacyComponent)(f.world.Get(cap2, diploID))
	d2.Relations = append(d2.Relations, components.CountryRelation{TargetCountry: 1})

	// Pulse 1 records the peaceful baseline; nothing is newsworthy yet.
	runEventPulses(f, 1)
	if got := len(PendingEventList(f.world, f.player)); got != 0 {
		t.Fatalf("baseline pulse queued %d events, want 0", got)
	}

	// War breaks out.
	d1 = (*components.DiplomacyComponent)(f.world.Get(cap1, diploID))
	d1.Relations[0].AtWar = true
	d2 = (*components.DiplomacyComponent)(f.world.Get(cap2, diploID))
	d2.Relations[0].AtWar = true
	runEventPulses(f, 1)
	war := queueKind(t, f.world, f.player, components.EventWarNews)
	if war.Amount != 2 {
		t.Fatalf("war news enemy = %d, want 2", war.Amount)
	}
	if line := ResolveEvent(f.world, f.hooks, f.player, &war, 0); line == "" {
		t.Errorf("war news must produce a chronicle line")
	}
	RemovePendingEvent(f.world, f.player, war.ID)

	// Peace returns.
	d1 = (*components.DiplomacyComponent)(f.world.Get(cap1, diploID))
	d1.Relations[0].AtWar = false
	d2 = (*components.DiplomacyComponent)(f.world.Get(cap2, diploID))
	d2.Relations[0].AtWar = false
	runEventPulses(f, 1)
	peace := queueKind(t, f.world, f.player, components.EventPeaceNews)
	if peace.Amount != 2 {
		t.Fatalf("peace news country = %d, want 2", peace.Amount)
	}
}

// TestEventDirector_RulerDied: a cached city ruler who vanished from the
// living surfaces as succession news, flagged claimable on an empty throne.
func TestEventDirector_RulerDied(t *testing.T) {
	f := newEventFixture(7)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	// Equal wealth blocks the tax-demand generator (needs a RICHER ruler).
	ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 50, 60)
	makeRuler(f.world, ruler, 50)

	runEventPulses(f, 1) // Caches the ruler
	if got := len(PendingEventList(f.world, f.player)); got != 0 {
		t.Fatalf("baseline pulse queued %d events, want 0", got)
	}

	f.world.RemoveEntity(ruler)
	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventRulerDied)
	if ev.ActorID != 2000 || ev.CityID != 10 {
		t.Fatalf("ruler-died actor/city = %d/%d, want 2000/10", ev.ActorID, ev.CityID)
	}
	if ev.Flag != components.EventFlagThroneClaimable {
		t.Errorf("empty throne must be flagged claimable, flag = %d", ev.Flag)
	}
	if line := ResolveEvent(f.world, f.hooks, f.player, &ev, 0); line == "" {
		t.Errorf("ruler-died must produce a chronicle line")
	}
}

// TestEventDirector_QueueCapAndDedupe: the queue never exceeds EventQueueCap
// and a pending kind is never duplicated; new IDs resync past persisted ones.
func TestEventDirector_QueueCapAndDedupe(t *testing.T) {
	f := newEventFixture(8)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 20, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 500, 60)
	makeRuler(f.world, ruler, 50)

	// Pre-fill the queue to the cap (as if loaded from a save).
	pendID := ecs.ComponentID[components.PendingEventsComponent](f.world)
	f.world.Add(f.player, pendID)
	pe := (*components.PendingEventsComponent)(f.world.Get(f.player, pendID))
	pe.Events = append(pe.Events,
		components.GameEvent{ID: 900, Kind: components.EventWarNews},
		components.GameEvent{ID: 901, Kind: components.EventPeaceNews},
		components.GameEvent{ID: 902, Kind: components.EventRulerDied},
	)

	runEventPulses(f, 2)
	if got := len(PendingEventList(f.world, f.player)); got != EventQueueCap {
		t.Fatalf("full queue grew to %d, cap is %d", got, EventQueueCap)
	}

	RemovePendingEvent(f.world, f.player, 901)
	runEventPulses(f, 1)
	tax := queueKind(t, f.world, f.player, components.EventTaxDemand)
	if tax.ID <= 902 {
		t.Errorf("new event ID = %d, must resync past persisted 902", tax.ID)
	}

	// Further pulses: the tax kind is pending, so it never duplicates, and
	// the cap holds.
	runEventPulses(f, 3)
	if got := countKind(f.world, f.player, components.EventTaxDemand); got != 1 {
		t.Errorf("pending tax demands = %d, want exactly 1 (dedupe)", got)
	}
	if got := len(PendingEventList(f.world, f.player)); got > EventQueueCap {
		t.Errorf("queue length = %d, cap is %d", got, EventQueueCap)
	}
}

// TestEventDirector_Festival: overflowing granaries spark a festival; joining
// restores rest and forges mutual hooks with two citizens; working pays.
func TestEventDirector_Festival(t *testing.T) {
	f := newEventFixture(9)
	makeEventCity(f.world, 10, 1, 500) // Above FestivalFoodThreshold
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	cit1 := makeEventNPC(f.world, 5001, 10, 1, 200, 200, 50, 40)
	giveJob(f.world, cit1, components.JobGuard)
	cit2 := makeEventNPC(f.world, 5002, 10, 1, 210, 210, 50, 40)
	giveJob(f.world, cit2, components.JobGuard)

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventFestival)
	if ev.CityID != 10 {
		t.Fatalf("festival city = %d, want 10", ev.CityID)
	}

	ResolveEvent(f.world, f.hooks, f.player, &ev, 0)
	needsID := ecs.ComponentID[components.Needs](f.world)
	n := (*components.Needs)(f.world.Get(f.player, needsID))
	if n.Rest != 50+FestivalRestGain {
		t.Errorf("rest after festival = %v, want %v", n.Rest, 50+FestivalRestGain)
	}
	// Both citizens end up as mutual +2 friends (distinct picks enforced).
	for _, id := range []uint64{5001, 5002} {
		if got := f.hooks.GetHook(1000, id); got != FestivalHookGain {
			t.Errorf("player hook toward %d = %d, want %d", id, got, FestivalHookGain)
		}
		if got := f.hooks.GetHook(id, 1000); got != FestivalHookGain {
			t.Errorf("citizen %d hook toward player = %d, want %d", id, got, FestivalHookGain)
		}
	}

	// Work through it: a small payday instead.
	ResolveEvent(f.world, f.hooks, f.player, &ev, 1)
	n = (*components.Needs)(f.world.Get(f.player, needsID))
	if n.Wealth != 50+FestivalWorkGold {
		t.Errorf("wealth after working = %v, want %v", n.Wealth, 50+FestivalWorkGold)
	}
}

// TestEventDirector_Deterministic: identical seeds and worlds produce an
// identical event stream (determinism law).
func TestEventDirector_Deterministic(t *testing.T) {
	build := func() *eventFixture {
		f := newEventFixture(10)
		makeEventCity(f.world, 10, 1, 500)
		f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 20, 50)
		possess(f.world, f.player)
		ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 500, 60)
		makeRuler(f.world, ruler, 50)
		bandit := makeEventNPC(f.world, 4000, 10, 1, 104, 100, 0, 60)
		giveJob(f.world, bandit, components.JobBandit)
		f.hooks.AddHook(3000, 1000, -40) // Grudge from a ghost: filtered (not alive)
		f.hooks.AddHook(4000, 1000, 8)   // The bandit adores the player: marriage eligible
		f.hooks.AddHook(4000, 2000, -25) // ...and resents the ruler: plot invite eligible
		return f
	}

	fa := build()
	runEventPulses(fa, 4)
	qa := PendingEventList(fa.world, fa.player)

	fb := build()
	runEventPulses(fb, 4)
	qb := PendingEventList(fb.world, fb.player)

	if len(qa) != len(qb) {
		t.Fatalf("queue lengths diverged: %d vs %d", len(qa), len(qb))
	}
	for i := range qa {
		if qa[i] != qb[i] {
			t.Errorf("event %d diverged: %+v vs %+v", i, qa[i], qb[i])
		}
	}
	if len(qa) == 0 {
		t.Fatalf("director generated no events in 4 pulses with eligible generators")
	}
}

// markMarried stamps a Married DynastyComponent on an NPC (test rig).
func markMarried(world *ecs.World, e ecs.Entity) {
	dynID := ecs.ComponentID[components.DynastyComponent](world)
	if !world.Has(e, dynID) {
		world.Add(e, dynID)
	}
	(*components.DynastyComponent)(world.Get(e, dynID)).Married = true
}

// TestEventDirector_MarriageProposal_Accept: the fondest eligible admirer
// proposes (married admirers and lesser hooks lose the pick); accepting weds
// both through the standard Marry path with its mutual affection bond.
func TestEventDirector_MarriageProposal_Accept(t *testing.T) {
	f := newEventFixture(11)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer) // Blocks the job-offer generator
	possess(f.world, f.player)
	suitor := makeEventNPC(f.world, 6000, 10, 1, 120, 120, 50, 40)
	makeEventNPC(f.world, 7000, 10, 1, 130, 130, 50, 40) // Lesser hook: loses the pick
	admirer := makeEventNPC(f.world, 8000, 10, 1, 140, 140, 50, 40)
	markMarried(f.world, admirer) // Deepest hook but married: ineligible
	f.hooks.AddHook(6000, 1000, 20)
	f.hooks.AddHook(7000, 1000, 10)
	f.hooks.AddHook(8000, 1000, 50)

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventMarriageProposal)
	if ev.ActorID != 6000 || ev.CityID != 10 {
		t.Fatalf("proposal actor/city = %d/%d, want 6000/10", ev.ActorID, ev.CityID)
	}

	line := ResolveEvent(f.world, f.hooks, f.player, &ev, 0)
	if line == "" {
		t.Errorf("ResolveEvent(accept) should return a chronicle line")
	}
	dynID := ecs.ComponentID[components.DynastyComponent](f.world)
	if !f.world.Has(f.player, dynID) {
		t.Fatalf("accepting must add a DynastyComponent to the player")
	}
	pd := (*components.DynastyComponent)(f.world.Get(f.player, dynID))
	if !pd.Married || pd.SpouseID != 6000 {
		t.Errorf("player dynasty = %+v, want Married with SpouseID 6000", *pd)
	}
	sd := (*components.DynastyComponent)(f.world.Get(suitor, dynID))
	if !sd.Married || sd.SpouseID != 1000 {
		t.Errorf("suitor dynasty = %+v, want Married with SpouseID 1000", *sd)
	}
	// Marriage seals the mutual affection bond on top of the existing hook.
	if got := f.hooks.GetHook(6000, 1000); got != 30 {
		t.Errorf("suitor hook toward player = %d, want 30 (20 + wedding 10)", got)
	}
	if got := f.hooks.GetHook(1000, 6000); got != 10 {
		t.Errorf("player hook toward suitor = %d, want 10 (wedding bond)", got)
	}
}

// TestEventDirector_MarriageProposal_Decline: turning a suitor down costs
// their affection and weds nobody.
func TestEventDirector_MarriageProposal_Decline(t *testing.T) {
	f := newEventFixture(12)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	suitor := makeEventNPC(f.world, 6000, 10, 1, 120, 120, 50, 40)
	f.hooks.AddHook(6000, 1000, 20)

	runEventPulses(f, 1)
	ev := queueKind(t, f.world, f.player, components.EventMarriageProposal)

	if line := ResolveEvent(f.world, f.hooks, f.player, &ev, 1); line == "" {
		t.Errorf("ResolveEvent(decline) should return a chronicle line")
	}
	if got := f.hooks.GetHook(6000, 1000); got != 20-MarriageDeclineHookLoss {
		t.Errorf("spurned suitor hook = %d, want %d", got, 20-MarriageDeclineHookLoss)
	}
	dynID := ecs.ComponentID[components.DynastyComponent](f.world)
	if f.world.Has(f.player, dynID) && (*components.DynastyComponent)(f.world.Get(f.player, dynID)).Married {
		t.Errorf("declining must not wed the player")
	}
	if f.world.Has(suitor, dynID) && (*components.DynastyComponent)(f.world.Get(suitor, dynID)).Married {
		t.Errorf("declining must not wed the suitor")
	}
}

// TestEventDirector_MarriageProposal_MarriedPlayerIneligible: a married
// player never draws a proposal.
func TestEventDirector_MarriageProposal_MarriedPlayerIneligible(t *testing.T) {
	f := newEventFixture(14)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	markMarried(f.world, f.player)
	possess(f.world, f.player)
	makeEventNPC(f.world, 6000, 10, 1, 120, 120, 50, 40)
	f.hooks.AddHook(6000, 1000, 20)

	runEventPulses(f, 2)
	if got := countKind(f.world, f.player, components.EventMarriageProposal); got != 0 {
		t.Errorf("married player drew %d proposals, want 0", got)
	}
}

// TestEventDirector_PlotInvitation: the ruler's bitterest enemy recruits the
// player. Refusing changes nothing; joining creates the seize-rule plot plus
// the conspirator hook (repeat joins only deepen the hook); betraying walks
// the PlotSystem exposure path (Exposed flag, legitimacy dent, target grudge).
func TestEventDirector_PlotInvitation(t *testing.T) {
	f := newEventFixture(13)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	giveJob(f.world, f.player, components.JobFarmer)
	possess(f.world, f.player)
	// Equal wealth blocks the tax-demand generator.
	ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 50, 60)
	makeRuler(f.world, ruler, 50)
	plotter := makeEventNPC(f.world, 9000, 10, 1, 110, 110, 50, 40)
	giveJob(f.world, plotter, components.JobGuard)
	f.hooks.AddHook(9000, 2000, -30) // Strong grudge against the ruler

	runEventPulses(f, 1)

	ev := queueKind(t, f.world, f.player, components.EventPlotInvitation)
	if ev.ActorID != 9000 || ev.TargetID != 2000 || ev.CityID != 10 {
		t.Fatalf("invitation actor/target/city = %d/%d/%d, want 9000/2000/10",
			ev.ActorID, ev.TargetID, ev.CityID)
	}

	plotID := ecs.ComponentID[components.PlotComponent](f.world)

	// Refuse: nothing moves.
	if line := ResolveEvent(f.world, f.hooks, f.player, &ev, 1); line == "" {
		t.Errorf("ResolveEvent(refuse) should return a chronicle line")
	}
	if f.world.Has(plotter, plotID) {
		t.Fatalf("refusing must not start a plot")
	}
	if got := f.hooks.GetHook(1000, 9000); got != 0 {
		t.Errorf("refusing must not bond the player, hook = %d", got)
	}

	// Join: a fresh seize-rule plot on the ruler plus the conspirator hook.
	if line := ResolveEvent(f.world, f.hooks, f.player, &ev, 0); line == "" {
		t.Errorf("ResolveEvent(join) should return a chronicle line")
	}
	if !f.world.Has(plotter, plotID) {
		t.Fatalf("joining must start the plotter's plot")
	}
	plot := (*components.PlotComponent)(f.world.Get(plotter, plotID))
	if plot.TargetID != 2000 || plot.Kind != components.PlotSeizeRule {
		t.Errorf("plot = %+v, want TargetID 2000 kind PlotSeizeRule", *plot)
	}
	if got := f.hooks.GetHook(1000, 9000); got != PlotJoinHookGain {
		t.Errorf("conspirator hook = %d, want %d", got, PlotJoinHookGain)
	}

	// Join again: the existing plot stands, the bond only deepens.
	ResolveEvent(f.world, f.hooks, f.player, &ev, 0)
	if got := f.hooks.GetHook(1000, 9000); got != 2*PlotJoinHookGain {
		t.Errorf("repeat-join hook = %d, want %d", got, 2*PlotJoinHookGain)
	}

	// Betray: exposure penalties via the PlotSystem discovery path.
	legitID := ecs.ComponentID[components.LegitimacyComponent](f.world)
	f.world.Add(plotter, legitID)
	(*components.LegitimacyComponent)(f.world.Get(plotter, legitID)).Score = 30
	if line := ResolveEvent(f.world, f.hooks, f.player, &ev, 2); line == "" {
		t.Errorf("ResolveEvent(betray) should return a chronicle line")
	}
	plot = (*components.PlotComponent)(f.world.Get(plotter, plotID))
	if !plot.Exposed {
		t.Errorf("betrayal must mark the plot Exposed")
	}
	legit := (*components.LegitimacyComponent)(f.world.Get(plotter, legitID))
	if legit.Score != 30-plotExposureLegitimacyLoss {
		t.Errorf("plotter legitimacy = %d, want %d", legit.Score, 30-plotExposureLegitimacyLoss)
	}
	if got := f.hooks.GetHook(2000, 9000); got != plotExposureHookPenalty {
		t.Errorf("target grudge hook = %d, want %d", got, plotExposureHookPenalty)
	}
}

// TestEventDirector_PlotInvitation_BetrayWithoutPlot: betraying an invitation
// whose plot never materialized still teaches the target who schemed, and
// creates nothing.
func TestEventDirector_PlotInvitation_BetrayWithoutPlot(t *testing.T) {
	f := newEventFixture(15)
	makeEventCity(f.world, 10, 1, 100)
	f.player = makeEventNPC(f.world, 1000, 10, 1, 100, 100, 50, 50)
	possess(f.world, f.player)
	ruler := makeEventNPC(f.world, 2000, 10, 1, 105, 105, 50, 60)
	makeRuler(f.world, ruler, 50)
	plotter := makeEventNPC(f.world, 9000, 10, 1, 110, 110, 50, 40)

	ev := components.GameEvent{
		Kind: components.EventPlotInvitation, ActorID: 9000, TargetID: 2000, CityID: 10}
	if line := ResolveEvent(f.world, f.hooks, f.player, &ev, 2); line == "" {
		t.Errorf("ResolveEvent(betray) should return a chronicle line")
	}
	plotID := ecs.ComponentID[components.PlotComponent](f.world)
	if f.world.Has(plotter, plotID) {
		t.Errorf("betrayal must not create a plot")
	}
	if got := f.hooks.GetHook(2000, 9000); got != plotExposureHookPenalty {
		t.Errorf("target grudge hook = %d, want %d", got, plotExposureHookPenalty)
	}
}
