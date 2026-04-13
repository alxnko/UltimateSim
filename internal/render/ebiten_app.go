package render

import (
	"sync"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// LoadingStatus tracks the asynchronous generation of the simulation.
type LoadingStatus struct {
	Progress  float32
	Message   string
	Done      bool
	TM        *engine.TickManager
	Grid      *engine.MapGrid
	HookGraph *engine.SparseHookGraph
	Mutex     sync.Mutex
}
