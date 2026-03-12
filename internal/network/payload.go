package network

// Phase 12.2: Delta Extraction Queries
// Define payload structures strictly adhering to DOD 16-byte alignment

// PositionDelta represents a fractional positional update for a single entity.
// Fields: uint64 (8 bytes) + float32 (4 bytes) + float32 (4 bytes) = 16 bytes exactly.
// This perfectly aligns to 16-byte CPU cache bounds without any Go compiler padding.
type PositionDelta struct {
	EntityID uint64
	X        float32
	Y        float32
}

// DeltaPayload represents the batched payload to be sent to clients.
type DeltaPayload struct {
	Tick    int
	Deltas  []PositionDelta
}
