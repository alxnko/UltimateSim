package math

import (
	"math"
	"math/rand/v2"
)

// Phase 02.2: Procedural Generation Pipeline - Noise Function

// Perlin represents a 2D Perlin noise generator.
// It is initialized with a specific seed to ensure deterministic outputs.
type Perlin struct {
	p [512]int
}

// NewPerlin creates a new Perlin noise generator with the given seed.
// We use math/rand/v2 with ChaCha8 for cryptographically secure deterministic seeding.
func NewPerlin(seed [32]byte) *Perlin {
	src := rand.NewChaCha8(seed)
	rng := rand.New(src)

	p := &Perlin{}
	var permutation [256]int
	for i := 0; i < 256; i++ {
		permutation[i] = i
	}

	// Shuffle the permutation array deterministically
	rng.Shuffle(256, func(i, j int) {
		permutation[i], permutation[j] = permutation[j], permutation[i]
	})

	for i := 0; i < 256; i++ {
		p.p[i] = permutation[i]
		p.p[256+i] = permutation[i]
	}

	return p
}

// fade applies the smoothstep function: 6t^5 - 15t^4 + 10t^3
func fade(t float32) float32 {
	return t * t * t * (t*(t*6-15) + 10)
}

// lerp performs linear interpolation between a and b.
func lerp(t, a, b float32) float32 {
	return a + t*(b-a)
}

// grad computes the dot product of the distance and gradient vectors.
func grad(hash int, x, y float32) float32 {
	h := hash & 3
	var u, v float32
	if h < 2 {
		u = x
	} else {
		u = -x
	}
	if h&1 == 0 {
		v = y
	} else {
		v = -y
	}
	return u + v
}

// Noise2D generates 2D Perlin noise at the given coordinates.
// Returns a value between -1.0 and 1.0.
// We use float32 to adhere strictly to DOD guidelines for CPU cache efficiency.
func (p *Perlin) Noise2D(x, y float32) float32 {
	X := int(math.Floor(float64(x))) & 255
	Y := int(math.Floor(float64(y))) & 255

	x -= float32(math.Floor(float64(x)))
	y -= float32(math.Floor(float64(y)))

	u := fade(x)
	v := fade(y)

	A := p.p[X] + Y
	B := p.p[X+1] + Y

	val := lerp(v, lerp(u, grad(p.p[A], x, y), grad(p.p[B], x-1, y)),
		lerp(u, grad(p.p[A+1], x, y-1), grad(p.p[B+1], x-1, y-1)))

	// The theoretical max bounds of this implementation is sqrt(2/4) = 0.707
	// Scaling by 1.414 ensures the output reaches [-1.0, 1.0].
	return val * 1.414
}
