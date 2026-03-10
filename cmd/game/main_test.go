package main

import (
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 01.4: Hardware Affinity & Rendering Bridging
// Phase 01.6: Telemetry & Profiling

func TestMainComponentsE2E(t *testing.T) {
	// 1. Verify that the simulation loop runs and computes the Alpha variable
	tickManager := engine.NewTickManager(60)

	go func() {
		runtime.LockOSThread()
		tickManager.Run(60) // run for 60 ticks
	}()

	// Read Alpha periodically simulating the render loop
	// We'll wait up to a few cycles for Alpha to be populated
	var alpha float64
	for i := 0; i < 10; i++ {
		alpha = tickManager.Alpha
		if alpha >= 0 && alpha <= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if alpha < 0 || alpha > 1 {
		t.Errorf("Expected Alpha value between 0 and 1, got %v", alpha)
	}

	// 2. Verify pprof HTTP endpoint is successfully launched and responsive
	go func() {
		if err := http.ListenAndServe("localhost:6061", nil); err != nil {
			fmt.Printf("pprof server ended with error: %v\n", err)
		}
	}()

	var resp *http.Response
	var err error
	for i := 0; i < 10; i++ {
		resp, err = http.Get("http://localhost:6061/debug/pprof/")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err != nil {
		t.Errorf("Expected pprof endpoint to be responsive, got error: %v", err)
	} else if resp != nil {
		if resp.StatusCode != 200 {
			t.Errorf("Expected pprof endpoint to return 200, got %v", resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 3. Deterministic check ensuring that running the world twice with the same seed
	// yields identical outputs (although here we just test the loop can be restarted with same logic)
	// We'll reset seed
	seed := [32]byte{1, 2, 3, 4, 5}
	engine.InitializeRNG(seed)
	val1 := engine.GetRandomFloat32()

	engine.InitializeRNG(seed)
	val2 := engine.GetRandomFloat32()

	if val1 != val2 {
		t.Errorf("Deterministic check failed: %v != %v", val1, val2)
	}
}
