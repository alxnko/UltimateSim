package systems_test

import (
	"math"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (P5/L1): deterministic tests for DailyRoutineSystem.

// newLandGrid builds a grid where every tile is walkable grassland
// (NewMapGrid zero-value biome is BiomeOcean, which would reject anchors).
func newLandGrid(w, h int) *engine.MapGrid {
	grid := engine.NewMapGrid(w, h)
	for i := range grid.Tiles {
		grid.Tiles[i].BiomeID = engine.BiomeGrassland
	}
	return grid
}

func spawnRoutineVillage(world *ecs.World, x, y float32, cityID uint64) {
	villageID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)
	v := world.NewEntity(villageID, posID, idID)
	p := (*components.Position)(world.Get(v, posID))
	p.X, p.Y = x, y
	ident := (*components.Identity)(world.Get(v, idID))
	ident.ID = cityID
}

func spawnWorker(world *ecs.World, id uint64, cityID uint32, job uint8, x, y float32) ecs.Entity {
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	idID := ecs.ComponentID[components.Identity](world)
	needsID := ecs.ComponentID[components.Needs](world)
	pathID := ecs.ComponentID[components.Path](world)
	npcID := ecs.ComponentID[components.NPC](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)

	e := world.NewEntity(posID, velID, idID, needsID, pathID, npcID, affID, jobID, slID)
	p := (*components.Position)(world.Get(e, posID))
	p.X, p.Y = x, y
	ident := (*components.Identity)(world.Get(e, idID))
	ident.ID = id
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.CityID = cityID
	j := (*components.JobComponent)(world.Get(e, jobID))
	j.JobID = job
	n := (*components.Needs)(world.Get(e, needsID))
	n.Food = 1000 // Keep metabolic concerns out of routine tests
	return e
}

// runTicks advances the shared calendar and the system together.
func runTicks(sys *systems.DailyRoutineSystem, cal *engine.Calendar, world *ecs.World, n int) {
	for i := 0; i < n; i++ {
		cal.Ticks++
		sys.Update(world)
	}
}

func dist(x1, y1, x2, y2 float32) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
}

// buildAnchorWorld constructs the deterministic anchor scenario: one village
// at (32,32), a forest patch east, a mountain patch south, and one worker of
// each interesting job.
func buildAnchorWorld() (*ecs.World, *engine.MapGrid, *engine.Calendar, *systems.DailyRoutineSystem, map[uint64]ecs.Entity) {
	world := ecs.NewWorld()
	grid := newLandGrid(64, 64)
	// Forest patch ~8 tiles east of the village center.
	for y := 31; y <= 33; y++ {
		for x := 40; x <= 42; x++ {
			grid.Tiles[y*64+x].BiomeID = engine.BiomeTemperateDeciduousForest
		}
	}
	// Mountain patch ~8 tiles south of the village center.
	for y := 40; y <= 42; y++ {
		for x := 31; x <= 33; x++ {
			grid.Tiles[y*64+x].BiomeID = engine.BiomeMountain
		}
	}
	cal := engine.NewCalendar()
	sys := systems.NewDailyRoutineSystem(&world, grid, cal)

	spawnRoutineVillage(&world, 32, 32, 500)
	ents := map[uint64]ecs.Entity{
		1001: spawnWorker(&world, 1001, 500, components.JobFarmer, 32, 32),
		1002: spawnWorker(&world, 1002, 500, components.JobFarmer, 32, 32),
		1003: spawnWorker(&world, 1003, 500, components.JobLumberjack, 32, 32),
		1004: spawnWorker(&world, 1004, 500, components.JobMiner, 32, 32),
		1005: spawnWorker(&world, 1005, 500, components.JobArtisan, 32, 32),
		1006: spawnWorker(&world, 1006, 500, components.JobGuard, 32, 32),
	}
	return &world, grid, cal, sys, ents
}

func TestDailyRoutine_AnchorDeterminismAndPlacement(t *testing.T) {
	type anchor struct{ x, y float32 }
	collect := func() map[uint64]anchor {
		world, _, cal, sys, ents := buildAnchorWorld()
		routineID := ecs.ComponentID[components.RoutineComponent](world)
		runTicks(sys, cal, world, 120) // assign pass fires at tick 120
		out := make(map[uint64]anchor)
		for id, e := range ents {
			if !world.Has(e, routineID) {
				t.Fatalf("worker %d should have been tagged with RoutineComponent", id)
			}
			rc := (*components.RoutineComponent)(world.Get(e, routineID))
			out[id] = anchor{rc.AnchorX, rc.AnchorY}
		}
		return out
	}

	run1 := collect()
	run2 := collect()

	for id, a1 := range run1 {
		a2 := run2[id]
		if a1 != a2 {
			t.Errorf("anchor for NPC %d not deterministic: run1=(%f,%f) run2=(%f,%f)",
				id, a1.x, a1.y, a2.x, a2.y)
		}
	}

	// Farmers (no farm structure): fertile ring 6-12 tiles from the center.
	for _, id := range []uint64{1001, 1002} {
		d := dist(run1[id].x, run1[id].y, 32, 32)
		if d < 5.0 || d > 13.0 {
			t.Errorf("farmer %d anchor should sit on the 6-12 ring, got dist %f", id, d)
		}
	}
	// Identity jitter separates coworkers.
	if run1[1001] == run1[1002] {
		t.Errorf("two farmers should not share the exact same anchor: %+v", run1[1001])
	}
	// Lumberjack anchors at the nearest forest tile (40,32) + jitter <= 2.
	if d := dist(run1[1003].x, run1[1003].y, 40, 32); d > 3.0 {
		t.Errorf("lumberjack anchor should hug the forest tile (40,32), got dist %f", d)
	}
	// Miner anchors at the nearest mountain tile (32,40) + jitter <= 2.
	if d := dist(run1[1004].x, run1[1004].y, 32, 40); d > 3.0 {
		t.Errorf("miner anchor should hug the mountain tile (32,40), got dist %f", d)
	}
	// Artisan works the village-center ring 2-4.
	if d := dist(run1[1005].x, run1[1005].y, 32, 32); d < 1.9 || d > 4.1 {
		t.Errorf("artisan anchor should sit on the 2-4 ring, got dist %f", d)
	}
	// Guard anchor is the village center itself (patrol ring is derived).
	if run1[1006].x != 32 || run1[1006].y != 32 {
		t.Errorf("guard anchor should be the village center, got (%f,%f)", run1[1006].x, run1[1006].y)
	}
}

func TestDailyRoutine_PhaseTransitionsMoveTargets(t *testing.T) {
	world, _, cal, sys, ents := buildAnchorWorld()
	farmer := ents[1001]
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)
	routineID := ecs.ComponentID[components.RoutineComponent](world)

	(*components.SettlementLogic)(world.Get(farmer, slID)).TicksAtZeroVelocity = 500

	// Morning (phase 0, ticks 0-599): work-travel toward the job anchor.
	// Component pointers are fetched AFTER this run: the assign pass adds
	// RoutineComponent (a structural change) which relocates the entity's
	// component memory to a new archetype.
	runTicks(sys, cal, world, 150)
	path := (*components.Path)(world.Get(farmer, pathID))
	pos := (*components.Position)(world.Get(farmer, posID))
	sl := (*components.SettlementLogic)(world.Get(farmer, slID))
	rc := (*components.RoutineComponent)(world.Get(farmer, routineID))
	if !path.HasPath {
		t.Fatalf("morning: farmer should be walking to work")
	}
	if path.TargetX != rc.AnchorX || path.TargetY != rc.AnchorY {
		t.Errorf("morning target should be the anchor (%f,%f), got (%f,%f)",
			rc.AnchorX, rc.AnchorY, path.TargetX, path.TargetY)
	}
	if sl.TicksAtZeroVelocity != 0 {
		t.Errorf("routine NPCs must not accumulate settlement idle ticks, got %d", sl.TicksAtZeroVelocity)
	}

	// Day (phase 1, ticks 600-1199): arrived at the anchor, wiggle stays local.
	pos.X, pos.Y = rc.AnchorX, rc.AnchorY
	path.HasPath = false
	path.Nodes = nil
	cal.Ticks = 900
	runTicks(sys, cal, world, 30)
	if rc.Phase != systems.RoutinePhaseDay {
		t.Errorf("day: routine phase should be %d, got %d", systems.RoutinePhaseDay, rc.Phase)
	}
	if path.HasPath {
		if d := dist(path.TargetX, path.TargetY, rc.AnchorX, rc.AnchorY); d > 2.0 {
			t.Errorf("day wiggle target should stay near the anchor, got dist %f", d)
		}
	}

	// Evening (phase 2, ticks 1200-1799): return to the village center.
	pos.X, pos.Y = rc.AnchorX, rc.AnchorY
	path.HasPath = false
	path.Nodes = nil
	cal.Ticks = 1500
	runTicks(sys, cal, world, 30)
	if !path.HasPath {
		t.Fatalf("evening: farmer should be walking home")
	}
	if d := dist(path.TargetX, path.TargetY, 32, 32); d > 3.0 {
		t.Errorf("evening target should be near the village center, got dist %f", d)
	}
	homeX, homeY := path.TargetX, path.TargetY

	// Night (phase 3, ticks 1800-2399) at home: rest, no path issued.
	pos.X, pos.Y = homeX, homeY
	path.HasPath = false
	path.Nodes = nil
	cal.Ticks = 2100
	runTicks(sys, cal, world, 30)
	if path.HasPath {
		t.Errorf("night at home: farmer should rest with no path, got target (%f,%f)",
			path.TargetX, path.TargetY)
	}
	if rc.Phase != systems.RoutinePhaseNight {
		t.Errorf("night: routine phase should be %d, got %d", systems.RoutinePhaseNight, rc.Phase)
	}

	// Night far from home: still walks back to the center.
	pos.X, pos.Y = 50, 50
	path.HasPath = false
	path.Nodes = nil
	cal.Ticks = 2160
	runTicks(sys, cal, world, 30)
	if !path.HasPath {
		t.Fatalf("night away: farmer should head home")
	}
	if d := dist(path.TargetX, path.TargetY, 32, 32); d > 3.0 {
		t.Errorf("night-return target should be near the village center, got dist %f", d)
	}
}

func TestDailyRoutine_GuardPatrolRotates(t *testing.T) {
	world, _, cal, sys, ents := buildAnchorWorld()
	guard := ents[1006]
	pathID := ecs.ComponentID[components.Path](world)

	// Assign pass at tick 120, then a first patrol target (rotation step 0).
	// The Path pointer is fetched after the run: the assign pass relocates
	// the entity's components when it adds RoutineComponent.
	runTicks(sys, cal, world, 150)
	path := (*components.Path)(world.Get(guard, pathID))
	if !path.HasPath {
		t.Fatalf("guard should have a patrol target after the first window")
	}
	t1x, t1y := path.TargetX, path.TargetY
	if d := dist(t1x, t1y, 32, 32); d < 7.0 || d > 9.0 {
		t.Errorf("patrol target should sit on the radius-8 ring, got dist %f", d)
	}

	// Later in the same morning the ring target must have rotated.
	path.HasPath = false
	path.Nodes = nil
	cal.Ticks = 429 // rotation step 429/150 = 2 -> angle advanced
	runTicks(sys, cal, world, 30)
	if !path.HasPath {
		t.Fatalf("guard should have a second patrol target")
	}
	t2x, t2y := path.TargetX, path.TargetY
	if d := dist(t2x, t2y, 32, 32); d < 7.0 || d > 9.0 {
		t.Errorf("rotated patrol target should stay on the radius-8 ring, got dist %f", d)
	}
	if d := dist(t1x, t1y, t2x, t2y); d < 1.0 {
		t.Errorf("guard patrol target should visibly rotate around the ring, moved only %f", d)
	}
}

// TestWanderSystem_RoutineExclusion mirrors TestWanderSystem_PossessedBypass:
// a routine-tagged NPC must be invisible to WanderSystem even when hungry,
// while an identical untagged NPC still wanders for food.
func TestWanderSystem_RoutineExclusion(t *testing.T) {
	world := ecs.NewWorld()
	grid := engine.NewMapGrid(10, 10)
	grid.Resources[5*10+5].FoodValue = 10
	grid.FoodCache = append(grid.FoodCache, 5*10+5)

	queue := engine.NewPathRequestQueue(10, 1)
	sys := systems.NewWanderSystem(&world, grid, queue)

	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	routineID := ecs.ComponentID[components.RoutineComponent](&world)

	// Routine-driven employed citizen: excluded from wandering.
	employed := world.NewEntity(posID, idID, needsID, pathID, routineID)
	(*components.Position)(world.Get(employed, posID)).X = 1
	(*components.Identity)(world.Get(employed, idID)).ID = 1
	(*components.Needs)(world.Get(employed, needsID)).Food = 10 // starving

	// Unemployed control: keeps the wander behavior.
	drifter := world.NewEntity(posID, idID, needsID, pathID)
	(*components.Position)(world.Get(drifter, posID)).X = 1
	(*components.Identity)(world.Get(drifter, idID)).ID = 2
	(*components.Needs)(world.Get(drifter, needsID)).Food = 10

	for i := 0; i < 35; i++ {
		sys.Update(&world)
	}

	if (*components.Path)(world.Get(employed, pathID)).HasPath {
		t.Errorf("WanderSystem must not steer routine-tagged NPCs")
	}
	if !(*components.Path)(world.Get(drifter, pathID)).HasPath {
		t.Errorf("unemployed NPCs must keep wandering for food")
	}
}

// TestDailyRoutine_UntagsOnJobLoss verifies the cleanup pass removes the
// routine when the NPC becomes unemployed, handing them back to WanderSystem.
func TestDailyRoutine_UntagsOnJobLoss(t *testing.T) {
	world, _, cal, sys, ents := buildAnchorWorld()
	farmer := ents[1001]
	jobID := ecs.ComponentID[components.JobComponent](world)
	routineID := ecs.ComponentID[components.RoutineComponent](world)

	runTicks(sys, cal, world, 120)
	if !world.Has(farmer, routineID) {
		t.Fatalf("farmer should be routine-tagged after the assign pass")
	}

	(*components.JobComponent)(world.Get(farmer, jobID)).JobID = components.JobNone
	runTicks(sys, cal, world, 120)
	if world.Has(farmer, routineID) {
		t.Errorf("routine tag should be removed when the NPC loses their job")
	}
}
