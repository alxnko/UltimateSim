package systems

// Grand Strategy Phase G1/G2/G5/G6: the interactive event director — the
// CK-style drama engine. Every pulse it examines the possessed player's
// situation and the world state and MAY enqueue ONE interactive event on the
// player's PendingEventsComponent. The UI (internal/ui/panel_events.go) pops
// the queue as a modal with 2-3 choices; every choice funnels through the
// single exported ResolveEvent so effects stay deterministic and testable.
// Spec: docs/superpowers/specs/2026-08-06-grand-strategy-playable.md
//
// Determinism: seeded engine RNG only, fixed query shapes, sorted snapshots,
// maps used strictly for lookup (never iterated unsorted). All structural
// mutations (component adds) happen after every query loop has finished.

import (
	"fmt"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Event director tuning constants (exported where the UI/tests surface them).
const (
	// EventPulseTicks is the main event cadence: ~10s of play at 1x/60TPS.
	EventPulseTicks uint64 = 600
	// eventJitterTicks/eventJitterChance: between main pulses, every jitter
	// tick rolls a 1-in-N chance of a bonus pulse so cadence feels organic
	// instead of metronomic (spec G6: an event every ~30-90s, never a dead
	// stretch; dedupe + the queue cap keep the flow from flooding).
	eventJitterTicks  uint64 = 60
	eventJitterChance uint64 = 12

	// EventQueueCap is the maximum number of pending events on the player.
	EventQueueCap = 3

	// Tax demand: demand = base + rand%spread gold.
	TaxDemandBase           int32  = 15
	taxDemandSpread         uint64 = 26
	TaxPayHookGain                 = 10 // Ruler warms to a compliant subject
	TaxRefuseHookLoss              = 15 // Feud seed: ruler's grudge on refusal
	TaxRefuseLegitimacyLoss uint32 = 5  // Public defiance weakens the ruler

	// Bandit shakedown.
	BanditRadius          float32 = 30 // Menace search radius around the player (tiles)
	BanditSafetyThreshold float32 = 30 // Desperate NPCs (Safety below) also menace
	BanditMinDemand       int32   = 5
	FleeDistance          float32 = 6  // Teleport-nudge away from the menace
	FleeRestLoss          float32 = 10

	// Insult contest: score = Genome.Strength + positive incoming hooks.
	InsultContestWinGain     = 15 // Loser's hook toward the winner rises
	InsultContestLossPenalty = 10 // Standing up and losing deepens the grudge

	// Festival.
	FestivalFoodThreshold uint32  = 400 // City food stock that sparks a festival
	FestivalRestGain      float32 = 20
	FestivalHookGain              = 2 // Mutual hooks with 2 random citizens
	FestivalWorkGold      float32 = 8

	// Marriage proposal: the fondest unmarried adult proposes (spec G1).
	MarriageDeclineHookLoss = 5 // Spurned affection curdles

	// Plot invitation: a malcontent recruits the player (spec G1).
	PlotInviteHookThreshold = -20 // Hook toward the ruler that marks a recruiter
	PlotJoinHookGain        = 5   // Joining bonds the player to the plotter (conspirator hook)
)

// eventTradeJobs is the fixed job rotation a city can offer (genesis trades).
var eventTradeJobs = [...]uint8{
	components.JobFarmer, components.JobLumberjack, components.JobArtisan,
	components.JobGuard, components.JobBuilder, components.JobMiner,
}

// eventNPC is the flat per-pulse snapshot of one living NPC.
type eventNPC struct {
	entity    ecs.Entity
	id        uint64
	x, y      float32
	wealth    float32
	safety    float32
	cityID    uint32
	countryID uint32
	age       uint16
	strength  uint8
	jobID     uint8
	hasJob    bool
	hasPos    bool
	hasNeeds  bool
	isRuler   bool
	possessed bool
	married   bool
}

// EventDirectorSystem generates interactive events. Register under
// engine.PhaseResolution.
type EventDirectorSystem struct {
	hooks       *engine.SparseHookGraph
	tick        uint64
	clock       *engine.TickManager // shared persisted clock; nil in unit tests
	nextEventID uint64

	// Diff caches for news events (war/peace flips, dead rulers). primed is
	// false until the first pulse recorded a baseline, so pre-existing wars
	// never spam news at game start (or after load).
	primed     bool
	warCache   map[uint64]bool   // pair key (see warPairKey) -> AtWar
	rulerCache map[uint32]uint64 // cityID -> ruler Identity.ID

	// Component IDs.
	npcID         ecs.ID
	identID       ecs.ID
	posID         ecs.ID
	needsID       ecs.ID
	affID         ecs.ID
	jobID         ecs.ID
	genID         ecs.ID
	adminID       ecs.ID
	possID        ecs.ID
	villageID     ecs.ID
	storageID     ecs.ID
	ruinID        ecs.ID
	capID         ecs.ID
	countryCompID ecs.ID
	diploID       ecs.ID
	pendID        ecs.ID
	dynID         ecs.ID
}

// SetClock binds the shared TickManager clock so GameEvent.Tick stamps live
// in the persisted tick domain (survives save/load).
func (s *EventDirectorSystem) SetClock(tm *engine.TickManager) { s.clock = tm }

// NewEventDirectorSystem binds the director to the world's component IDs.
func NewEventDirectorSystem(world *ecs.World, hooks *engine.SparseHookGraph) *EventDirectorSystem {
	return &EventDirectorSystem{
		hooks:         hooks,
		warCache:      make(map[uint64]bool),
		rulerCache:    make(map[uint32]uint64),
		npcID:         ecs.ComponentID[components.NPC](world),
		identID:       ecs.ComponentID[components.Identity](world),
		posID:         ecs.ComponentID[components.Position](world),
		needsID:       ecs.ComponentID[components.Needs](world),
		affID:         ecs.ComponentID[components.Affiliation](world),
		jobID:         ecs.ComponentID[components.JobComponent](world),
		genID:         ecs.ComponentID[components.GenomeComponent](world),
		adminID:       ecs.ComponentID[components.AdministrationMarker](world),
		possID:        ecs.ComponentID[components.Possessed](world),
		villageID:     ecs.ComponentID[components.Village](world),
		storageID:     ecs.ComponentID[components.StorageComponent](world),
		ruinID:        ecs.ComponentID[components.RuinComponent](world),
		capID:         ecs.ComponentID[components.CapitalComponent](world),
		countryCompID: ecs.ComponentID[components.CountryComponent](world),
		diploID:       ecs.ComponentID[components.DiplomacyComponent](world),
		pendID:        ecs.ComponentID[components.PendingEventsComponent](world),
		dynID:         ecs.ComponentID[components.DynastyComponent](world),
	}
}

// Update runs the pulse cadence: a guaranteed pulse every EventPulseTicks plus
// a rare-roll bonus pulse on jitter ticks so the rhythm feels organic. The RNG
// call pattern is a pure function of the tick counter (determinism law).
func (s *EventDirectorSystem) Update(world *ecs.World) {
	s.tick++
	if s.clock != nil {
		s.tick = s.clock.Ticks
	}
	pulse := s.tick%EventPulseTicks == 0
	if !pulse && s.tick%eventJitterTicks == 0 {
		if uint64(engine.GetRandomInt())%eventJitterChance == 0 {
			pulse = true
		}
	}
	if !pulse {
		return
	}
	s.runPulse(world)
}

// warPairKey packs an unordered country pair into a sortable map key.
func warPairKey(a, b uint32) uint64 {
	if a > b {
		a, b = b, a
	}
	return uint64(a)<<32 | uint64(b)
}

// runPulse snapshots the world, diffs drama (war/peace, dead rulers), then
// rolls at most one organic event for the possessed player.
func (s *EventDirectorSystem) runPulse(world *ecs.World) {
	// --- 1. Snapshot every living NPC (query fully consumed, then sorted). ---
	var npcs []eventNPC
	nf := ecs.All(s.npcID, s.identID)
	nq := world.Query(&nf)
	for nq.Next() {
		d := eventNPC{entity: nq.Entity()}
		ident := (*components.Identity)(nq.Get(s.identID))
		d.id = ident.ID
		d.age = ident.Age
		if nq.Has(s.dynID) {
			d.married = (*components.DynastyComponent)(nq.Get(s.dynID)).Married
		}
		if nq.Has(s.posID) {
			p := (*components.Position)(nq.Get(s.posID))
			d.x, d.y, d.hasPos = p.X, p.Y, true
		}
		if nq.Has(s.needsID) {
			n := (*components.Needs)(nq.Get(s.needsID))
			d.wealth, d.safety, d.hasNeeds = n.Wealth, n.Safety, true
		}
		if nq.Has(s.affID) {
			a := (*components.Affiliation)(nq.Get(s.affID))
			d.cityID, d.countryID = a.CityID, a.CountryID
		}
		if nq.Has(s.jobID) {
			d.jobID = (*components.JobComponent)(nq.Get(s.jobID)).JobID
			d.hasJob = true
		}
		if nq.Has(s.genID) {
			d.strength = (*components.GenomeComponent)(nq.Get(s.genID)).Strength
		}
		d.isRuler = nq.Has(s.adminID)
		d.possessed = nq.Has(s.possID)
		npcs = append(npcs, d)
	}
	slices.SortFunc(npcs, func(a, b eventNPC) int {
		switch {
		case a.id < b.id:
			return -1
		case a.id > b.id:
			return 1
		default:
			return 0
		}
	})

	playerIdx := -1
	alive := make(map[uint64]bool, len(npcs)) // Lookup only
	rulers := make(map[uint32]uint64)         // cityID -> lowest ruler id (lookup only)
	for i := range npcs {
		alive[npcs[i].id] = true
		if npcs[i].possessed && playerIdx < 0 {
			playerIdx = i
		}
		if npcs[i].isRuler && npcs[i].cityID != 0 {
			if _, taken := rulers[npcs[i].cityID]; !taken {
				rulers[npcs[i].cityID] = npcs[i].id // npcs sorted: lowest id wins
			}
		}
	}

	var playerCity, playerCountry uint32
	if playerIdx >= 0 {
		playerCity = npcs[playerIdx].cityID
		playerCountry = npcs[playerIdx].countryID
	}

	// --- 2. War-state snapshot from every capital's diplomacy ledger. ---
	warNow := make(map[uint64]bool)
	cf := ecs.All(s.capID, s.countryCompID, s.affID)
	cq := world.Query(&cf)
	for cq.Next() {
		aff := (*components.Affiliation)(cq.Get(s.affID))
		if aff.CountryID == 0 || !cq.Has(s.diploID) {
			continue
		}
		d := (*components.DiplomacyComponent)(cq.Get(s.diploID))
		for i := range d.Relations {
			key := warPairKey(aff.CountryID, d.Relations[i].TargetCountry)
			warNow[key] = warNow[key] || d.Relations[i].AtWar
		}
	}

	// --- 3. Player-city granary level (festival trigger). ---
	var playerCityFood uint32
	vfilter := ecs.All(s.villageID, s.identID, s.storageID)
	vq := world.Query(&vfilter)
	for vq.Next() {
		if vq.Has(s.ruinID) {
			continue
		}
		ident := (*components.Identity)(vq.Get(s.identID))
		if playerCity != 0 && uint32(ident.ID) == playerCity {
			playerCityFood = (*components.StorageComponent)(vq.Get(s.storageID)).Food
		}
	}

	// --- 4. Diff news against the caches (spec G2 drama surfacing). ---
	var news []components.GameEvent
	if s.primed && playerIdx >= 0 && playerCountry != 0 {
		keys := make([]uint64, 0, len(warNow)+len(s.warCache))
		seen := make(map[uint64]struct{}, len(warNow)+len(s.warCache))
		for k := range warNow {
			keys = append(keys, k)
			seen[k] = struct{}{}
		}
		for k := range s.warCache {
			if _, dup := seen[k]; !dup {
				keys = append(keys, k)
			}
		}
		slices.Sort(keys) // Deterministic diff order
		for _, k := range keys {
			a, b := uint32(k>>32), uint32(k)
			if a != playerCountry && b != playerCountry {
				continue
			}
			other := a
			if a == playerCountry {
				other = b
			}
			was, is := s.warCache[k], warNow[k]
			if is && !was {
				news = append(news, components.GameEvent{
					Kind: components.EventWarNews, CityID: playerCity, Amount: int32(other)})
			} else if was && !is {
				news = append(news, components.GameEvent{
					Kind: components.EventPeaceNews, CityID: playerCity, Amount: int32(other)})
			}
		}
	}
	if s.primed && playerIdx >= 0 && playerCity != 0 {
		if prev := s.rulerCache[playerCity]; prev != 0 && prev != rulers[playerCity] && !alive[prev] {
			ev := components.GameEvent{
				Kind: components.EventRulerDied, CityID: playerCity, ActorID: prev}
			// All queries are closed here; CanClaimLeadership runs its own
			// fully-consumed queries.
			if CanClaimLeadership(world, s.hooks, npcs[playerIdx].entity) {
				ev.Flag = components.EventFlagThroneClaimable
			}
			news = append(news, ev)
		}
	}

	// Refresh caches every pulse, even while nobody is possessed.
	s.warCache = warNow
	rc := make(map[uint32]uint64, len(rulers))
	for city, id := range rulers {
		rc[city] = id
	}
	s.rulerCache = rc
	s.primed = true

	if playerIdx < 0 {
		return
	}
	player := &npcs[playerIdx]

	// --- 5. Queue state: cap + kind dedupe. ---
	var pendingKinds [16]bool
	queueLen := 0
	if world.Has(player.entity, s.pendID) {
		pe := (*components.PendingEventsComponent)(world.Get(player.entity, s.pendID))
		queueLen = len(pe.Events)
		for i := range pe.Events {
			if k := pe.Events[i].Kind; int(k) < len(pendingKinds) {
				pendingKinds[k] = true
			}
		}
	}
	if queueLen >= EventQueueCap {
		return
	}

	// --- 6. Choose this pulse's single event: news first, then one organic
	// roll among the eligible generators. ---
	var chosen components.GameEvent
	found := false
	for i := range news {
		if !pendingKinds[news[i].Kind] {
			chosen = news[i]
			found = true
			break
		}
	}
	if !found {
		cands := s.organicCandidates(npcs, playerIdx, rulers, playerCityFood)
		eligible := cands[:0]
		for i := range cands {
			if !pendingKinds[cands[i].Kind] {
				eligible = append(eligible, cands[i])
			}
		}
		if len(eligible) > 0 {
			chosen = eligible[engine.GetRandomInt()%len(eligible)]
			if chosen.Kind == components.EventTaxDemand {
				chosen.Amount = TaxDemandBase + int32(uint64(engine.GetRandomInt())%taxDemandSpread)
			}
			found = true
		}
	}
	if !found {
		return
	}

	chosen.Tick = s.tick
	s.enqueue(world, player.entity, chosen)
}

// organicCandidates builds the eligible generator list in a fixed order.
// Pure function of the pulse snapshot (plus hook-graph reads) — no RNG here
// so the director's roll count stays easy to reason about.
func (s *EventDirectorSystem) organicCandidates(npcs []eventNPC, playerIdx int,
	rulers map[uint32]uint64, cityFood uint32) []components.GameEvent {

	p := &npcs[playerIdx]
	index := make(map[uint64]int, len(npcs)) // Lookup only
	for i := range npcs {
		index[npcs[i].id] = i
	}
	var out []components.GameEvent

	// Tax demand: the city ruler (not the player) squeezes a poorer subject.
	if p.cityID != 0 && p.hasNeeds {
		if rid := rulers[p.cityID]; rid != 0 && rid != p.id {
			if ri, ok := index[rid]; ok && npcs[ri].hasNeeds && npcs[ri].wealth > p.wealth {
				out = append(out, components.GameEvent{
					Kind: components.EventTaxDemand, ActorID: rid, CityID: p.cityID})
			}
		}
	}

	// Bandit shakedown: the nearest menacing NPC (bandit-jobbed or desperate).
	if p.hasPos && p.hasNeeds {
		best := -1
		var bestD float32
		for i := range npcs {
			if i == playerIdx || !npcs[i].hasPos {
				continue
			}
			menacing := (npcs[i].hasJob && npcs[i].jobID == components.JobBandit) ||
				(npcs[i].hasNeeds && npcs[i].safety < BanditSafetyThreshold)
			if !menacing {
				continue
			}
			dx, dy := npcs[i].x-p.x, npcs[i].y-p.y
			d := dx*dx + dy*dy
			if d > BanditRadius*BanditRadius {
				continue
			}
			if best < 0 || d < bestD { // npcs sorted: ties keep the lowest id
				best, bestD = i, d
			}
		}
		if best >= 0 {
			amount := int32(p.wealth / 2)
			if amount < BanditMinDemand {
				amount = BanditMinDemand
			}
			out = append(out, components.GameEvent{
				Kind: components.EventBanditShakedown, ActorID: npcs[best].id,
				CityID: p.cityID, Amount: amount})
		}
	}

	// Job offer: the city offers its scarcest trade to the unemployed.
	if p.cityID != 0 && !p.hasJob {
		var counts [len(eventTradeJobs)]int
		for i := range npcs {
			if npcs[i].cityID != p.cityID || !npcs[i].hasJob {
				continue
			}
			for j := range eventTradeJobs {
				if npcs[i].jobID == eventTradeJobs[j] {
					counts[j]++
					break
				}
			}
		}
		bestJob := 0
		for j := 1; j < len(eventTradeJobs); j++ {
			if counts[j] < counts[bestJob] {
				bestJob = j
			}
		}
		out = append(out, components.GameEvent{
			Kind: components.EventJobOffer, ActorID: rulers[p.cityID],
			CityID: p.cityID, Amount: int32(eventTradeJobs[bestJob])})
	}

	// Insult: the living NPC holding the deepest grudge against the player.
	if s.hooks != nil && p.id != 0 {
		incoming := s.hooks.GetAllIncomingHooks(p.id)
		ids := make([]uint64, 0, len(incoming))
		for id := range incoming {
			ids = append(ids, id)
		}
		slices.Sort(ids) // Map order is not deterministic; sorted scan is.
		var bestID uint64
		bestVal := 0
		for _, id := range ids {
			v := incoming[id]
			if v >= 0 || id == p.id || !alivenessOf(index, id) {
				continue
			}
			if bestID == 0 || v < bestVal {
				bestID, bestVal = id, v
			}
		}
		if bestID != 0 {
			out = append(out, components.GameEvent{
				Kind: components.EventInsult, ActorID: bestID, CityID: p.cityID})
		}
	}

	// Festival: overflowing granaries spark a celebration.
	if p.cityID != 0 && cityFood >= FestivalFoodThreshold {
		out = append(out, components.GameEvent{
			Kind: components.EventFestival, CityID: p.cityID})
	}

	// Marriage proposal: the fondest unmarried adult proposes to an unmarried
	// adult player (deterministic pick: highest hook, lowest id on ties).
	if s.hooks != nil && p.id != 0 && !p.married && p.age >= MarriageAdultAge {
		incoming := s.hooks.GetAllIncomingHooks(p.id)
		ids := make([]uint64, 0, len(incoming))
		for id := range incoming {
			ids = append(ids, id)
		}
		slices.Sort(ids) // Map order is not deterministic; sorted scan is.
		var bestID uint64
		bestVal := 0
		for _, id := range ids {
			v := incoming[id]
			if v <= 0 || id == p.id {
				continue
			}
			i, ok := index[id]
			if !ok {
				continue // Not among the living
			}
			if npcs[i].married || npcs[i].age < MarriageAdultAge {
				continue
			}
			if v > bestVal { // Ascending id scan: ties keep the lowest id
				bestID, bestVal = id, v
			}
		}
		if bestID != 0 {
			out = append(out, components.GameEvent{
				Kind: components.EventMarriageProposal, ActorID: bestID, CityID: p.cityID})
		}
	}

	// Plot invitation: the bitterest living enemy of the player's city ruler
	// (never the player, never the ruler) recruits the player as conspirator
	// (deterministic pick: most negative hook, lowest id on ties).
	if s.hooks != nil && p.cityID != 0 {
		if rid := rulers[p.cityID]; rid != 0 && rid != p.id {
			incoming := s.hooks.GetAllIncomingHooks(rid)
			ids := make([]uint64, 0, len(incoming))
			for id := range incoming {
				ids = append(ids, id)
			}
			slices.Sort(ids)
			var bestID uint64
			bestVal := 0
			for _, id := range ids {
				v := incoming[id]
				if v > PlotInviteHookThreshold || id == p.id || id == rid {
					continue
				}
				if !alivenessOf(index, id) {
					continue
				}
				if bestID == 0 || v < bestVal {
					bestID, bestVal = id, v
				}
			}
			if bestID != 0 {
				out = append(out, components.GameEvent{
					Kind: components.EventPlotInvitation, ActorID: bestID,
					TargetID: rid, CityID: p.cityID})
			}
		}
	}

	return out
}

// alivenessOf reports whether the identity is in the living-NPC snapshot.
func alivenessOf(index map[uint64]int, id uint64) bool {
	_, ok := index[id]
	return ok
}

// enqueue appends the event with a unique monotonic ID. Performs the one
// deferred structural add (PendingEventsComponent) strictly after all pulse
// queries have finished.
func (s *EventDirectorSystem) enqueue(world *ecs.World, player ecs.Entity, ev components.GameEvent) {
	if !world.Has(player, s.pendID) {
		world.Add(player, s.pendID)
	}
	pe := (*components.PendingEventsComponent)(world.Get(player, s.pendID))
	// Director state is not saved: after a load, resync the counter past any
	// persisted queue IDs so removal-by-ID stays unambiguous.
	for i := range pe.Events {
		if pe.Events[i].ID > s.nextEventID {
			s.nextEventID = pe.Events[i].ID
		}
	}
	s.nextEventID++
	ev.ID = s.nextEventID
	pe.Events = append(pe.Events, ev)
}

// --- Queue access for the UI (called between ticks, outside query loops) ---

// PendingEventList returns a defensive copy of the player's event queue in
// arrival order (oldest first).
func PendingEventList(world *ecs.World, player ecs.Entity) []components.GameEvent {
	if !world.Alive(player) {
		return nil
	}
	pendID := ecs.ComponentID[components.PendingEventsComponent](world)
	if !world.Has(player, pendID) {
		return nil
	}
	pe := (*components.PendingEventsComponent)(world.Get(player, pendID))
	out := make([]components.GameEvent, len(pe.Events))
	copy(out, pe.Events)
	return out
}

// RemovePendingEvent deletes the queued event with the given ID (no-op when
// absent), preserving the order of the remaining events.
func RemovePendingEvent(world *ecs.World, player ecs.Entity, id uint64) {
	if !world.Alive(player) {
		return
	}
	pendID := ecs.ComponentID[components.PendingEventsComponent](world)
	if !world.Has(player, pendID) {
		return
	}
	pe := (*components.PendingEventsComponent)(world.Get(player, pendID))
	for i := range pe.Events {
		if pe.Events[i].ID == id {
			pe.Events = append(pe.Events[:i], pe.Events[i+1:]...)
			return
		}
	}
}

// --- Effect resolution (single deterministic entry point, spec G1) ---

// ResolveEvent applies the effects of picking `choice` on the event and
// returns a chronicle line describing the outcome ("" for a no-op). Callers
// (the UI) must invoke it outside any open query loop; structural mutations
// (CombatMarker/JobComponent adds) happen after all internal queries finish.
//
// Choice indexes per kind:
//
//	EventTaxDemand:        0 Pay, 1 Refuse
//	EventBanditShakedown:  0 Pay, 1 Fight, 2 Flee
//	EventJobOffer:         0 Accept, 1 Decline
//	EventInsult:           0 Laugh it off, 1 Demand apology, 2 Strike them
//	EventFestival:         0 Join, 1 Work through it
//	EventMarriageProposal: 0 Accept, 1 Decline
//	EventPlotInvitation:   0 Join, 1 Refuse, 2 Betray
//	EventWarNews/EventPeaceNews/EventRulerDied: 0 Acknowledge
func ResolveEvent(world *ecs.World, hooks *engine.SparseHookGraph, player ecs.Entity,
	ev *components.GameEvent, choice int) string {

	if ev == nil || !world.Alive(player) {
		return ""
	}
	identID := ecs.ComponentID[components.Identity](world)
	needsID := ecs.ComponentID[components.Needs](world)

	var playerID uint64
	if world.Has(player, identID) {
		playerID = (*components.Identity)(world.Get(player, identID)).ID
	}
	actorName, actorKnown := identityNameOf(world, ev.ActorID)
	city := cityNameOf(world, ev.CityID)

	switch ev.Kind {
	case components.EventTaxDemand:
		actor, actorOK := findNPCByIdentity(world, ev.ActorID)
		if choice == 0 { // Pay
			pay := float32(ev.Amount)
			if pay < 0 {
				pay = 0
			}
			if world.Has(player, needsID) {
				n := (*components.Needs)(world.Get(player, needsID))
				if pay > n.Wealth {
					pay = n.Wealth
				}
				if pay < 0 {
					pay = 0
				}
				n.Wealth -= pay
			} else {
				pay = 0
			}
			if actorOK && world.Has(actor, needsID) {
				(*components.Needs)(world.Get(actor, needsID)).Wealth += pay
			}
			if hooks != nil && ev.ActorID != 0 && playerID != 0 {
				hooks.AddHook(ev.ActorID, playerID, TaxPayHookGain)
			}
			return fmt.Sprintf("You pay %d gold to %s.", int(pay), actorName)
		}
		// Refuse: feud seed toward the ruler, whose authority takes a dent.
		if hooks != nil && ev.ActorID != 0 && playerID != 0 {
			hooks.AddHook(ev.ActorID, playerID, -TaxRefuseHookLoss)
		}
		if actorOK {
			legitID := ecs.ComponentID[components.LegitimacyComponent](world)
			if world.Has(actor, legitID) {
				lg := (*components.LegitimacyComponent)(world.Get(actor, legitID))
				if lg.Score > TaxRefuseLegitimacyLoss {
					lg.Score -= TaxRefuseLegitimacyLoss
				} else {
					lg.Score = 0
				}
			}
		}
		return fmt.Sprintf("You refuse %s's levy. The court will remember.", actorName)

	case components.EventBanditShakedown:
		actor, actorOK := findNPCByIdentity(world, ev.ActorID)
		switch choice {
		case 0: // Pay
			pay := float32(ev.Amount)
			if pay < 0 {
				pay = 0
			}
			if world.Has(player, needsID) {
				n := (*components.Needs)(world.Get(player, needsID))
				if pay > n.Wealth {
					pay = n.Wealth
				}
				if pay < 0 {
					pay = 0
				}
				n.Wealth -= pay
			} else {
				pay = 0
			}
			if actorOK && world.Has(actor, needsID) {
				(*components.Needs)(world.Get(actor, needsID)).Wealth += pay
			}
			return fmt.Sprintf("You hand %d gold to %s and walk away alive.", int(pay), actorName)
		case 1: // Fight: existing combat path via CombatMarker
			attachCombatMarker(world, player, ev.ActorID)
			return fmt.Sprintf("You draw steel against %s!", actorName)
		default: // Flee: teleport-nudge away, lose some rest
			posID := ecs.ComponentID[components.Position](world)
			if world.Has(player, posID) {
				pp := (*components.Position)(world.Get(player, posID))
				dx, dy := FleeDistance, float32(0)
				if actorOK && world.Has(actor, posID) {
					ap := (*components.Position)(world.Get(actor, posID))
					vx, vy := pp.X-ap.X, pp.Y-ap.Y
					if vx != 0 || vy != 0 {
						// Deterministic dominant-axis flee direction.
						ax, ay := vx, vy
						if ax < 0 {
							ax = -ax
						}
						if ay < 0 {
							ay = -ay
						}
						if ax >= ay {
							dx, dy = FleeDistance, 0
							if vx < 0 {
								dx = -FleeDistance
							}
						} else {
							dx, dy = 0, FleeDistance
							if vy < 0 {
								dy = -FleeDistance
							}
						}
					}
				}
				pp.X += dx
				pp.Y += dy
			}
			if world.Has(player, needsID) {
				n := (*components.Needs)(world.Get(player, needsID))
				n.Rest -= FleeRestLoss
				if n.Rest < 0 {
					n.Rest = 0
				}
			}
			return fmt.Sprintf("You flee from %s, heart pounding.", actorName)
		}

	case components.EventJobOffer:
		if choice == 0 { // Accept: JobComponent swap
			jobCompID := ecs.ComponentID[components.JobComponent](world)
			if !world.Has(player, jobCompID) {
				world.Add(player, jobCompID)
			}
			j := (*components.JobComponent)(world.Get(player, jobCompID))
			j.JobID = uint8(ev.Amount)
			j.EmployerID = ev.ActorID
			return fmt.Sprintf("You take up work as a %s in %s.", eventJobTitle(uint8(ev.Amount)), city)
		}
		return "You decline the offer of work."

	case components.EventInsult:
		switch choice {
		case 0:
			return fmt.Sprintf("You laugh off %s's insult.", actorName)
		case 1: // Contest: higher STR + positive incoming hooks wins opinion
			actor, actorOK := findNPCByIdentity(world, ev.ActorID)
			playerScore := int(strengthOf(world, player)) + positiveIncomingScore(hooks, playerID)
			actorScore := positiveIncomingScore(hooks, ev.ActorID)
			if actorOK {
				actorScore += int(strengthOf(world, actor))
			}
			if hooks != nil && ev.ActorID != 0 && playerID != 0 {
				if playerScore >= actorScore {
					hooks.AddHook(ev.ActorID, playerID, InsultContestWinGain)
					return fmt.Sprintf("%s backs down before you. The crowd approves.", actorName)
				}
				hooks.AddHook(ev.ActorID, playerID, -InsultContestLossPenalty)
			}
			return fmt.Sprintf("%s sneers — your demand only feeds the grudge.", actorName)
		default: // Strike them: existing combat path
			attachCombatMarker(world, player, ev.ActorID)
			return fmt.Sprintf("You strike %s!", actorName)
		}

	case components.EventFestival:
		if choice == 0 { // Join: rest + mutual hooks with 2 random citizens
			if world.Has(player, needsID) {
				n := (*components.Needs)(world.Get(player, needsID))
				n.Rest += FestivalRestGain
				if n.Rest > 100 {
					n.Rest = 100
				}
			}
			ids := cityCitizenIDs(world, ev.CityID, playerID)
			if len(ids) > 0 && hooks != nil && playerID != 0 {
				i1 := engine.GetRandomInt() % len(ids)
				i2 := i1
				if len(ids) > 1 {
					i2 = engine.GetRandomInt() % len(ids)
					if i2 == i1 {
						i2 = (i1 + 1) % len(ids)
					}
				}
				hooks.AddHook(playerID, ids[i1], FestivalHookGain)
				hooks.AddHook(ids[i1], playerID, FestivalHookGain)
				if i2 != i1 {
					hooks.AddHook(playerID, ids[i2], FestivalHookGain)
					hooks.AddHook(ids[i2], playerID, FestivalHookGain)
				}
			}
			return fmt.Sprintf("You join the festival of %s. New friends, full heart.", city)
		}
		if world.Has(player, needsID) {
			(*components.Needs)(world.Get(player, needsID)).Wealth += FestivalWorkGold
		}
		return fmt.Sprintf("You work through the festival (+%d gold).", int(FestivalWorkGold))

	case components.EventMarriageProposal:
		if choice == 0 { // Accept: the standard marriage path seals it
			actor, actorOK := findNPCByIdentity(world, ev.ActorID)
			if !actorOK {
				return fmt.Sprintf("%s is gone; the proposal dies unanswered.", actorName)
			}
			if err := Marry(world, hooks, player, actor); err != nil {
				return fmt.Sprintf("The wedding falls through: %s.", err.Error())
			}
			return fmt.Sprintf("You wed %s. May your houses prosper.", actorName)
		}
		// Decline: spurned affection curdles.
		if hooks != nil && ev.ActorID != 0 && playerID != 0 {
			hooks.AddHook(ev.ActorID, playerID, -MarriageDeclineHookLoss)
		}
		return fmt.Sprintf("You decline %s's proposal as gently as you can.", actorName)

	case components.EventPlotInvitation:
		actor, actorOK := findNPCByIdentity(world, ev.ActorID)
		switch choice {
		case 0: // Join: the plotter gains a plot (if new) plus a conspirator hook
			plotCompID := ecs.ComponentID[components.PlotComponent](world)
			if actorOK && !world.Has(actor, plotCompID) {
				if target, targetOK := findNPCByIdentity(world, ev.TargetID); targetOK {
					if err := StartPlot(world, actor, target, components.PlotSeizeRule); err != nil {
						return fmt.Sprintf("The conspiracy of %s collapses before it begins.", actorName)
					}
				}
			}
			// Conspirator bond: PlotSystem counts positive incoming hooks on
			// the plotter as fellow conspirators (Power +5 each).
			if hooks != nil && ev.ActorID != 0 && playerID != 0 {
				hooks.AddHook(playerID, ev.ActorID, PlotJoinHookGain)
			}
			return fmt.Sprintf("You join %s's conspiracy in the shadows.", actorName)
		case 1: // Refuse: you keep your hands — and the secret — to yourself.
			return fmt.Sprintf("You want no part of %s's scheming.", actorName)
		default: // Betray: the same exposure path PlotSystem walks on discovery
			exposePlotter(world, hooks, actor, actorOK, ev.ActorID, ev.TargetID)
			return fmt.Sprintf("You betray %s's plot to its target.", actorName)
		}

	case components.EventWarNews:
		return fmt.Sprintf("War! Your realm now fights Realm %d.", ev.Amount)

	case components.EventPeaceNews:
		return fmt.Sprintf("Peace returns between your realm and Realm %d.", ev.Amount)

	case components.EventRulerDied:
		who := actorName
		if !actorKnown {
			who = "The old ruler"
		}
		line := fmt.Sprintf("%s, ruler of %s, is dead.", who, city)
		if ev.Flag == components.EventFlagThroneClaimable {
			line += " The throne sits empty."
		}
		return line
	}
	return ""
}

// --- ResolveEvent helpers (all queries fully consumed) ---

// findNPCByIdentity locates the living NPC entity carrying Identity.ID == id.
func findNPCByIdentity(world *ecs.World, id uint64) (ecs.Entity, bool) {
	if id == 0 {
		return ecs.Entity{}, false
	}
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)

	var found ecs.Entity
	ok := false
	f := ecs.All(npcID, identID)
	q := world.Query(&f)
	for q.Next() {
		if ok {
			continue
		}
		if (*components.Identity)(q.Get(identID)).ID == id {
			found = q.Entity()
			ok = true
		}
	}
	return found, ok
}

// exposePlotter applies the discovery penalties PlotSystem levies on an
// exposed plot — same constants, same channels: the plot is marked Exposed
// (PlotSystem unravels it next cycle), a plot-carrying plotter loses
// plotExposureLegitimacyLoss legitimacy, and the target puts the
// plotExposureHookPenalty grudge hook on the plotter. A betrayed invitation
// with no live plot still teaches the target who schemed (hook only).
func exposePlotter(world *ecs.World, hooks *engine.SparseHookGraph,
	plotter ecs.Entity, plotterOK bool, plotterID, targetID uint64) {

	hadPlot := false
	if plotterOK {
		plotCompID := ecs.ComponentID[components.PlotComponent](world)
		if world.Has(plotter, plotCompID) {
			(*components.PlotComponent)(world.Get(plotter, plotCompID)).Exposed = true
			hadPlot = true
		}
	}
	if hadPlot {
		legitID := ecs.ComponentID[components.LegitimacyComponent](world)
		if world.Has(plotter, legitID) {
			legit := (*components.LegitimacyComponent)(world.Get(plotter, legitID))
			if legit.Score > plotExposureLegitimacyLoss {
				legit.Score -= plotExposureLegitimacyLoss
			} else {
				legit.Score = 0
			}
		}
	}
	if hooks != nil && plotterID != 0 && targetID != 0 {
		hooks.AddHook(targetID, plotterID, plotExposureHookPenalty)
	}
}

// attachCombatMarker starts combat via the existing CombatSystem path: the
// player gains a CombatMarker aimed at the target's Identity.ID (mirrors
// player_input.go / blood_feud.go). Structural add outside all query loops.
func attachCombatMarker(world *ecs.World, player ecs.Entity, targetIdentity uint64) {
	if targetIdentity == 0 {
		return
	}
	combatID := ecs.ComponentID[components.CombatMarker](world)
	if !world.Has(player, combatID) {
		world.Add(player, combatID)
	}
	(*components.CombatMarker)(world.Get(player, combatID)).TargetID = targetIdentity
}

// strengthOf reads Genome.Strength (0 when the entity has no genome).
func strengthOf(world *ecs.World, ent ecs.Entity) uint8 {
	genID := ecs.ComponentID[components.GenomeComponent](world)
	if !world.Alive(ent) || !world.Has(ent, genID) {
		return 0
	}
	return (*components.GenomeComponent)(world.Get(ent, genID)).Strength
}

// cityCitizenIDs returns the sorted Identity.IDs of the city's living NPCs,
// excluding the given identity.
func cityCitizenIDs(world *ecs.World, cityID uint32, exclude uint64) []uint64 {
	if cityID == 0 {
		return nil
	}
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	var ids []uint64
	f := ecs.All(npcID, identID, affID)
	q := world.Query(&f)
	for q.Next() {
		if (*components.Affiliation)(q.Get(affID)).CityID != cityID {
			continue
		}
		id := (*components.Identity)(q.Get(identID)).ID
		if id != exclude {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

// identityNameOf resolves a living NPC's display name by Identity.ID.
func identityNameOf(world *ecs.World, id uint64) (string, bool) {
	if id == 0 {
		return "someone", false
	}
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)

	name := ""
	f := ecs.All(npcID, identID)
	q := world.Query(&f)
	for q.Next() {
		ident := (*components.Identity)(q.Get(identID))
		if ident.ID == id && ident.Name != "" {
			name = ident.Name
		}
	}
	if name == "" {
		return fmt.Sprintf("Citizen %d", id), false
	}
	return name, true
}

// cityNameOf resolves a settlement's display name (mirrors ui.CityName; the
// systems package cannot import ui without a cycle).
func cityNameOf(world *ecs.World, cityID uint32) string {
	if cityID == 0 {
		return "the wilds"
	}
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)

	name := fmt.Sprintf("City %d", cityID)
	f := ecs.All(villageID, identID)
	q := world.Query(&f)
	for q.Next() {
		ident := (*components.Identity)(q.Get(identID))
		if uint32(ident.ID) == cityID && ident.Name != "" {
			name = ident.Name
		}
	}
	return name
}

// eventJobTitle names the offered trade (ui.JobName is unreachable from here).
func eventJobTitle(id uint8) string {
	switch id {
	case components.JobFarmer:
		return "farmer"
	case components.JobLumberjack:
		return "lumberjack"
	case components.JobArtisan:
		return "artisan"
	case components.JobGuard:
		return "guard"
	case components.JobBuilder:
		return "builder"
	case components.JobMiner:
		return "miner"
	default:
		return "laborer"
	}
}
