package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 61 - The Biological Sabotage Engine
// BiologicalSabotageSystem bridges Social Hierarchy, Economy, Biology, and Justice.
// NPCs with extreme negative hooks against a Ruler physically visit their Village,
// halve its Food, spawn a DiseaseEntity (biological fallout), and receive a CrimeMarker.

type BiologicalSabotageSystem struct {
	hookGraph   *engine.SparseHookGraph
	tickCounter uint64

	npcFilter ecs.Filter

	npcID   ecs.ID
	posID   ecs.ID
	affID   ecs.ID
	idID    ecs.ID
	villID  ecs.ID
	storID  ecs.ID
	adminID ecs.ID
	crimeID ecs.ID
	disID   ecs.ID
}

func NewBiologicalSabotageSystem(world *ecs.World, hooks *engine.SparseHookGraph) *BiologicalSabotageSystem {
	npcID := ecs.ComponentID[components.NPC](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	idID := ecs.ComponentID[components.Identity](world)

	npcMask := ecs.All(npcID, posID, affID, idID)

	return &BiologicalSabotageSystem{
		hookGraph:   hooks,
		tickCounter: 0,
		npcFilter:   &npcMask,
		npcID:       npcID,
		posID:       posID,
		affID:       affID,
		idID:        idID,
		villID:      ecs.ComponentID[components.Village](world),
		storID:      ecs.ComponentID[components.StorageComponent](world),
		adminID:     ecs.ComponentID[components.AdministrationMarker](world),
		crimeID:     ecs.ComponentID[components.CrimeMarker](world),
		disID:       ecs.ComponentID[components.DiseaseEntity](world),
	}
}

type sabotageVillageData struct {
	Entity  ecs.Entity
	RulerID uint64
	X       float32
	Y       float32
	Food    uint32
}

func (s *BiologicalSabotageSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process logic periodically to save CPU (staggered from Phase 57 Class Warfare)
	if s.tickCounter%53 != 0 {
		return
	}

	if s.hookGraph == nil {
		return
	}

	// DOD O(1) Pre-caching strategy:
	// We map CityID -> sabotageVillageData (Entity, Ruler ID, Pos X/Y, Food)
	villages := make(map[uint32]*sabotageVillageData)

	// 1. Gather all Administration Rulers
	adminQuery := world.Query(ecs.All(s.adminID, s.affID, s.idID))
	for adminQuery.Next() {
		aff := (*components.Affiliation)(adminQuery.Get(s.affID))
		ident := (*components.Identity)(adminQuery.Get(s.idID))

		villages[aff.CityID] = &sabotageVillageData{
			RulerID: ident.ID,
		}
	}

	// 2. Gather City Storage & Position (the Village entity)
	villageQuery := world.Query(ecs.All(s.villID, s.affID, s.posID, s.storID))
	for villageQuery.Next() {
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))

		if data, exists := villages[aff.CityID]; exists {
			pos := (*components.Position)(villageQuery.Get(s.posID))
			stor := (*components.StorageComponent)(villageQuery.Get(s.storID))

			data.Entity = villageQuery.Entity()
			data.X = pos.X
			data.Y = pos.Y
			data.Food = stor.Food
		}
	}

	sabotages := make([]sabotageVillageData, 0)
	punishments := make([]ecs.Entity, 0)

	// 3. Main Loop: Iterate all NPCs
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		ident := (*components.Identity)(npcQuery.Get(s.idID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))

		// Get local ruler data
		vData, exists := villages[aff.CityID]
		if !exists || vData.Entity.IsZero() {
			continue
		}

		// Self check
		if ident.ID == vData.RulerID {
			continue
		}

		// Check grudge
		grudge := s.hookGraph.GetHook(ident.ID, vData.RulerID)
		if grudge <= -100 {
			// Ensure physical proximity to the village to execute sabotage
			dx := pos.X - vData.X
			dy := pos.Y - vData.Y
			distSq := (dx * dx) + (dy * dy)

			if distSq < 2.0 {
				// Prevent tick spamming - spend the hook
				s.hookGraph.AddHook(ident.ID, vData.RulerID, 100)

				sabotages = append(sabotages, *vData)
				punishments = append(punishments, npcQuery.Entity())

				// Break to allow only one sabotage per village per tick, keeping deterministic arrays simple
				delete(villages, aff.CityID)
			}
		}
	}

	// 4. Apply Sabotage structurally outside query
	for _, sData := range sabotages {
		if world.Alive(sData.Entity) {
			stor := (*components.StorageComponent)(world.Get(sData.Entity, s.storID))
			stor.Food /= 2

			// Spawn biological fallout
			diseaseEnt := world.NewEntity()
			world.Add(diseaseEnt, s.posID, s.disID)

			dPos := (*components.Position)(world.Get(diseaseEnt, s.posID))
			dPos.X = sData.X
			dPos.Y = sData.Y

			dis := (*components.DiseaseEntity)(world.Get(diseaseEnt, s.disID))
			dis.ID = 999       // Arbitrary Plague ID for poisoned food
			dis.Lethality = 80 // Extremely lethal
		}
	}

	for _, pEnt := range punishments {
		if world.Alive(pEnt) {
			if !world.Has(pEnt, s.crimeID) {
				world.Add(pEnt, s.crimeID)
			}
			cm := (*components.CrimeMarker)(world.Get(pEnt, s.crimeID))
			cm.CrimeLevel = 3 // Severe Crime
			cm.Bounty = 500
		}
	}
}
