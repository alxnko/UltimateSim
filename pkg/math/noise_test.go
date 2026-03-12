package math

import (
	"testing"
)

func TestNewPerlin(t *testing.T) {
	seed1 := [32]byte{1, 2, 3, 4}
	seed2 := [32]byte{1, 2, 3, 4}
	seed3 := [32]byte{5, 6, 7, 8}

	p1 := NewPerlin(seed1)
	p2 := NewPerlin(seed2)
	p3 := NewPerlin(seed3)

	// Test determinism: same seed should produce same permutation
	for i := range p1.p {
		if p1.p[i] != p2.p[i] {
			t.Errorf("NewPerlin with same seed produced different permutations at index %d", i)
		}
	}

	// Test different seeds produce different permutations
	same := true
	for i := range p1.p {
		if p1.p[i] != p3.p[i] {
			same = false
			break
		}
	}
	if same {
		t.Errorf("NewPerlin with different seeds produced identical permutations")
	}
}

func TestNoise2D(t *testing.T) {
	seed := [32]byte{42}
	p := NewPerlin(seed)

	t.Run("Consistency", func(t *testing.T) {
		x, y := float32(1.5), float32(2.5)
		v1 := p.Noise2D(x, y)
		v2 := p.Noise2D(x, y)
		if v1 != v2 {
			t.Errorf("Noise2D is not consistent: %f != %f", v1, v2)
		}
	})

	t.Run("DeterminismAcrossInstances", func(t *testing.T) {
		p2 := NewPerlin(seed)
		x, y := float32(10.1), float32(20.2)
		v1 := p.Noise2D(x, y)
		v3 := p2.Noise2D(x, y)

		if v1 != v3 {
			t.Errorf("Noise2D is not deterministic across instances with same seed: %f != %f", v1, v3)
		}
	})

	t.Run("Range", func(t *testing.T) {
		for x := float32(-10); x < 10; x += 0.7 {
			for y := float32(-10); y < 10; y += 0.7 {
				v := p.Noise2D(x, y)
				// The implementation uses 1.414 scaling, which can sometimes slightly exceed 1.0
				// due to floating point precision and the nature of the gradient vectors used.
				if v < -1.2 || v > 1.2 {
					t.Errorf("Noise2D value %f at (%f, %f) out of reasonable range [-1.2, 1.2]", v, x, y)
				}
			}
		}
	})

	t.Run("DifferentSeedsDifferentValues", func(t *testing.T) {
		p2 := NewPerlin([32]byte{24})
		x, y := float32(1.23), float32(4.56)
		v1 := p.Noise2D(x, y)
		v2 := p2.Noise2D(x, y)
		if v1 == v2 {
			t.Errorf("Noise2D with different seeds produced identical value %f at (%f, %f)", v1, x, y)
		}
	})
}
