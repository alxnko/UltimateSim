package systems

import (
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 10.3: Biological Entropy (Plagues & Immune Arrays)
// DiseaseVectorSystem handles the random generation of plagues and evaluates lethality.

type DiseaseVectorSystem struct {
	mapGrid        *engine.MapGrid
	spawnChance    float32
	trafficTrigger uint32
	currentTick    uint64

	toRemove   []ecs.Entity
	toImmunize []immuneData
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *DiseaseVectorSystem) IsExpensive() bool {
	return true
}

// IsNonEssential returns true to skip this system during fast-forward.
func (s *DiseaseVectorSystem) IsNonEssential() bool {
	return true
}

type immuneData struct {
	entity    ecs.Entity
	diseaseID uint32
}

func NewDiseaseVectorSystem(world *ecs.World, mapGrid *engine.MapGrid) *DiseaseVectorSystem {
	return &DiseaseVectorSystem{
		mapGrid:        mapGrid,
		spawnChance:    0.01, // 1% chance per high-traffic tile evaluated per tick (adjustable)
		trafficTrigger: 500,  // Minimum foot traffic to be considered a trade hub array
		toRemove:       make([]ecs.Entity, 0, 100),
		toImmunize:     make([]immuneData, 0, 100),
	}
}

func (s *DiseaseVectorSystem) Update(world *ecs.World) {
	s.currentTick++

	// Seed local PRNG to guarantee determinism without polluting global state
	var seed [32]byte
	seed[0] = byte(s.currentTick)
	seed[1] = byte(s.currentTick >> 8)
	prng := rand.New(rand.NewChaCha8(seed))

	// 1. Evaluate high-traffic trade hub arrays for random disease generation
	for y := 0; y < s.mapGrid.Height; y++ {
		for x := 0; x < s.mapGrid.Width; x++ {
			tileIndex := y*s.mapGrid.Width + x
			state := s.mapGrid.TileStates[tileIndex]

			if state.FootTraffic > s.trafficTrigger {
				if prng.Float32() < s.spawnChance {
					// Spawn a DiseaseEntity
					e := world.NewEntity()
					posID := ecs.ComponentID[components.Position](world)
					diseaseID := ecs.ComponentID[components.DiseaseEntity](world)

					world.Add(e, posID, diseaseID)

					pos := (*components.Position)(world.Get(e, posID))
					pos.X = float32(x)
					pos.Y = float32(y)

					disease := (*components.DiseaseEntity)(world.Get(e, diseaseID))
					disease.ID = prng.Uint32() // Generate a unique identifier for this plague
					// Base lethality between 50 and 90
					disease.Lethality = uint8(50 + prng.IntN(41))
				}
			}
		}
	}

	// 2. Extract all active diseases into a flat array for DOD iteration
	type diseaseData struct {
		x         int
		y         int
		id        uint32
		lethality uint8
	}
	var activeDiseases []diseaseData

	diseasePosID := ecs.ComponentID[components.Position](world)
	diseaseEntID := ecs.ComponentID[components.DiseaseEntity](world)
	diseaseFilter := ecs.All(diseasePosID, diseaseEntID)

	diseaseQuery := world.Query(diseaseFilter)
	for diseaseQuery.Next() {
		pos := (*components.Position)(diseaseQuery.Get(diseasePosID))
		disease := (*components.DiseaseEntity)(diseaseQuery.Get(diseaseEntID))

		activeDiseases = append(activeDiseases, diseaseData{
			x:         int(pos.X),
			y:         int(pos.Y),
			id:        disease.ID,
			lethality: disease.Lethality,
		})
	}

	if len(activeDiseases) == 0 {
		return // No diseases to process
	}

	// 3. Query all vulnerable entities (Position + Genetics)
	posID := ecs.ComponentID[components.Position](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	immunityID := ecs.ComponentID[components.ImmunityTag](world)

	targetFilter := ecs.All(posID, genID)
	query := world.Query(targetFilter)

	for query.Next() {
		pos := (*components.Position)(query.Get(posID))
		gen := (*components.GenomeComponent)(query.Get(genID))
		entX, entY := int(pos.X), int(pos.Y)

		var immunity *components.ImmunityTag
		if query.Has(immunityID) {
			immunity = (*components.ImmunityTag)(query.Get(immunityID))
		}

		for _, disease := range activeDiseases {
			// Check if entity is on the same tile as the disease
			if entX == disease.x && entY == disease.y {
				// Check for immunity
				isImmune := false
				if immunity != nil {
					for _, immuneTo := range immunity.ImmuneTo {
						if immuneTo == disease.id {
							isImmune = true
							break
						}
					}
				}

				if isImmune {
					continue // Ignore this disease, entity is immune
				}

				// Evaluate lethality mathematically
				// Base health is 0-255. Lethality is 50-90.
				// A health roll lower than lethality means death.
				healthRoll := prng.IntN(100) + int(gen.Health)

				if healthRoll < int(disease.lethality) {
					// Failed the check, entity dies
					s.toRemove = append(s.toRemove, query.Entity())
					break // Dead entities don't need to process other diseases
				} else {
					// Survived the check, gain immunity
					s.toImmunize = append(s.toImmunize, immuneData{
						entity:    query.Entity(),
						diseaseID: disease.id,
					})
				}
			}
		}
	}

	// 4. Process mutations outside the query loop to prevent ECS panics
	for _, e := range s.toRemove {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}

	for _, data := range s.toImmunize {
		if world.Alive(data.entity) {
			if !world.Has(data.entity, immunityID) {
				world.Add(data.entity, immunityID)
			}
			immunity := (*components.ImmunityTag)(world.Get(data.entity, immunityID))

			// Check if we already appended this disease ID recently to avoid duplicates
			hasImmunity := false
			for _, immuneTo := range immunity.ImmuneTo {
				if immuneTo == data.diseaseID {
					hasImmunity = true
					break
				}
			}

			if !hasImmunity {
				immunity.ImmuneTo = append(immunity.ImmuneTo, data.diseaseID)
			}
		}
	}

	// Clear slices for reuse to prevent GC spikes
	s.toRemove = s.toRemove[:0]
	s.toImmunize = s.toImmunize[:0]
}
