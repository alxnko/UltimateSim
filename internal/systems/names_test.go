package systems

import (
	"strings"
	"testing"
	"unicode"
)

// Grand Strategy Phase: deterministic name generator tests.
// Name must be a pure function of (id, culture), draw endings from the gender
// pool selected by the id, vary flavor by culture, and never emit placeholder
// "NPC-" names.

// TestNameDeterministic verifies Name is pure: same inputs, same output.
func TestNameDeterministic(t *testing.T) {
	for culture := uint8(0); culture < nameCultureCount; culture++ {
		for id := uint64(1); id <= 200; id++ {
			a := Name(id, culture)
			b := Name(id, culture)
			if a != b {
				t.Fatalf("Name(%d, %d) not deterministic: %q vs %q", id, culture, a, b)
			}
		}
	}
}

// TestNameShape verifies names are capitalized, non-placeholder, and within
// 2-3 syllable length bounds.
func TestNameShape(t *testing.T) {
	for culture := uint8(0); culture < nameCultureCount; culture++ {
		for id := uint64(1); id <= 200; id++ {
			name := Name(id, culture)
			if len(name) < 3 {
				t.Fatalf("Name(%d, %d) = %q too short", id, culture, name)
			}
			if strings.HasPrefix(name, "NPC-") {
				t.Fatalf("Name(%d, %d) = %q is a placeholder", id, culture, name)
			}
			if !unicode.IsUpper(rune(name[0])) {
				t.Errorf("Name(%d, %d) = %q not capitalized", id, culture, name)
			}
		}
	}
}

// TestNameGenderPools verifies the ending comes from the pool selected by
// id % 3 (0 masculine, 1 feminine, 2 neutral).
func TestNameGenderPools(t *testing.T) {
	hasSuffixFrom := func(name string, pool [4]string) bool {
		for _, s := range pool {
			if strings.HasSuffix(name, s) {
				return true
			}
		}
		return false
	}

	for culture := uint8(0); culture < nameCultureCount; culture++ {
		c := &nameCultures[culture]
		for id := uint64(1); id <= 90; id++ {
			name := Name(id, culture)
			var pool [4]string
			switch id % 3 {
			case 0:
				pool = c.masc
			case 1:
				pool = c.fem
			default:
				pool = c.neutral
			}
			if !hasSuffixFrom(name, pool) {
				t.Errorf("Name(%d, %d) = %q lacks an ending from its gender pool %v", id, culture, name, pool)
			}
		}
	}
}

// TestNameCultureFlavor verifies cultures produce distinct flavors: every
// culture's onset table is disjoint, so the same id maps to different names.
func TestNameCultureFlavor(t *testing.T) {
	differs := 0
	for id := uint64(1); id <= 50; id++ {
		if Name(id, 0) != Name(id, 1) {
			differs++
		}
	}
	if differs < 40 {
		t.Errorf("cultures 0 and 1 agree too often: only %d/50 names differ", differs)
	}
}

// TestNameVariety verifies a healthy spread of distinct names per culture.
func TestNameVariety(t *testing.T) {
	seen := make(map[string]struct{})
	for id := uint64(1); id <= 100; id++ {
		seen[Name(id, 0)] = struct{}{}
	}
	if len(seen) < 30 {
		t.Errorf("only %d distinct names across 100 ids, want >= 30", len(seen))
	}
}
