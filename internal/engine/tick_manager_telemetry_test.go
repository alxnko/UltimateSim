package engine

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mlange-42/arche/ecs"
)

// Phase 01.6: Telemetry
// Write an E2E test verifying telemetry does not disrupt TPS.

type slowSystem struct{}

func (s *slowSystem) Update(world *ecs.World) {
	// Simulate work that takes 1ms
	time.Sleep(1 * time.Millisecond)
}

func TestTickManager_Telemetry(t *testing.T) {
	// Intercept stdout to verify logging output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tm := NewTickManager(60)
	tm.AddSystem(&slowSystem{})

	// Run for 65 ticks so it logs at least once (logs after TPS=60 ticks)
	tm.Run(65)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Ticks Processing Time (ms):") {
		t.Errorf("Telemetry output missing. Output: %s", output)
	}
}
