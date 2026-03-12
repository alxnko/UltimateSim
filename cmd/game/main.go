package main

import (
	"fmt"
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
	engine.GenerateMap(grid, seed)

	// Initialize the TickManager with 60 TPS bounds outside goroutine so it can be passed
	tickManager := engine.NewTickManager(60)

	// Phase 09.2: Dynamic Attrition
	tickManager.AddSystem(systems.NewSpoilageSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRustSystem(), engine.PhaseResolution)

	// Phase 09.3: Infrastructure Wear System
	tickManager.AddSystem(systems.NewInfrastructureWearSystem(grid), engine.PhaseMovement)

	// Simulation Goroutine
	go func() {
		// Phase 01.4: Hardware Affinity
		// Pin this goroutine to an OS thread to prevent cache invalidations
		runtime.LockOSThread()

		fmt.Println("Simulation Goroutine locked to OS thread.")

		// Run simulation loop indefinitely
		tickManager.Run(-1)
	}()

	// Phase 11.1: Switch Pattern Loop -> Unified Raylib loop
	// We handle everything in raylib to prevent OpenGL CGO collision with Ebiten.
	// Phase 01.4: Hardware Affinity
	// Pin the Window Context Goroutine to prevent OS-level cache invalidations on multicore CPUs
	runtime.LockOSThread()
	render.RunRaylibApp(tickManager, grid)
}
