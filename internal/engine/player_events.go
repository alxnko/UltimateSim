package engine

// Shell Phase: Player lifecycle events surfaced from simulation systems to the UI.
// Single-goroutine access pattern: systems write during Tick(), UI reads/clears
// between ticks on the same goroutine — no locking required.

// Death causes surfaced to the heir-selection UI.
const (
	DeathCauseStarvation uint8 = 1
	DeathCauseBleedOut   uint8 = 2
)

// PlayerEvents carries one-shot events from the simulation to the UI layer.
type PlayerEvents struct {
	PlayerDied   bool
	DeathCause   uint8
	DeathTick    uint64
	DeadFamilyID uint32
	DeadID       uint64
}

// Clear resets all one-shot flags after the UI consumed them.
func (p *PlayerEvents) Clear() {
	p.PlayerDied = false
	p.DeathCause = 0
	p.DeathTick = 0
	p.DeadFamilyID = 0
	p.DeadID = 0
}
