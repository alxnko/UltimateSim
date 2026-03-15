package systems

import (
	"log"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: DeathSystem
// Scans for any Entity where Needs.Food <= 0. If found, trigger the Despawn pipeline.

// itemSpawnData holds temporary data extracted from a dying entity to spawn a Legend Item
type itemSpawnData struct {
	posX     float32
	posY     float32
	nameID   uint32
	prestige uint32
}

// Phase 25.1: Succession Engine cache
type heirData struct {
	DeadID        uint64
	FamilyID      uint32
	Prestige      uint32
	InheritedDebt uint32
	OutgoingHooks map[uint64]int
	IncomingHooks map[uint64]int
	Artifact      *components.LegendComponent // Phase 32.1: Artifact Inheritance
}

type DeathSystem struct {
	filter       ecs.Filter
	toRemove     []ecs.Entity
	itemsToSpawn []itemSpawnData
	deadPos      []components.Position // Phase 20.3: Used for mapping trauma
	hookGraph    *engine.SparseHookGraph
	heirs        []heirData
}

// NewDeathSystem creates a new DeathSystem.
func NewDeathSystem(world *ecs.World, hooks *engine.SparseHookGraph) *DeathSystem {
	// Query entities that have Needs
	needsID := ecs.ComponentID[components.Needs](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	// Phase 05.3: Arche-Go Component Filters
	// Explicitly build Without(ruinID) to skip over ruins.
	mask := ecs.All(needsID).Without(ruinID)

	return &DeathSystem{
		filter:       &mask,
		toRemove:     make([]ecs.Entity, 0, 100),
		itemsToSpawn: make([]itemSpawnData, 0, 10),
		deadPos:      make([]components.Position, 0, 100),
		hookGraph:    hooks,
		heirs:        make([]heirData, 0, 10),
	}
}

// Update executes the system logic per tick.
func (s *DeathSystem) Update(world *ecs.World) {
	needsID := ecs.ComponentID[components.Needs](world)
	legacyID := ecs.ComponentID[components.Legacy](world)
	identityID := ecs.ComponentID[components.Identity](world)
	positionID := ecs.ComponentID[components.Position](world)
	affilID := ecs.ComponentID[components.Affiliation](world)

	// We must register Component IDs BEFORE query
	equipID := ecs.ComponentID[components.EquipmentComponent](world)

	// Collect entities to remove to avoid modifying the world while iterating
	// Reset the slice length to 0, retaining capacity to avoid GC pressure
	s.toRemove = s.toRemove[:0]
	s.itemsToSpawn = s.itemsToSpawn[:0]
	s.deadPos = s.deadPos[:0]
	s.heirs = s.heirs[:0]

	query := world.Query(s.filter)
	for query.Next() {
		needs := (*components.Needs)(query.Get(needsID))

		if needs.Food <= 0 {
			s.toRemove = append(s.toRemove, query.Entity())

			var posX, posY float32
			if query.Has(positionID) {
				pos := (*components.Position)(query.Get(positionID))
				posX = pos.X
				posY = pos.Y
				s.deadPos = append(s.deadPos, *pos)
			}

			// Phase 09.5: Item Inheritance logic
			// Only spawn if they actually have an equipped weapon
			if query.Has(legacyID) {
				legacy := (*components.Legacy)(query.Get(legacyID))

				var hasEquippedArtifact bool
				var artifactNameID uint32
				if query.Has(equipID) {
					equip := (*components.EquipmentComponent)(query.Get(equipID))
					if equip.Equipped {
						hasEquippedArtifact = true
						artifactNameID = equip.Weapon.NameID
					}
				}

				if legacy.Prestige >= components.ExtremePrestigeThreshold && hasEquippedArtifact {
					s.itemsToSpawn = append(s.itemsToSpawn, itemSpawnData{
						posX:     posX,
						posY:     posY,
						prestige: legacy.Prestige,
						nameID:   artifactNameID,
					})
				}
			}

			// Phase 25.1 & Phase 32.1: Social Legacy, Succession Engine, and Artifact Inheritance
			// Cache traits for potential heirs
			if s.hookGraph != nil && query.Has(identityID) && query.Has(affilID) {
				ident := (*components.Identity)(query.Get(identityID))
				affil := (*components.Affiliation)(query.Get(affilID))
				var pres, debt uint32
				if query.Has(legacyID) {
					leg := (*components.Legacy)(query.Get(legacyID))
					pres = leg.Prestige
					debt = leg.InheritedDebt
				}

				var artifact *components.LegendComponent
				if query.Has(equipID) {
					equip := (*components.EquipmentComponent)(query.Get(equipID))
					if equip.Equipped {
						artifactCopy := equip.Weapon // Struct copy to preserve data after despawn
						artifact = &artifactCopy
					}
				}

				outgoing := s.hookGraph.GetAllHooks(ident.ID)
				incoming := s.hookGraph.GetAllIncomingHooks(ident.ID)

				// Only trigger succession if there's a reason (hooks or debt or prestige or artifact)
				if len(outgoing) > 0 || len(incoming) > 0 || pres > 0 || debt > 0 || artifact != nil {
					s.heirs = append(s.heirs, heirData{
						DeadID:        ident.ID,
						FamilyID:      affil.FamilyID,
						Prestige:      pres,
						InheritedDebt: debt,
						OutgoingHooks: outgoing,
						IncomingHooks: incoming,
						Artifact:      artifact,
					})
				}
			}

			// log root causes to standard output
			// ecs.Entity formats safely to string via %v
			log.Printf("Entity %v despawned due to starvation (Food <= 0)", query.Entity())
		}
	}

	// Phase 20.3: Traumatic Traditions
	// Map dead positions to jurisdictions to increment Trauma
	if len(s.deadPos) > 0 {
		jurID := ecs.ComponentID[components.JurisdictionComponent](world)
		posID := ecs.ComponentID[components.Position](world)
		jurQuery := world.Query(ecs.All(jurID, posID))

		type jurData struct {
			comp *components.JurisdictionComponent
			x    float32
			y    float32
		}
		jurisdictions := make([]jurData, 0, 20)
		for jurQuery.Next() {
			jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))
			p := (*components.Position)(jurQuery.Get(posID))
			jurisdictions = append(jurisdictions, jurData{
				comp: jur,
				x:    p.X,
				y:    p.Y,
			})
		}

		for _, dp := range s.deadPos {
			for i := 0; i < len(jurisdictions); i++ {
				j := &jurisdictions[i]
				dx := dp.X - j.x
				dy := dp.Y - j.y
				if dx*dx+dy*dy <= j.comp.RadiusSquared {
					if j.comp.Trauma < 65535 {
						j.comp.Trauma++
					}
					break
				}
			}
		}
	}

	// Phase 25.1: Execute Succession
	if len(s.heirs) > 0 {
		// Create a separate array of initial dead IDs for cleanup
		var cleanupIDs []uint64
		for _, h := range s.heirs {
			cleanupIDs = append(cleanupIDs, h.DeadID)
		}

		// We need to iterate over all living NPCs to find suitable heirs
		npcID := ecs.ComponentID[components.NPC](world)
		npcQuery := world.Query(ecs.All(npcID, identityID, affilID, legacyID))

		type artifactAssignment struct {
			entity   ecs.Entity
			artifact *components.LegendComponent
		}
		var artifactAssignments []artifactAssignment

		for npcQuery.Next() {
			if false { // npcQuery.Has(needsID) {
				needs := (*components.Needs)(npcQuery.Get(needsID))
				if needs.Food <= 0 {
					continue
				}
			}
			affil := (*components.Affiliation)(npcQuery.Get(affilID))
			ident := (*components.Identity)(npcQuery.Get(identityID))
			legacy := (*components.Legacy)(npcQuery.Get(legacyID))

			for i := 0; i < len(s.heirs); i++ {
				h := s.heirs[i]
				if h.FamilyID == affil.FamilyID && h.DeadID != ident.ID && s.heirs[i].DeadID != 0 {
					// Found an heir! Transfer Legacy
					legacy.Prestige += h.Prestige
					legacy.InheritedDebt += h.InheritedDebt

					// Transfer Hooks
					for target, points := range h.OutgoingHooks {
						if target != ident.ID { // Don't hook yourself
							s.hookGraph.AddHook(ident.ID, target, points)
						}
					}
					for source, points := range h.IncomingHooks {
						if source != ident.ID {
							s.hookGraph.AddHook(source, ident.ID, points)
						}
					}

					// Phase 32.1: Transfer Artifact (Aura of Legitimacy)
					if h.Artifact != nil {
						artifactAssignments = append(artifactAssignments, artifactAssignment{
							entity:   npcQuery.Entity(),
							artifact: h.Artifact,
						})

						// We must also remove this artifact from the spawning pool so it isn't dropped on the map!
						for j := 0; j < len(s.itemsToSpawn); j++ {
							if s.itemsToSpawn[j].prestige == h.Prestige && s.itemsToSpawn[j].nameID == h.Artifact.NameID {
								// Remove this element fast by swapping with the last
								s.itemsToSpawn[j] = s.itemsToSpawn[len(s.itemsToSpawn)-1]
								s.itemsToSpawn = s.itemsToSpawn[:len(s.itemsToSpawn)-1]
								break
							}
						}
					}

					// Clear the dead ID so it's not inherited multiple times
					s.heirs[i].DeadID = 0
				}
			}
		}

		// Clean up the dead ID's hooks
		for _, deadID := range cleanupIDs {
			s.hookGraph.RemoveAllHooks(deadID)
		}

		// Apply deferred Artifact Inheritance Mid-tick Phase
		for _, assign := range artifactAssignments {
			if !world.Has(assign.entity, equipID) {
				world.Add(assign.entity, equipID)
			}
			equip := (*components.EquipmentComponent)(world.Get(assign.entity, equipID))
			equip.Weapon = *assign.artifact
			equip.Equipped = true
		}
	}

	// Remove dead entities
	for _, entity := range s.toRemove {
		world.RemoveEntity(entity)
	}

	// Spawn legend items
	if len(s.itemsToSpawn) > 0 {
		itemID := ecs.ComponentID[components.ItemEntity](world)
		legendID := ecs.ComponentID[components.LegendComponent](world)
		posID := ecs.ComponentID[components.Position](world)

		for _, itemData := range s.itemsToSpawn {
			newEnt := world.NewEntity(itemID, legendID, posID)

			legend := (*components.LegendComponent)(world.Get(newEnt, legendID))
			legend.NameID = itemData.nameID
			legend.Prestige = itemData.prestige
			// History starts empty
			legend.History = make([]uint32, 0)

			pos := (*components.Position)(world.Get(newEnt, posID))
			pos.X = itemData.posX
			pos.Y = itemData.posY
		}
	}
}
