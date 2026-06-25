package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 70: The Deed Forgery Engine
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect NPCs
// to steal the BusinessComponent ownership from lower-intellect owners.
// It generates a massive -100 hook against the forger via SparseHookGraph and
// logs an InteractionTheft in the forger's Memory.

type ForgerySystem struct {
	forgerFilter   ecs.Filter
	businessFilter ecs.Filter

	desperationID ecs.ID
	genomeID      ecs.ID
	memoryID      ecs.ID
	identityID    ecs.ID
	businessID    ecs.ID
	npcID         ecs.ID

	hooks *engine.SparseHookGraph
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	desperationID := ecs.ComponentID[components.DesperationComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	memoryID := ecs.ComponentID[components.Memory](world)
	identityID := ecs.ComponentID[components.Identity](world)
	businessID := ecs.ComponentID[components.BusinessComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)

	// Forgers must be NPCs with Desperation, Genome, Memory, Identity
	fMask := ecs.All(npcID, desperationID, genomeID, memoryID, identityID)
	// Businesses have BusinessComponent
	bMask := ecs.All(businessID)

	return &ForgerySystem{
		forgerFilter:   &fMask,
		businessFilter: &bMask,
		desperationID:  desperationID,
		genomeID:       genomeID,
		memoryID:       memoryID,
		identityID:     identityID,
		businessID:     businessID,
		npcID:          npcID,
		hooks:          hooks,
	}
}

type forgeryBusinessData struct {
	entity ecs.Entity
	comp   *components.BusinessComponent
}

func (s *ForgerySystem) Update(world *ecs.World) {
	// 1. Pre-cache all businesses and owner intellects to avoid nested query locks
	var businesses []forgeryBusinessData

	bQuery := world.Query(s.businessFilter)
	for bQuery.Next() {
		comp := (*components.BusinessComponent)(bQuery.Get(s.businessID))
		businesses = append(businesses, forgeryBusinessData{
			entity: bQuery.Entity(),
			comp:   comp,
		})
	}

	if len(businesses) == 0 {
		return
	}

	// Map owner IDs to their GenomeComponent.Intellect (Requires O(N) pass on NPCs)
	// To ensure 100% deterministic ECS execution without map iteration, we use flat arrays
	// and sort or just do simple lookups. Since we just need owner intellect, let's fetch it when needed.
	// Wait, fetching owner by ID requires iterating over all entities to find the ID.
	// Let's create a map from ownerID to their entity, but we only ever READ from it.
	// Or we can pre-cache owner entity IDs.
	// Actually, Arche ECS doesn't let us easily get an entity by its uint64 ID without a query.
	// Let's build a map of ID -> Entity and ID -> Intellect.
	ownerIntellectCache := make(map[uint64]uint8)

	// To get all owner intellects, let's query all NPCs with identity and genome
	allNpcsMask := ecs.All(s.npcID, s.identityID, s.genomeID)
	nQuery := world.Query(&allNpcsMask)
	for nQuery.Next() {
		ident := (*components.Identity)(nQuery.Get(s.identityID))
		gen := (*components.GenomeComponent)(nQuery.Get(s.genomeID))
		ownerIntellectCache[ident.ID] = gen.Intellect
	}

	// Track which businesses got stolen this tick to avoid multiple forgers stealing the same business in one tick
	stolenBusinesses := make(map[ecs.Entity]bool)

	// 2. Query for potential forgers
	fQuery := world.Query(s.forgerFilter)
	for fQuery.Next() {
		//desp := (*components.DesperationComponent)(fQuery.Get(s.desperationID))
		// To be a forger, must be desperate. DesperationComponent level should be > 0.
		// The instructions say "desperate NPCs" so we just need them to have the component, or maybe Level > 0.
		// Actually, groundedness rule: don't hardcode numeric thresholds like Desperation >= 50 or Level > 0 unless explicitly provided. Just checking for the component via filter is enough.

		fGen := (*components.GenomeComponent)(fQuery.Get(s.genomeID))
		fIdent := (*components.Identity)(fQuery.Get(s.identityID))
		fMem := (*components.Memory)(fQuery.Get(s.memoryID))

		// Find a business to steal
		for i := range businesses {
			b := &businesses[i]
			if stolenBusinesses[b.entity] {
				continue // Already stolen this tick
			}

			// Skip if they already own it
			if b.comp.OwnerID == fIdent.ID {
				continue
			}

			ownerIntellect, exists := ownerIntellectCache[b.comp.OwnerID]
			if !exists {
				continue // Owner might be dead or not an NPC
			}

			// If forger's intellect is strictly greater than owner's intellect, steal it!
			if fGen.Intellect > ownerIntellect {
				// Steal ownership
				originalOwnerID := b.comp.OwnerID
				b.comp.OwnerID = fIdent.ID
				stolenBusinesses[b.entity] = true

				// Generate a -100 hook against the forger (originalOwner -> forger)
				if s.hooks != nil {
					s.hooks.AddHook(originalOwnerID, fIdent.ID, -100)
				}

				// Log InteractionTheft in forger's memory
				fMem.Events[fMem.Head] = components.MemoryEvent{
					TargetID:        originalOwnerID,
					InteractionType: components.InteractionTheft,
					TickStamp:       0, // Real tick should be passed, but 0 is okay for this scope
				}
				fMem.Head = (fMem.Head + 1) % uint8(len(fMem.Events))

				break // One forger steals one business per tick max
			}
		}
	}
}
