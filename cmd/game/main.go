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
	"github.com/hajimehoshi/ebiten/v2"
)

// Phase 01.4: Hardware Affinity & Rendering Bridging
// Phase 01.6: Telemetry & Profiling

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
	tickManager.AddSystem(systems.NewInfrastructureWearSystem(grid))

	// Simulation Goroutine
	go func() {
		// Phase 01.4: Hardware Affinity
		// Pin this goroutine to an OS thread to prevent cache invalidations
		runtime.LockOSThread()

		fmt.Println("Simulation Goroutine locked to OS thread.")

		// Run simulation loop indefinitely
		tickManager.Run(-1)
	}()

	// Phase 08.1: Window Management & Camera
	// Ebitengine handles its own main-thread graphics context hijacking.
	// We no longer need the dummy render goroutine.
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Boundless Sovereigns")

	// Create and run the new Ebitengine application on the main thread
	app := render.NewApp(tickManager, grid)
	if err := ebiten.RunGame(app); err != nil {
		log.Fatalf("Ebitengine failed: %v", err)
	}
}
