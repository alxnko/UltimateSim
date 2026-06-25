package render

import (
	"sync"

	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
)

// LoadingStatus tracks the asynchronous generation of the simulation and holds
// the shared handles the UI layer needs once the world is ready.
type LoadingStatus struct {
	Progress  float32
	Message   string
	Done      bool
	TM        *engine.TickManager
	Grid      *engine.MapGrid
	HookGraph *engine.SparseHookGraph
	Seed      byte // Shell Phase: remembered for save metadata

	// Shell Phase: cross-tick player bridges, created once by the engine build.
	Bridge *systems.InputBridge
	Events *engine.PlayerEvents

	Mutex sync.Mutex
}
