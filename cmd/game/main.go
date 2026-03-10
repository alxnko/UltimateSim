package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"time"

	"github.com/ALXNKO/UltimateSim/internal/engine"
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

	var wg sync.WaitGroup
	wg.Add(2)

	// Simulation Goroutine
	go func() {
		defer wg.Done()
		// Phase 01.4: Hardware Affinity
		// Pin this goroutine to an OS thread to prevent cache invalidations
		runtime.LockOSThread()

		fmt.Println("Simulation Goroutine locked to OS thread.")

		// Initialize the TickManager with 60 TPS bounds
		tickManager := engine.NewTickManager(60)

		// Run simulation loop indefinitely
		tickManager.Run(-1)
	}()

	// Render/Window Context Goroutine
	go func() {
		defer wg.Done()
		// Phase 01.4: Hardware Affinity
		// Pin this goroutine to an OS thread to prevent cache invalidations
		runtime.LockOSThread()

		fmt.Println("Render/Window Context Goroutine locked to OS thread.")

		// Placeholder for actual render loop
		// We'll just read alpha values periodically as a simulation of decoupled rendering
		for {
			time.Sleep(16 * time.Millisecond) // roughly 60 FPS
			// In reality, this would read tickManager.Alpha via a thread-safe mechanism
			// For now it's just a dummy loop
		}
	}()

	wg.Wait()
}
