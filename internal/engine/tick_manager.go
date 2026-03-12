package engine

import (
	"fmt"
	"time"

	"github.com/mlange-42/arche/ecs"
)

// Phase 01.3: ECS Core (arche-go) Setup
// Implement TickManager and System interface to manage arche-go World with 60 TPS cap and alpha calculation for rendering.

// System interface that all ECS systems must implement.
type System interface {
	Update(world *ecs.World)
}

// Phase 01.3: SystemRunner phase definitions
type SystemPhase int

const (
	PhaseInput SystemPhase = iota
	PhaseAI
	PhaseMovement
	PhaseResolution
	PhaseCleanup
	numPhases // Track the number of phases for array sizing
)

// TickManager orchestrates the ECS world and systems.
type TickManager struct {
	World   *ecs.World
	Systems [][]System // Array of system slices, indexed by SystemPhase
	TPS     int
	Alpha   float64

	IsPaused bool // Phase 12: Simulation Controls

	lastTick time.Time
	tickTime time.Duration
}

// NewTickManager creates a new TickManager initialized with a new ECS world.
func NewTickManager(tps int) *TickManager {
	world := ecs.NewWorld()
	return &TickManager{
		World:    &world,
		Systems:  make([][]System, numPhases),
		TPS:      tps,
		tickTime: time.Second / time.Duration(tps),
		lastTick: time.Now(),
	}
}

// AddSystem registers a new system to the TickManager under a specific phase.
func (tm *TickManager) AddSystem(sys System, phase SystemPhase) {
	if phase >= 0 && phase < numPhases {
		tm.Systems[phase] = append(tm.Systems[phase], sys)
	}
}

// Tick executes a single simulation tick by iterating through phases sequentially.
func (tm *TickManager) Tick() {
	if tm.IsPaused {
		return
	}
	for phase := PhaseInput; phase < numPhases; phase++ {
		for _, sys := range tm.Systems[phase] {
			sys.Update(tm.World)
		}
	}
}

// Run executes a blocking loop maintaining the targeted TPS.
// A maxTicks parameter is included to allow bounding execution for tests.
// If maxTicks is -1, it loops indefinitely.
func (tm *TickManager) Run(maxTicks int) {
	ticks := 0
	tm.lastTick = time.Now()

	// Phase 01.6: Telemetry
	var accumulatedTickTime time.Duration
	var ticksLogged int

	for {
		if maxTicks != -1 && ticks >= maxTicks {
			break
		}

		now := time.Now()
		elapsed := now.Sub(tm.lastTick)

		if elapsed >= tm.tickTime {
			// Phase 01.6: Telemetry
			tickStart := time.Now()
			tm.Tick()
			tickElapsed := time.Since(tickStart)

			accumulatedTickTime += tickElapsed
			ticksLogged++
			if ticksLogged >= tm.TPS {
				avgMs := float64(accumulatedTickTime.Microseconds()) / 1000.0 / float64(ticksLogged)
				// Print average ticks processing time to fulfill telemetry requirement
				fmt.Printf("Ticks Processing Time (ms): %.4f\n", avgMs)
				accumulatedTickTime = 0
				ticksLogged = 0
			}

			tm.lastTick = tm.lastTick.Add(tm.tickTime)
			ticks++
		} else {
			// Compute Alpha
			tm.Alpha = float64(elapsed) / float64(tm.tickTime)

			// Sleep to cap max loops and prevent fast-forward
			sleepTime := tm.tickTime - elapsed
			if sleepTime > time.Millisecond {
				time.Sleep(sleepTime - time.Millisecond) // sleep slightly less to avoid oversleeping
			}
		}
	}
}

// TogglePause flips the pause state.
func (tm *TickManager) TogglePause() {
	tm.IsPaused = !tm.IsPaused
}
