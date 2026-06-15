package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 70: The Deed Forgery Engine
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect NPCs
// to steal the BusinessComponent ownership from lower-intellect owners.

type ForgerySystem struct {
	forgerFilter   ecs.Filter
	businessFilter ecs.Filter

	npcID         ecs.ID
	identID       ecs.ID
	despID        ecs.ID
	genomeID      ecs.ID
	memID         ecs.ID
	businessID    ecs.ID
	businessTagID ecs.ID

	hooks *engine.SparseHookGraph
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	memID := ecs.ComponentID[components.Memory](world)

	businessTagID := ecs.ComponentID[components.BusinessEntity](world)
	businessID := ecs.ComponentID[components.BusinessComponent](world)

	fMask := ecs.All(npcID, identID, despID, genomeID, memID)
	bMask := ecs.All(businessTagID, businessID)

	return &ForgerySystem{
		forgerFilter:   &fMask,
		businessFilter: &bMask,
		npcID:          npcID,
		identID:        identID,
		despID:         despID,
		genomeID:       genomeID,
		memID:          memID,
		businessID:     businessID,
		businessTagID:  businessTagID,
		hooks:          hooks,
	}
}

func (s *ForgerySystem) Update(world *ecs.World) {
	// 1. Identify all potential forgers
	// Desperation > 50 and Intellect > 70
	type forgerData struct {
		entity    ecs.Entity
		id        uint64
		intellect uint8
	}
	var forgers []forgerData

	fQuery := world.Query(s.forgerFilter)
	for fQuery.Next() {
		desp := (*components.DesperationComponent)(fQuery.Get(s.despID))
		if desp.Level > 50 {
			genome := (*components.GenomeComponent)(fQuery.Get(s.genomeID))
			if genome.Intellect > 70 {
				ident := (*components.Identity)(fQuery.Get(s.identID))
				forgers = append(forgers, forgerData{
					entity:    fQuery.Entity(),
					id:        ident.ID,
					intellect: genome.Intellect,
				})
			}
		}
	}

	if len(forgers) == 0 {
		return
	}

	// 2. Identify vulnerable businesses
	// Cache all NPCs into a map for quick intellect lookup (deterministic iteration keys later)
	// Actually, just cache all NPCs into an array, and search through it or just query the world for the owner ID
	// Let's create a map of NPC ID -> Intellect for quick lookup, we won't iterate the map directly.
	npcIntellectMap := make(map[uint64]uint8)
	npcQuery := world.Query(ecs.All(s.npcID, s.identID, s.genomeID))
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(s.identID))
		genome := (*components.GenomeComponent)(npcQuery.Get(s.genomeID))
		npcIntellectMap[ident.ID] = genome.Intellect
	}

	// Iterate over businesses
	bQuery := world.Query(s.businessFilter)
	for bQuery.Next() {
		biz := (*components.BusinessComponent)(bQuery.Get(s.businessID))
		ownerID := biz.OwnerID

		// If owner intellect is missing or low, it's vulnerable
		ownerIntellect, exists := npcIntellectMap[ownerID]
		if !exists {
			ownerIntellect = 0
		}

		// Find a forger that has higher intellect than owner
		// and hasn't already forged something this tick (to keep it simple)
		for i := 0; i < len(forgers); i++ {
			f := &forgers[i]
			if f.intellect > ownerIntellect {
				// Forgery succeeds!

				// 1. Transfer ownership
				biz.OwnerID = f.id

				// 2. Generate massive negative hook from the old owner to the forger
				if s.hooks != nil && exists {
					s.hooks.AddHook(ownerID, f.id, -100)
				}

				// 3. Log an InteractionTheft in the forger's Memory buffer
				if world.Alive(f.entity) {
					mem := (*components.Memory)(world.Get(f.entity, s.memID))

					event := components.MemoryEvent{
						TargetID:        ownerID,
						TickStamp:       0, // We could use a TickManager, but 0 is fine here as per other systems
						InteractionType: components.InteractionTheft,
					}

					// Ring buffer pattern
					mem.Events[mem.Head] = event
					mem.Head = (mem.Head + 1) % uint8(len(mem.Events))
				}

				// Remove forger from list so they only steal one business per tick
				forgers = append(forgers[:i], forgers[i+1:]...)

				// Move to next business
				break
			}
		}

		if len(forgers) == 0 {
			break
		}
	}
}
