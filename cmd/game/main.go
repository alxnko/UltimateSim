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

func main() {
	// Phase 01.6: Telemetry & Profiling
	// Boot net/http/pprof instance on localhost:6060
	go func() {
		log.Println("Starting pprof server on localhost:6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()

	// Phase 02: Map Generation
	// Instantiate MapGrid and generate terrain deterministically
	grid := engine.NewMapGrid(100, 100)
	seed := [32]byte{1, 2, 3, 4, 5} // Deterministic seed
	engine.InitializeRNG(seed)
	engine.GenerateMap(grid, seed)

	// Phase 17.2: Build Secondary Nav Mesh for Oceanic Pathfinding
	engine.BuildOceanicNavMesh(grid)

	// Phase 04.2: Async Path Queue Pool
	pathQueue := engine.NewPathRequestQueue(1000, runtime.NumCPU())
	pathQueue.StartWorkers()

	// Phase 06.3: Sparse Hook Graph
	hookGraph := engine.NewSparseHookGraph()

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
	tickManager.AddSystem(systems.NewRustSystem(), engine.PhaseResolution)
	// --- PHASE: AI ---
	tickManager.AddSystem(systems.NewDesperationSystem(world), engine.PhaseAI)
	tickManager.AddSystem(systems.NewWanderSystem(world, grid, pathQueue), engine.PhaseAI)
	tickManager.AddSystem(systems.NewNavalRoutingSystem(world, grid, pathQueue, calendar), engine.PhaseAI)

	// --- PHASE: MOVEMENT ---
	tickManager.AddSystem(systems.NewMovementSystem(world, grid, calendar), engine.PhaseMovement)
	tickManager.AddSystem(systems.NewInfrastructureWearSystem(grid), engine.PhaseMovement)

	// --- PHASE: RESOLUTION ---
	tickManager.AddSystem(systems.NewMetabolismSystem(world, calendar), engine.PhaseResolution)
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
	tickManager.AddSystem(systems.NewDebtDefaultSystem(), engine.PhaseResolution)

	// Register Gossip
	tickManager.AddSystem(systems.NewGossipDistributionSystem(world, hookGraph), engine.PhaseResolution)

	// Register Language Drift
	tickManager.AddSystem(systems.NewLanguageDriftSystem(world), engine.PhaseResolution)

	// Phase 18: Justice Engine
	tickManager.AddSystem(systems.NewJusticeSystem(world), engine.PhaseResolution)

	// --- PHASE: CLEANUP ---
	tickManager.AddSystem(systems.NewDeathSystem(world), engine.PhaseCleanup)

	// Phase 03.2: Genesis Spawner (Runs once at tick 0)
	tickManager.AddSystem(systems.NewNPCSpawnerSystem(world, grid), engine.PhaseCleanup)

	/*
		// Simulation Goroutine
		go func() {
			// Phase 01.4: Hardware Affinity
			// Pin this goroutine to an OS thread to prevent cache invalidations
			runtime.LockOSThread()

			fmt.Println("Simulation Goroutine locked to OS thread.")

			// Run simulation loop indefinitely
			tickManager.Run(-1)
		}()
	*/
	// NOTE: Simulation is now driven manually by the render loop to prevent race conditions in the ECS world.

	// Phase 11.1: Switch Pattern Loop -> Unified Raylib loop
	// We handle everything in raylib to prevent OpenGL CGO collision with Ebiten.
	// Phase 01.4: Hardware Affinity
	// Pin the Window Context Goroutine to prevent OS-level cache invalidations on multicore CPUs
	runtime.LockOSThread()
	render.RunRaylibApp(tickManager, grid)
}
