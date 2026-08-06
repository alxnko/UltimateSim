package zombiepoc

// PoC for the GC zombie ("found pointer to free object") corruption:
// arche v0.15.3 archetype.copy moves component memory as raw bytes
// ([]byte copy, no pointer write barriers). If a component holds a heap
// pointer (slice/string/map), any archetype move (Add/Remove component,
// swap-remove on entity deletion) copies that pointer without the GC's
// hybrid write barrier. During a concurrent mark phase the pointee can be
// missed, swept free, and later re-discovered via the raw-copied slot:
// "runtime: marked free object in span ... found pointer to free object".
//
// Run with: GODEBUG=clobberfree=1 GOGC=5 go test ./internal/zombiepoc/ -count=N
//
// The Holder component mimics components.Path: a small []Pos whose backing
// arrays are 16 bytes (2 x 8-byte Pos), matching the elemsize=16 zombie
// spans in every observed crash.

import (
	"testing"

	"github.com/mlange-42/arche/ecs"
)

type Pos struct{ X, Y float32 }

type Holder struct {
	Nodes []Pos
}

type MarkerA struct{ V uint32 }
type MarkerB struct{ V uint32 }

func TestArcheRawCopyZombie(t *testing.T) {
	world := ecs.NewWorld()
	holderID := ecs.ComponentID[Holder](&world)
	markAID := ecs.ComponentID[MarkerA](&world)
	markBID := ecs.ComponentID[MarkerB](&world)

	const n = 2000
	ents := make([]ecs.Entity, 0, n)
	for i := 0; i < n; i++ {
		e := world.NewEntity(holderID)
		h := (*Holder)(world.Get(e, holderID))
		h.Nodes = make([]Pos, 2) // 16-byte backing array
		ents = append(ents, e)
	}

	// Churn loop: replace slices (make old backing arrays garbage), toggle
	// components (raw-copy archetype moves), remove + respawn entities
	// (swap-remove raw copies + buffer extends), and allocate garbage to
	// keep the GC marking concurrently.
	garbage := make([][]byte, 64)
	for iter := 0; iter < 60000; iter++ {
		e := ents[iter%len(ents)]
		if !world.Alive(e) {
			continue
		}

		switch iter % 3 {
		case 0:
			if world.Has(e, markAID) {
				world.Remove(e, markAID)
			} else {
				world.Add(e, markAID)
			}
		case 1:
			if world.Has(e, markBID) {
				world.Remove(e, markBID)
			} else {
				world.Add(e, markBID)
			}
		case 2:
			// Fresh path assignment, like a PathResult overwriting Path.Nodes.
			h := (*Holder)(world.Get(e, holderID))
			h.Nodes = make([]Pos, 2)
		}

		if iter%7 == 0 {
			// Kill and respawn: swap-remove copies the last entity's raw
			// bytes into the hole; respawn extends buffers.
			victim := ents[(iter*13)%len(ents)]
			if world.Alive(victim) {
				world.RemoveEntity(victim)
			}
			ne := world.NewEntity(holderID)
			h := (*Holder)(world.Get(ne, holderID))
			h.Nodes = make([]Pos, 2)
			ents[(iter*13)%len(ents)] = ne
		}

		// GC pressure so mark phases interleave with the raw copies.
		garbage[iter%len(garbage)] = make([]byte, 128)
	}
	_ = garbage
}
