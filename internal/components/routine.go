package components

// Grand Strategy Phase (P5/L1): Visible NPC daily life.
// Spec: docs/superpowers/specs/2026-08-06-grand-strategy-playable.md
//
// RoutineComponent tags an employed, city-bound NPC as steered by
// DailyRoutineSystem (job anchors + day-phase movement) instead of
// WanderSystem. AnchorX/AnchorY is the deterministic job anchor computed once
// per NPC from the city's village position + job + Identity.ID jitter; for
// guards the anchor is the village center their patrol ring orbits. Phase is
// the last day phase applied (see systems.RoutinePhase*). JobSeen records the
// JobID the anchor was computed for so career changes trigger an anchor
// refresh. The component is derived state: DailyRoutineSystem re-tags and
// recomputes it deterministically, so a world without it self-heals.
type RoutineComponent struct {
	AnchorX float32
	AnchorY float32
	Phase   uint8
	JobSeen uint8
	_       uint16 // Padding to exactly 12 bytes
}
