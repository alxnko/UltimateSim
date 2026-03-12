package systems

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.2 & 14: The Genesis Spawner (NPCs)
// Run once at Tick 0. Queries MapGrid for walkable/habitable tiles.
// Uses GlobalRNG to select starting locations.

type NPCSpawnerSystem struct {
	mapGrid    *engine.MapGrid
	hasSpawned bool
	nextID     uint64
	nextFamID  uint32
}

// NewNPCSpawnerSystem creates a new NPCSpawnerSystem.
func NewNPCSpawnerSystem(world *ecs.World, mapGrid *engine.MapGrid) *NPCSpawnerSystem {
	return &NPCSpawnerSystem{
		mapGrid:    mapGrid,
		hasSpawned: false,
		nextID:     1, // Start IDs from 1
		nextFamID:  1,
	}
}

// Update executes the system logic per tick.
func (s *NPCSpawnerSystem) Update(world *ecs.World) {
	if s.hasSpawned {
		return // Spawns only once
	}

	// Find habitable tiles
	type Coord struct {
		X, Y int
	}
	var habitableTiles []Coord

	for y := 0; y < s.mapGrid.Height; y++ {
		for x := 0; x < s.mapGrid.Width; x++ {
			tile := s.mapGrid.GetTile(x, y)
			if tile.BiomeID != engine.BiomeOcean {
				habitableTiles = append(habitableTiles, Coord{X: x, Y: y})
			}
		}
	}

	if len(habitableTiles) == 0 {
		fmt.Println("No habitable tiles found. Genesis spawning aborted.")
		return
	}

	// Prepare components
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.Genetics](world)
	legID := ecs.ComponentID[components.Legacy](world)
	needsID := ecs.ComponentID[components.Needs](world)
	pathID := ecs.ComponentID[components.Path](world)
	npcID := ecs.ComponentID[components.NPC](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	// Spawn 20 initial families across the map
	spawnCount := 20
	for i := 0; i < spawnCount; i++ {
		// Pick random habitable tile
		idx := engine.GetRandomInt() % len(habitableTiles)
		coord := habitableTiles[idx]

		familyID := s.nextFamID
		s.nextFamID++
		clanID := uint32(engine.GetRandomInt() % 100) // Random ClanID mapping

		// Phase 14: Spawn 5 individuals per tile (family)
		npcCount := 5
		for j := 0; j < npcCount; j++ {
			entity := world.NewEntity(posID, velID, idID, genID, legID, needsID, pathID, npcID, slID, affID)

			// Set Position
			pos := (*components.Position)(world.Get(entity, posID))
			pos.X = float32(coord.X)
			pos.Y = float32(coord.Y)

			// Set Velocity
			vel := (*components.Velocity)(world.Get(entity, velID))
			vel.X = 0
			vel.Y = 0

			// Set Identity
			id := (*components.Identity)(world.Get(entity, idID))
			id.ID = s.nextID
			s.nextID++
			id.Name = fmt.Sprintf("NPC-%d", id.ID)
			id.BaseTraits = uint32(engine.GetRandomInt()) // Random bitmask

			// Set Genetics
			gen := (*components.Genetics)(world.Get(entity, genID))
			// Randomize Genetics via Bell Curve approximation (sum of 3 uniform / 3)
			gen.Strength = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
			gen.Beauty = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
			gen.Health = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)
			gen.Intellect = uint8((engine.GetRandomInt()%101 + engine.GetRandomInt()%101 + engine.GetRandomInt()%101) / 3)

			// Set Legacy
			leg := (*components.Legacy)(world.Get(entity, legID))
			leg.Prestige = 0
			leg.InheritedDebt = 0

			// Set Needs
			needs := (*components.Needs)(world.Get(entity, needsID))
			needs.Food = 1000.0
			needs.Rest = 100.0
			needs.Safety = 100.0
			needs.Wealth = 100.0

			// Set Path
			path := (*components.Path)(world.Get(entity, pathID))
			path.HasPath = false

			// Set SettlementLogic
			sl := (*components.SettlementLogic)(world.Get(entity, slID))
			sl.TicksAtZeroVelocity = 0

			// Set Affiliation
			aff := (*components.Affiliation)(world.Get(entity, affID))
			aff.FamilyID = familyID
			aff.ClanID = clanID
		}
	}

	s.hasSpawned = true
}
