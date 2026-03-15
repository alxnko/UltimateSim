package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.4: Birth & Genetics Math
// BirthSystem processes settlements with food surplus, converting Storage.Food into Population.
// Calculates genetic inheritance and trait drift for newly spawned citizens.

type BirthSystem struct {
	filter ecs.Filter
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *BirthSystem) IsExpensive() bool {
	return true
}

func NewBirthSystem(world *ecs.World) *BirthSystem {
	storageID := ecs.ComponentID[components.StorageComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	idID := ecs.ComponentID[components.Identity](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	// Explicitly skip ruins to save CPU cycles
	mask := ecs.All(storageID, popID, genID, idID).Without(ruinID)
	return &BirthSystem{
		filter: &mask,
	}
}

func (s *BirthSystem) Update(world *ecs.World) {
	storageID := ecs.ComponentID[components.StorageComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	idID := ecs.ComponentID[components.Identity](world)

	query := world.Query(s.filter)

	for query.Next() {
		storage := (*components.StorageComponent)(query.Get(storageID))
		pop := (*components.PopulationComponent)(query.Get(popID))

		// If surplus food exists, trigger birth event
		if storage.Food >= 50 {
			storage.Food -= 50
			pop.Count++

			var p1Gen, p2Gen components.GenomeComponent
			var p1Traits, p2Traits uint32

			if len(pop.Citizens) < 2 {
				// Use the foundational village genetics if there aren't enough citizens yet
				baseGen := (*components.GenomeComponent)(query.Get(genID))
				baseID := (*components.Identity)(query.Get(idID))

				p1Gen = *baseGen
				p2Gen = *baseGen
				p1Traits = baseID.BaseTraits
				p2Traits = baseID.BaseTraits
			} else {
				// Pick two random parents from the citizen pool deterministically
				idx1 := engine.GetRandomInt() % len(pop.Citizens)
				idx2 := engine.GetRandomInt() % len(pop.Citizens)

				p1 := pop.Citizens[idx1]
				p2 := pop.Citizens[idx2]

				p1Gen = p1.Genetics
				p2Gen = p2.Genetics
				p1Traits = p1.BaseTraits
				p2Traits = p2.BaseTraits
			}

			domMask := uint32(engine.GetRandomInt())
			recMask := uint32(engine.GetRandomInt())

			// Calculate child genetics (Average of parents +/- 5 points mutation)
			childGenetics := components.GenomeComponent{
				Strength:  clampGenetics(int(p1Gen.Strength)+int(p2Gen.Strength), engine.GetRandomInt()),
				Beauty:    clampGenetics(int(p1Gen.Beauty)+int(p2Gen.Beauty), engine.GetRandomInt()),
				Health:    clampGenetics(int(p1Gen.Health)+int(p2Gen.Health), engine.GetRandomInt()),
				Intellect: clampGenetics(int(p1Gen.Intellect)+int(p2Gen.Intellect), engine.GetRandomInt()),
				Dominant:  (p1Gen.Dominant & domMask) | (p2Gen.Dominant & ^domMask),
				Recessive: (p1Gen.Recessive & recMask) | (p2Gen.Recessive & ^recMask),
			}

			// Phase 19.1: Inbreeding Penalties
			// Calculate similarity between parent Dominant arrays.
			// The XOR result will have 0s where bits are identical. We count differences (1s).
			diffCount := 0
			xorRes := p1Gen.Dominant ^ p2Gen.Dominant
			for xorRes > 0 {
				diffCount += int(xorRes & 1)
				xorRes >>= 1
			}

			// If parents share extremely similar dominant traits (< 5 bits difference), penalize health.
			if diffCount < 5 {
				childGenetics.Health /= 2
			}

			// Inherit BaseTraits via 50% chance bitmask evaluation
			mask := uint32(engine.GetRandomInt())
			childTraits := (p1Traits & mask) | (p2Traits & ^mask)

			newCitizen := components.CitizenData{
				Genetics:   childGenetics,
				BaseTraits: childTraits,
				Age:        0,
			}

			pop.Citizens = append(pop.Citizens, newCitizen)
		}
	}
}

// clampGenetics takes the sum of two parent traits, averages them, applies mutation, and clamps to 0-255.
func clampGenetics(sum int, mutationRoll int) uint8 {
	avg := sum / 2
	mutation := (mutationRoll % 11) - 5 // Range -5 to +5
	val := avg + mutation

	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}
