package systems

import (
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 61 - The Biological Sabotage Engine
// Bridging Social Hierarchy (Hooks), Economy (Storage), Biology (Diseases), and Justice.
// NPCs with extreme negative hooks (<= -100) against a City's Ruler will execute Biological Sabotage
// if they are physically adjacent to the Village. They halve the food supply, spawn a high lethality Disease,
// neutralize their hook, and get tagged with a CrimeMarker.

type BiologicalSabotageSystem struct {
	hooks       *engine.SparseHookGraph
	tickCounter uint64

	villageID ecs.ID
	storID    ecs.ID
	affID     ecs.ID
	posID     ecs.ID

	adminID ecs.ID
	idID    ecs.ID

	npcID     ecs.ID
	crimeID   ecs.ID
	diseaseID ecs.ID
}

func NewBiologicalSabotageSystem(world *ecs.World, hooks *engine.SparseHookGraph) *BiologicalSabotageSystem {
	return &BiologicalSabotageSystem{
		hooks:       hooks,
		tickCounter: 0,

		villageID: ecs.ComponentID[components.Village](world),
		storID:    ecs.ComponentID[components.StorageComponent](world),
		affID:     ecs.ComponentID[components.Affiliation](world),
		posID:     ecs.ComponentID[components.Position](world),

		adminID: ecs.ComponentID[components.AdministrationMarker](world),
		idID:    ecs.ComponentID[components.Identity](world),

		npcID:     ecs.ComponentID[components.NPC](world),
		crimeID:   ecs.ComponentID[components.CrimeMarker](world),
		diseaseID: ecs.ComponentID[components.DiseaseEntity](world),
	}
}

type villageSabotageData struct {
	Entity ecs.Entity
	CityID uint32
	X      float32
	Y      float32
	Stor   *components.StorageComponent
}

type sabotageEvent struct {
	NPC       ecs.Entity
	NPCID     uint64
	TargetID  uint64
	HookValue int
	Village   ecs.Entity
	X         float32
	Y         float32
}

func (s *BiologicalSabotageSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process logic periodically to save CPU
	if s.tickCounter%50 != 0 {
		return
	}

	if s.hooks == nil {
		return
	}

	// 1. Gather all Administration Rulers mapped by CityID
	cityRulers := make(map[uint32]uint64)
	adminQuery := world.Query(filter.All(s.adminID, s.affID, s.idID))
	for adminQuery.Next() {
		aff := (*components.Affiliation)(adminQuery.Get(s.affID))
		ident := (*components.Identity)(adminQuery.Get(s.idID))
		cityRulers[aff.CityID] = ident.ID
	}

	// 2. Gather Village Data
	var villages []villageSabotageData
	villageQuery := world.Query(filter.All(s.villageID, s.affID, s.posID, s.storID))
	for villageQuery.Next() {
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		pos := (*components.Position)(villageQuery.Get(s.posID))
		stor := (*components.StorageComponent)(villageQuery.Get(s.storID))

		if _, exists := cityRulers[aff.CityID]; exists {
			villages = append(villages, villageSabotageData{
				Entity: villageQuery.Entity(),
				CityID: aff.CityID,
				X:      pos.X,
				Y:      pos.Y,
				Stor:   stor,
			})
		}
	}

	// 3. Evaluate NPCs
	var sabotages []sabotageEvent
	npcQuery := world.Query(filter.All(s.npcID, s.idID, s.posID))
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(s.idID))
		pos := (*components.Position)(npcQuery.Get(s.posID))

		outgoingHooks := s.hooks.GetAllHooks(ident.ID)

		var worstTargetID uint64 = 0
		var worstHook int = -99 // Must be <= -100

		for targetID, hookVal := range outgoingHooks {
			if hookVal < worstHook {
				worstHook = hookVal
				worstTargetID = targetID
			} else if hookVal == worstHook && targetID > worstTargetID {
				worstTargetID = targetID // deterministic tiebreaker
			}
		}

		if worstTargetID == 0 {
			continue // No extreme grudge
		}

		// Find if this target is a ruler of a nearby village
		for i := 0; i < len(villages); i++ {
			v := &villages[i]
			rulerID := cityRulers[v.CityID]

			if rulerID == worstTargetID {
				// Check distance
				dx := pos.X - v.X
				dy := pos.Y - v.Y
				distSq := dx*dx + dy*dy

				if distSq < 2.0 {
					sabotages = append(sabotages, sabotageEvent{
						NPC:       npcQuery.Entity(),
						NPCID:     ident.ID,
						TargetID:  worstTargetID,
						HookValue: worstHook,
						Village:   v.Entity,
						X:         v.X,
						Y:         v.Y,
					})
					break // One sabotage per NPC per tick
				}
			}
		}
	}

	// 4. Execute Sabotages structurally outside the query lock
	var seed [32]byte
	seed[0] = byte(s.tickCounter)
	seed[1] = byte(s.tickCounter >> 8)
	prng := rand.New(rand.NewChaCha8(seed))

	for _, sab := range sabotages {
		if !world.Alive(sab.NPC) || !world.Alive(sab.Village) {
			continue
		}

		// 4a. Halve Food Supply
		if world.Has(sab.Village, s.storID) {
			stor := (*components.StorageComponent)(world.Get(sab.Village, s.storID))
			stor.Food = stor.Food / 2
		}

		// 4b. Spawn Disease Entity
		diseaseEnt := world.NewEntity(s.posID, s.diseaseID)
		dPos := (*components.Position)(world.Get(diseaseEnt, s.posID))
		dPos.X = sab.X
		dPos.Y = sab.Y

		dComp := (*components.DiseaseEntity)(world.Get(diseaseEnt, s.diseaseID))
		dComp.ID = prng.Uint32()
		dComp.Lethality = uint8(80 + prng.IntN(21)) // 80 to 100

		// 4c. Neutralize Hook
		s.hooks.AddHook(sab.NPCID, sab.TargetID, -sab.HookValue)

		// 4d. Tag with CrimeMarker
		if !world.Has(sab.NPC, s.crimeID) {
			world.Add(sab.NPC, s.crimeID)
			crime := (*components.CrimeMarker)(world.Get(sab.NPC, s.crimeID))
			crime.CrimeLevel = 3 // High crime
			crime.Bounty = 500
		}
	}
}
