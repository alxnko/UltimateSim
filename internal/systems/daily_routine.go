package systems

import (
	"math"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (P5/L1): Visible NPC daily life.
// Spec: docs/superpowers/specs/2026-08-06-grand-strategy-playable.md
//
// DailyRoutineSystem makes employed, city-bound citizens look busy at a
// glance: morning streams of workers leaving the village toward their job
// anchors, working clusters at farms/forest/mine during the day, an evening
// return to the village center, and night stillness. It steers movement the
// same way WanderSystem does — by writing the entity's Path (a straight-line
// node) and letting MovementSystem resolve kinematics at existing walk
// speeds — so no second velocity convention is introduced.
//
// Ownership: the system tags qualifying NPCs (JobComponent with a real job +
// Affiliation.CityID != 0) with components.RoutineComponent. WanderSystem
// excludes routine-tagged NPCs, so wandering remains only for the
// unemployed/wilderness. NPCs that are Possessed, under a PlayerOrder, in
// combat, or with a path in flight are never touched.

// Day phases derived from the global tick (engine.TicksPerDay per phase).
const (
	RoutinePhaseMorning uint8 = 0 // travel to the job anchor
	RoutinePhaseDay     uint8 = 1 // work: idle-wiggle around the anchor
	RoutinePhaseEvening uint8 = 2 // return to the village center
	RoutinePhaseNight   uint8 = 3 // rest: zero velocity near the center
)

const (
	routineAssignInterval uint64  = 120  // ticks between tag/anchor passes
	routineEvalInterval   uint64  = 30   // per-NPC steering cadence
	routineArriveSq       float32 = 2.25 // ~1.5 tiles: "at the anchor"
	routineRestSq         float32 = 6.25 // ~2.5 tiles: close enough to rest
	guardPatrolRadius     float32 = 8.0  // patrol ring radius (spec L1)
	guardRotateTicks      uint64  = 150  // ring target advances every 2.5s
	wiggleWindowTicks     uint64  = 90   // idle-wiggle target refresh window
	maxAnchorScanRadius   int     = 48   // ring-scan bound for biome anchors
)

// RoutinePhase derives the current day phase from the global tick counter.
func RoutinePhase(tick uint64) uint8 {
	return uint8((tick / uint64(engine.TicksPerDay)) % 4)
}

type DailyRoutineSystem struct {
	mapGrid   *engine.MapGrid
	calendar  *engine.Calendar
	filter    ecs.Filter
	tickStamp uint64
	centers   map[uint32]components.Position // CityID -> village center (key lookups only)
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *DailyRoutineSystem) IsExpensive() bool {
	return true
}

// NewDailyRoutineSystem creates the daily-life steering system.
func NewDailyRoutineSystem(world *ecs.World, mapGrid *engine.MapGrid, calendar *engine.Calendar) *DailyRoutineSystem {
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)
	pathID := ecs.ComponentID[components.Path](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	routineID := ecs.ComponentID[components.RoutineComponent](world)

	// Mirror WanderSystem's guard exclusions: possession and direct player
	// orders own the body; combat steering wins over routines.
	possessedID := ecs.ComponentID[components.Possessed](world)
	orderID := ecs.ComponentID[components.PlayerOrderComponent](world)
	combatID := ecs.ComponentID[components.CombatMarker](world)

	mask := ecs.All(posID, idID, pathID, jobID, affID, routineID).
		Without(possessedID, orderID, combatID)

	return &DailyRoutineSystem{
		mapGrid:  mapGrid,
		calendar: calendar,
		filter:   &mask,
		centers:  make(map[uint32]components.Position),
	}
}

// Update executes the system logic per tick (PhaseAI).
func (s *DailyRoutineSystem) Update(world *ecs.World) {
	s.tickStamp++
	tick := s.tickStamp
	if s.calendar != nil {
		tick = s.calendar.Ticks
	}

	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)
	pathID := ecs.ComponentID[components.Path](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	routineID := ecs.ComponentID[components.RoutineComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)

	// Village centers keyed by CityID (uint32(village Identity.ID), the
	// CityBinder convention). Rebuilt per tick — a handful of entities.
	clear(s.centers)
	vq := world.Query(ecs.All(villageID, posID, idID))
	for vq.Next() {
		p := (*components.Position)(vq.Get(posID))
		vid := (*components.Identity)(vq.Get(idID))
		s.centers[uint32(vid.ID)] = *p
	}
	if len(s.centers) == 0 {
		return
	}

	// Periodic structural pass: tag new routine NPCs, refresh stale anchors,
	// untag NPCs that no longer qualify.
	if tick%routineAssignInterval == 0 {
		s.assignRoutines(world)
	}

	phase := RoutinePhase(tick)

	query := world.Query(s.filter)
	for query.Next() {
		ident := (*components.Identity)(query.Get(idID))
		// Throttle per NPC on a stable identity offset (~2 evals/sec).
		if (tick+ident.ID)%routineEvalInterval != 0 {
			continue
		}

		// A routine-managed citizen never counts as a "migrating group at
		// rest": suppress spontaneous settlement founding while we own them.
		if query.Has(slID) {
			sl := (*components.SettlementLogic)(query.Get(slID))
			sl.TicksAtZeroVelocity = 0
		}

		path := (*components.Path)(query.Get(pathID))
		if path.HasPath {
			continue // pathfinding in flight or another system's walk: never fight it
		}

		aff := (*components.Affiliation)(query.Get(affID))
		center, ok := s.centers[aff.CityID]
		if !ok {
			continue // city died; the assign pass will untag eventually
		}

		rc := (*components.RoutineComponent)(query.Get(routineID))
		pos := (*components.Position)(query.Get(posID))
		rc.Phase = phase

		var tx, ty float32
		resting := false
		switch phase {
		case RoutinePhaseMorning, RoutinePhaseDay:
			if rc.JobSeen == components.JobGuard {
				// Guards orbit the village center: the ring target advances
				// with the tick so patrols are visibly moving.
				step := float64(tick / guardRotateTicks)
				ang := routineAngle(ident.ID) + step*(math.Pi/6)
				sin, cos := math.Sincos(ang)
				tx = center.X + guardPatrolRadius*float32(cos)
				ty = center.Y + guardPatrolRadius*float32(sin)
			} else {
				tx, ty = rc.AnchorX, rc.AnchorY
				dxa := pos.X - tx
				dya := pos.Y - ty
				if phase == RoutinePhaseDay && dxa*dxa+dya*dya <= routineArriveSq {
					// At work: small deterministic idle-wiggle around the
					// anchor so crowds don't stack on one pixel.
					window := tick / wiggleWindowTicks
					tx += routineJitter(ident.ID, window*2+1, 1.2)
					ty += routineJitter(ident.ID, window*2+2, 1.2)
				}
			}
		case RoutinePhaseEvening, RoutinePhaseNight:
			// Home is the village center plus a stable per-NPC scatter.
			tx = center.X + routineJitter(ident.ID, 5, 2.0)
			ty = center.Y + routineJitter(ident.ID, 6, 2.0)
			if phase == RoutinePhaseNight {
				dxh := pos.X - tx
				dyh := pos.Y - ty
				if dxh*dxh+dyh*dyh <= routineRestSq {
					resting = true // rest: no path, MovementSystem zeroes velocity
				}
			}
		}
		if resting {
			continue
		}

		tx, ty = s.clampToLand(tx, ty, center)
		dx := tx - pos.X
		dy := ty - pos.Y
		if dx*dx+dy*dy <= 0.09 {
			continue // already on target
		}

		// Straight-line walk: MovementSystem resolves velocity at the
		// existing biome walk speeds (same convention wander paths use).
		path.Nodes = []components.Position{{X: tx, Y: ty}}
		path.HasPath = true
		path.TargetX = tx
		path.TargetY = ty
	}
}

// routineCandidate defers structural mutation outside open queries.
type routineCandidate struct {
	entity ecs.Entity
	npcID  uint64
	job    uint8
	cityID uint32
}

// cityScan caches the nearest forest/mountain work tiles per city per pass.
type cityScan struct {
	forest    components.Position
	mount     components.Position
	hasForest bool
	hasMount  bool
}

// assignRoutines tags qualifying NPCs, refreshes anchors after career
// changes, and untags NPCs that lost job or city. All structural mutations
// are applied after every query has run to exhaustion.
func (s *DailyRoutineSystem) assignRoutines(world *ecs.World) {
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)
	pathID := ecs.ComponentID[components.Path](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	routineID := ecs.ComponentID[components.RoutineComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)
	structID := ecs.ComponentID[components.StructureComponent](world)

	// Completed Farm structures anchor farmers (deterministic query order).
	var farms []components.Position
	fq := world.Query(ecs.All(structID, posID))
	for fq.Next() {
		st := (*components.StructureComponent)(fq.Get(structID))
		if uint8(st.StructureType) == components.StructureFarm {
			farms = append(farms, *(*components.Position)(fq.Get(posID)))
		}
	}

	scans := make(map[uint32]*cityScan)

	// Pass 1: collect untagged employed city citizens.
	var adds []routineCandidate
	addMask := ecs.All(npcID, jobID, affID, idID, posID, pathID).Without(routineID)
	nq := world.Query(&addMask)
	for nq.Next() {
		job := (*components.JobComponent)(nq.Get(jobID))
		aff := (*components.Affiliation)(nq.Get(affID))
		if job.JobID == components.JobNone || aff.CityID == 0 {
			continue
		}
		if _, ok := s.centers[aff.CityID]; !ok {
			continue
		}
		ident := (*components.Identity)(nq.Get(idID))
		adds = append(adds, routineCandidate{
			entity: nq.Entity(),
			npcID:  ident.ID,
			job:    job.JobID,
			cityID: aff.CityID,
		})
	}

	// Pass 2: existing routine NPCs — refresh stale anchors inline
	// (non-structural), collect the no-longer-qualifying for removal.
	var removals []ecs.Entity
	rq := world.Query(ecs.All(routineID, jobID, affID, idID))
	for rq.Next() {
		job := (*components.JobComponent)(rq.Get(jobID))
		aff := (*components.Affiliation)(rq.Get(affID))
		if job.JobID == components.JobNone || aff.CityID == 0 {
			removals = append(removals, rq.Entity())
			continue
		}
		center, ok := s.centers[aff.CityID]
		if !ok {
			continue
		}
		rc := (*components.RoutineComponent)(rq.Get(routineID))
		if rc.JobSeen != job.JobID {
			ident := (*components.Identity)(rq.Get(idID))
			rc.AnchorX, rc.AnchorY = s.computeAnchor(job.JobID, ident.ID, aff.CityID, center, farms, scans)
			rc.JobSeen = job.JobID
		}
	}

	// Pass 3: routine NPCs that lost the JobComponent entirely.
	lostMask := ecs.All(routineID).Without(jobID)
	lq := world.Query(&lostMask)
	for lq.Next() {
		removals = append(removals, lq.Entity())
	}

	// Apply structural mutations outside all queries (deterministic order).
	for _, c := range adds {
		center := s.centers[c.cityID]
		ax, ay := s.computeAnchor(c.job, c.npcID, c.cityID, center, farms, scans)
		world.Add(c.entity, routineID)
		rc := (*components.RoutineComponent)(world.Get(c.entity, routineID))
		rc.AnchorX = ax
		rc.AnchorY = ay
		rc.JobSeen = c.job
		rc.Phase = RoutinePhaseMorning
	}
	for _, e := range removals {
		if world.Alive(e) {
			world.Remove(e, routineID)
		}
	}
}

// computeAnchor derives the deterministic job anchor for one NPC from its
// city's village position, job, and Identity.ID jitter.
func (s *DailyRoutineSystem) computeAnchor(job uint8, npcID uint64, cityID uint32, center components.Position, farms []components.Position, scans map[uint32]*cityScan) (float32, float32) {
	h := routineHash(npcID, uint64(job))
	switch job {
	case components.JobFarmer:
		if fx, fy, ok := nearestFarm(farms, center); ok {
			return s.clampToLand(fx+routineJitter(npcID, 11, 1.5), fy+routineJitter(npcID, 12, 1.5), center)
		}
		// No farm yet: fertile field ring 6-12 tiles from the center.
		return s.ringAnchor(center, 6, 12, h)
	case components.JobLumberjack:
		sc := s.cityScanFor(cityID, center, scans)
		if sc.hasForest {
			return s.clampToLand(sc.forest.X+routineJitter(npcID, 13, 2.0), sc.forest.Y+routineJitter(npcID, 14, 2.0), center)
		}
		return s.ringAnchor(center, 6, 12, h)
	case components.JobMiner:
		sc := s.cityScanFor(cityID, center, scans)
		if sc.hasMount {
			return s.clampToLand(sc.mount.X+routineJitter(npcID, 15, 2.0), sc.mount.Y+routineJitter(npcID, 16, 2.0), center)
		}
		return s.ringAnchor(center, 6, 12, h)
	case components.JobArtisan, components.JobBuilder:
		return s.ringAnchor(center, 2, 4, h)
	case components.JobGuard:
		// Guards orbit the center; the rotating ring target is derived at
		// steering time, so the stored anchor is the center itself.
		return center.X, center.Y
	default:
		return s.ringAnchor(center, 2, 5, h)
	}
}

// cityScanFor lazily ring-scans outward from the village center for the
// nearest forest and mountain/bare tiles, cached per city per assign pass.
// Fixed scan order keeps it deterministic.
func (s *DailyRoutineSystem) cityScanFor(cityID uint32, center components.Position, scans map[uint32]*cityScan) *cityScan {
	if sc, ok := scans[cityID]; ok {
		return sc
	}
	sc := &cityScan{}
	cx, cy := int(center.X), int(center.Y)
	for r := 2; r <= maxAnchorScanRadius && (!sc.hasForest || !sc.hasMount); r++ {
		s.scanRing(cx, cy, r, sc)
	}
	scans[cityID] = sc
	return sc
}

func (s *DailyRoutineSystem) scanRing(cx, cy, r int, sc *cityScan) {
	check := func(x, y int) {
		if x < 0 || y < 0 || x >= s.mapGrid.Width || y >= s.mapGrid.Height {
			return
		}
		switch s.mapGrid.Tiles[y*s.mapGrid.Width+x].BiomeID {
		case engine.BiomeTemperateDeciduousForest, engine.BiomeTemperateRainForest,
			engine.BiomeTropicalSeasonalForest, engine.BiomeTropicalRainForest:
			if !sc.hasForest {
				sc.forest = components.Position{X: float32(x), Y: float32(y)}
				sc.hasForest = true
			}
		case engine.BiomeMountain, engine.BiomeBare:
			if !sc.hasMount {
				sc.mount = components.Position{X: float32(x), Y: float32(y)}
				sc.hasMount = true
			}
		}
	}
	for dx := -r; dx <= r; dx++ {
		check(cx+dx, cy-r)
		check(cx+dx, cy+r)
	}
	for dy := -r + 1; dy <= r-1; dy++ {
		check(cx-r, cy+dy)
		check(cx+r, cy+dy)
	}
}

// ringAnchor places an anchor on a ring [minR, maxR] around the center at a
// hash-derived angle, walking inward until the tile is on land.
func (s *DailyRoutineSystem) ringAnchor(center components.Position, minR, maxR int, h uint64) (float32, float32) {
	ang := float64(h%3600) * math.Pi / 1800.0
	radius := float32(minR)
	if maxR > minR {
		radius += float32((h>>16)%uint64((maxR-minR)*10+1)) / 10.0
	}
	sin, cos := math.Sincos(ang)
	for radius >= 1 {
		x := center.X + radius*float32(cos)
		y := center.Y + radius*float32(sin)
		xi, yi := int(x), int(y)
		if xi >= 0 && yi >= 0 && xi < s.mapGrid.Width && yi < s.mapGrid.Height &&
			s.mapGrid.Tiles[yi*s.mapGrid.Width+xi].BiomeID != engine.BiomeOcean {
			return x, y
		}
		radius--
	}
	return center.X, center.Y
}

// clampToLand bounds a target to the map and falls back to the village
// center when the target tile is ocean (routine walks are straight lines).
func (s *DailyRoutineSystem) clampToLand(x, y float32, fallback components.Position) (float32, float32) {
	maxX := float32(s.mapGrid.Width - 1)
	maxY := float32(s.mapGrid.Height - 1)
	if x < 0 {
		x = 0
	} else if x > maxX {
		x = maxX
	}
	if y < 0 {
		y = 0
	} else if y > maxY {
		y = maxY
	}
	if s.mapGrid.GetTile(int(x), int(y)).BiomeID == engine.BiomeOcean {
		return fallback.X, fallback.Y
	}
	return x, y
}

// nearestFarm returns the closest farm to the village center within 30 tiles.
// Ties resolve to the first in query order — deterministic.
func nearestFarm(farms []components.Position, center components.Position) (float32, float32, bool) {
	const maxFarmDistSq = 900
	best := -1
	var bestD float32
	for i := range farms {
		dx := farms[i].X - center.X
		dy := farms[i].Y - center.Y
		d := dx*dx + dy*dy
		if d <= maxFarmDistSq && (best < 0 || d < bestD) {
			best = i
			bestD = d
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	return farms[best].X, farms[best].Y, true
}

// routineHash mixes a stable NPC identity with a salt via the splitmix64
// finalizer. Pure integer math: deterministic across runs and platforms.
func routineHash(id uint64, salt uint64) uint64 {
	h := id*0x9E3779B97F4A7C15 + salt*0xBF58476D1CE4E5B9 + 0x94D049BB133111EB
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	return h
}

// routineJitter maps (id, salt) to a deterministic offset in [-span, span].
func routineJitter(id uint64, salt uint64, span float32) float32 {
	h := routineHash(id, salt)
	return (float32(h%2001)/1000.0 - 1.0) * span
}

// routineAngle gives each guard a stable patrol phase offset on the ring.
func routineAngle(id uint64) float64 {
	return float64(routineHash(id, 777)%3600) * math.Pi / 1800.0
}
