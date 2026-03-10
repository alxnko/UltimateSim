package engine

import (
	"testing"
)

func TestRNGDeterminism(t *testing.T) {
	// Seed A
	seedA := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	InitializeRNG(seedA)

	seq1 := make([]int, 100)
	for i := 0; i < 100; i++ {
		seq1[i] = GetRandomInt()
	}

	// Re-initialize RNG with the same Seed A
	InitializeRNG(seedA)

	seq2 := make([]int, 100)
	for i := 0; i < 100; i++ {
		seq2[i] = GetRandomInt()
	}

	// Compare the two sequences
	for i := 0; i < 100; i++ {
		if seq1[i] != seq2[i] {
			t.Errorf("Determinism failure at index %d: expected %d, got %d", i, seq1[i], seq2[i])
		}
	}
}
