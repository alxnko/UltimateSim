package systems

// Shell Phase: InputBridge decouples the UI layer (camera, cursor math) from
// ECS systems. The UI fills intents each frame before TickManager.Tick();
// PlayerInputSystem consumes and clears them during PhaseInput. Systems push
// feedback events the UI drains after the tick. Single-goroutine access —
// no locking required.

// Bridge event kinds (simulation → UI feedback).
const (
	EventDamage       uint8 = 1 // Value = damage dealt; X,Y = world pos
	EventAttackSwing  uint8 = 2 // X,Y = world pos of swing target
	EventExhausted    uint8 = 3 // Player too tired to attack
	EventOrderRefused uint8 = 4 // Value = NPC Identity.ID low bits
	EventDeath        uint8 = 5 // Entity died at X,Y
)

// BridgeEvent is one feedback item for the UI effects layer.
type BridgeEvent struct {
	X     float32
	Y     float32
	Value int32
	Kind  uint8
}

// InputBridge carries player intents (UI → simulation) and feedback events
// (simulation → UI) across the tick boundary.
type InputBridge struct {
	// Attack intent: world coordinates resolved by the UI from cursor + camera.
	AttackX     float32
	AttackY     float32
	AttackValid bool

	// Feedback queue drained by the UI each frame.
	Events []BridgeEvent

	// MovementLocked suppresses WASD control while a modal UI owns the keyboard.
	MovementLocked bool
}

// PushEvent appends a feedback event (capacity-friendly).
func (b *InputBridge) PushEvent(kind uint8, x, y float32, value int32) {
	b.Events = append(b.Events, BridgeEvent{X: x, Y: y, Value: value, Kind: kind})
}

// ClearIntents resets one-shot intents after PlayerInputSystem consumed them.
func (b *InputBridge) ClearIntents() {
	b.AttackValid = false
}

// DrainEvents returns queued events and resets the queue, retaining capacity.
func (b *InputBridge) DrainEvents() []BridgeEvent {
	evs := b.Events
	b.Events = b.Events[len(b.Events):]
	return evs
}
