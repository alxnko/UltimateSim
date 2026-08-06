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
	// 1. Verify that the simulation loop runs and computes the Alpha variable.
	// The Run goroutine MUST be joined before the test returns: it kept
	// running ~1s into the next test before, racing the Alpha field and
	// leaking a live ticker across -count=2 iterations.
	tickManager := engine.NewTickManager(60)

	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		tickManager.Run(30) // run for 30 ticks (~0.5s)
	}()
	<-done

	alpha := tickManager.Alpha
	if alpha < 0 || alpha > 1 {
		t.Errorf("Expected Alpha value between 0 and 1, got %v", alpha)
	}

	// 2. Verify the pprof HTTP endpoint launches and responds; shut it down
	// before returning so nothing leaks into later tests or -count=2 reruns.
	srv := &http.Server{Addr: "localhost:6061"}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("pprof server ended with error: %v\n", err)
		}
	}()
	defer srv.Close()

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
