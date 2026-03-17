package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
)

// Phase 01.4: Hardware Affinity & Rendering Bridging
// Phase 01.6: Telemetry & Profiling
// Phase 11.1: Raylib rendering architecture

func BuildSimulation(gridWidth, gridHeight int, seedVal byte, status *render.LoadingStatus) {
	update := func(progress float32, msg string) {
		status.Mutex.Lock()
		status.Progress = progress
		status.Message = msg
		status.Mutex.Unlock()
	}

	update(0.1, "Initializing Map Grid...")
	// Phase 02: Map Generation
	// Phase 02.1: Map Scaling support configurable grid sizes and scales
	grid := engine.NewMapGrid(gridWidth, gridHeight)

	update(0.2, "Initializing RNG & Seed...")
	seed := [32]byte{seedVal, seedVal + 1, seedVal + 2, seedVal + 3, seedVal + 4} // Deterministic seed based on input
	engine.InitializeRNG(seed)

	update(0.3, "Generating Tectonic & Biome Data...")
	engine.GenerateMap(grid, seed)

	update(0.5, "Building Oceanic Nav Mesh...")
	// Phase 17.2: Build Secondary Nav Mesh for Oceanic Pathfinding
	engine.BuildOceanicNavMesh(grid)

	update(0.6, "Starting Pathfinding Workers...")
	// Phase 04.2: Async Path Queue Pool
	pathQueue := engine.NewPathRequestQueue(1000, runtime.NumCPU())
	pathQueue.StartWorkers()

	update(0.7, "Initializing Social Hook Graph...")
	// Phase 06.3: Sparse Hook Graph
	hookGraph := engine.NewSparseHookGraph()

	update(0.8, "Assembling ECS Engine & Systems...")
	// Phase 01.3: Initialize the TickManager with 60 TPS
	tickManager := engine.NewTickManager(60)
	world := tickManager.World

	// Phase 13.4: The Seasonal Pulse
	calendar := engine.NewCalendar()
	tickManager.AddSystem(systems.NewCalendarSystem(calendar), engine.PhaseInput)

	// Phase 19.2: Ecological Drift
	tickManager.AddSystem(systems.NewGlobalWeatherSystem(world, grid), engine.PhaseInput)

	// Phase 09.2: Dynamic Attrition
	tickManager.AddSystem(systems.NewSpoilageSystem(), engine.PhaseResolution)
	// Phase 37.1: The Quarantine Engine
	tickManager.AddSystem(systems.NewQuarantineSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewNaturalDisasterSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewExposureSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRustSystem(), engine.PhaseResolution)
	// --- PHASE: AI ---
	tickManager.AddSystem(systems.NewDesperationSystem(world), engine.PhaseAI)
	tickManager.AddSystem(systems.NewBanditrySystem(world), engine.PhaseAI)
	tickManager.AddSystem(systems.NewWanderSystem(world, grid, pathQueue), engine.PhaseAI)
	tickManager.AddSystem(systems.NewNavalRoutingSystem(world, grid, pathQueue, calendar), engine.PhaseAI)

	// --- PHASE: MOVEMENT ---
	tickManager.AddSystem(systems.NewMovementSystem(world, grid, calendar), engine.PhaseMovement)
	tickManager.AddSystem(systems.NewInfrastructureWearSystem(grid), engine.PhaseMovement)

	// --- PHASE: RESOLUTION ---
	tickManager.AddSystem(systems.NewMetabolismSystem(world, calendar, tickManager), engine.PhaseResolution)
	// Phase 17.3: Maritime Attrition & Piracy
	tickManager.AddSystem(systems.NewStormSystem(grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewNavalPiracySystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewBirthSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewDiseaseVectorSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCaravanSpawnerSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCareerChangeSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCityBinderSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewSettlementRuleSystem(grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPriceDiscoverySystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRuinTransformationSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewAdministrativeDecaySystem(), engine.PhaseResolution)
	// Phase 16.4: Administrative Reach & Friction
	tickManager.AddSystem(systems.NewAdministrativeFractureSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewLendingSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewDebtDefaultSystem(), engine.PhaseResolution)

	// Phase 28.1: The Vassal Rebellion Engine
	tickManager.AddSystem(systems.NewVassalRebellionSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 33.1: Cultural Friction & Ideological Secession Engine
	tickManager.AddSystem(systems.NewCulturalFrictionSystem(), engine.PhaseResolution)

	// Phase 29.1: Geopolitical Resource Wars
	tickManager.AddSystem(systems.NewResourceWarSystem(world, hookGraph), engine.PhaseResolution)

	// Register Gossip
	tickManager.AddSystem(systems.NewInformationTradeSystem(world, hookGraph), engine.PhaseResolution)

	tickManager.AddSystem(systems.NewGossipDistributionSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 39.1 & 39.2: The Epistemological Engine
	tickManager.AddSystem(systems.NewScholarSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewLedgerDiscoverySystem(world), engine.PhaseResolution)

	// Register Language Drift
	tickManager.AddSystem(systems.NewLanguageDriftSystem(world), engine.PhaseResolution)

	// Phase 30.1: Ideological Economy
	tickManager.AddSystem(systems.NewTitheSystem(world), engine.PhaseResolution)

	// Phase 18: Justice Engine
	tickManager.AddSystem(systems.NewJusticeSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 27.1: The Military Revolt Engine
	tickManager.AddSystem(systems.NewMilitaryRevoltSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 24.1: The Labor Union Engine
	tickManager.AddSystem(systems.NewLaborUnionSystem(world, hookGraph), engine.PhaseResolution)

	// --- PHASE: CLEANUP ---
	tickManager.AddSystem(systems.NewDeathSystem(world, hookGraph), engine.PhaseCleanup)
	tickManager.AddSystem(systems.NewAgingSystem(world, tickManager), engine.PhaseResolution)

	// Phase 03.2: Genesis Spawner (Runs once at tick 0)
	tickManager.AddSystem(systems.NewNPCSpawnerSystem(world, grid), engine.PhaseCleanup)

	update(1.0, "Engine Assembly Complete.")

	status.Mutex.Lock()
	status.TM = tickManager
	status.Grid = grid
	status.Done = true
	status.Mutex.Unlock()
}

func main() {
	// Phase 01.6: Telemetry & Profiling
	// Boot net/http/pprof instance on localhost:6060
	go func() {
		log.Println("Starting pprof server on localhost:6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()

	// NOTE: Simulation is now driven manually by the render loop to prevent race conditions in the ECS world.

	// Phase 11.1: Switch Pattern Loop -> Unified Raylib loop
	// We handle everything in raylib to prevent OpenGL CGO collision with Ebiten.
	// Phase 01.4: Hardware Affinity
	// Pin the Window Context Goroutine to prevent OS-level cache invalidations on multicore CPUs
	runtime.LockOSThread()

	// Delegate state management entirely to RaylibApp, passing the factory function
	render.RunRaylibApp(BuildSimulation)
}
