package systems

import (
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 31: Systemic Entropy (Natural Disasters)
// Occasionally spawns an earthquake/disaster at a random map grid coordinate.
// Destroys resources, infrastructure, and damages NPCs or Ruins villages within radius.

type NaturalDisasterSystem struct {
	mapGrid     *engine.MapGrid
	currentTick uint64
	spawnChance float32
}

func NewNaturalDisasterSystem(world *ecs.World, mapGrid *engine.MapGrid) *NaturalDisasterSystem {
	return &NaturalDisasterSystem{
		mapGrid:     mapGrid,
		currentTick: 0,
		spawnChance: 0.001, // 0.1% chance per tick to spawn a disaster
	}
}

func (s *NaturalDisasterSystem) Update(world *ecs.World) {
	s.currentTick++

	// Seed local PRNG to guarantee determinism without polluting global state
	var seed [32]byte
	seed[0] = byte(s.currentTick)
	seed[1] = byte(s.currentTick >> 8)
	prng := rand.New(rand.NewChaCha8(seed))

	// 1. Spawning Disasters
	if prng.Float32() < s.spawnChance {
		e := world.NewEntity()
		posID := ecs.ComponentID[components.Position](world)
		disasterID := ecs.ComponentID[components.NaturalDisasterEntity](world)
		disasterCompID := ecs.ComponentID[components.DisasterComponent](world)

		world.Add(e, posID, disasterID, disasterCompID)

		pos := (*components.Position)(world.Get(e, posID))
		pos.X = float32(prng.IntN(s.mapGrid.Width))
		pos.Y = float32(prng.IntN(s.mapGrid.Height))

		disaster := (*components.DisasterComponent)(world.Get(e, disasterCompID))
		disaster.RadiusSquared = float32(25 + prng.IntN(75)) // 25 to 100 squared radius
		disaster.Strength = float32(50 + prng.IntN(50)) // 50 to 100 strength
		disaster.Type = 1 // Earthquake

		// We will immediately process the disaster effects here to simulate an instant event,
		// and then despawn the entity since disasters are instant.

		// 2. Modify MapGrid within radius
		centerX := int(pos.X)
		centerY := int(pos.Y)
		radius := int(disaster.RadiusSquared) // For map grid, rough check is fine

		for y := centerY - 10; y <= centerY + 10; y++ {
			for x := centerX - 10; x <= centerX + 10; x++ {
				if x < 0 || x >= s.mapGrid.Width || y < 0 || y >= s.mapGrid.Height {
					continue
				}

				dx := x - centerX
				dy := y - centerY
				if dx*dx + dy*dy <= radius {
					idx := y*s.mapGrid.Width + x

					// Shatter ResourceDepot values
					s.mapGrid.Resources[idx].WoodValue = 0
					s.mapGrid.Resources[idx].StoneValue = 0

					// Reset TileState.FootTraffic (wiping roads)
					s.mapGrid.TileStates[idx].FootTraffic = 0
				}
			}
		}

		// 3. Process Entities (NPCs and Villages)

		// NPCs
		npcID := ecs.ComponentID[components.NPC](world)
		vitalsID := ecs.ComponentID[components.VitalsComponent](world)

		npcFilter := ecs.All(npcID, posID, vitalsID)
		npcQuery := world.Query(npcFilter)

		for npcQuery.Next() {
			npcPos := (*components.Position)(npcQuery.Get(posID))
			dx := npcPos.X - pos.X
			dy := npcPos.Y - pos.Y
			if dx*dx + dy*dy <= disaster.RadiusSquared {
				// Caught in the blast
				vitals := (*components.VitalsComponent)(npcQuery.Get(vitalsID))
				vitals.Pain += disaster.Strength

				// Systemic Emergence: If pain is extreme, they might die instantly or be incapacitated
				// DeathSystem handles starvation, but we can also use MetabolismSystem's pain mechanic
				// or just rely on the existing vitals integration.
				}
				}

				// Villages
				villageID := ecs.ComponentID[components.Village](world)
				storageID := ecs.ComponentID[components.StorageComponent](world)

				villageFilter := ecs.All(villageID, posID, storageID)
				villageQuery := world.Query(villageFilter)

				for villageQuery.Next() {
				villagePos := (*components.Position)(villageQuery.Get(posID))
				dx := villagePos.X - pos.X
				dy := villagePos.Y - villagePos.Y
				if dx*dx+dy*dy <= disaster.RadiusSquared {
				storage := (*components.StorageComponent)(villageQuery.Get(storageID))
				// Massive destruction of storage
				storage.Food = 0
				storage.Wood = 0
				storage.Stone = 0
				storage.Iron = 0
				}
				}

				// Despawn the disaster entity since its effect is instant
				world.RemoveEntity(e)
				}
				}
