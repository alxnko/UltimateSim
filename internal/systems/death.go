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
	LoanContract  *components.LoanContractComponent // Phase 46: Generational Debt Engine
	Beliefs       []components.Belief // Phase 25.2: Ideological Succession Engine
}

type DeathSystem struct {
	filter       ecs.Filter
	toRemove     []ecs.Entity
	itemsToSpawn []itemSpawnData
	deadPos      []components.Position // Phase 20.3: Used for mapping trauma
	hookGraph    *engine.SparseHookGraph
	heirs        []heirData
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *DeathSystem) IsExpensive() bool {
	return true
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
	loanID := ecs.ComponentID[components.LoanContractComponent](world)
	beliefID := ecs.ComponentID[components.BeliefComponent](world)

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

				var loanContract *components.LoanContractComponent
				if query.Has(loanID) {
					loan := (*components.LoanContractComponent)(query.Get(loanID))
					loanCopy := *loan // Struct copy
					loanContract = &loanCopy
				}

				var beliefs []components.Belief
				if query.Has(beliefID) {
					belComp := (*components.BeliefComponent)(query.Get(beliefID))
					if len(belComp.Beliefs) > 0 {
						beliefs = make([]components.Belief, len(belComp.Beliefs))
						copy(beliefs, belComp.Beliefs)
					}
				}

				outgoing := s.hookGraph.GetAllHooks(ident.ID)
				incoming := s.hookGraph.GetAllIncomingHooks(ident.ID)

				// Only trigger succession if there's a reason (hooks or debt or prestige or artifact or active loan or beliefs)
				if len(outgoing) > 0 || len(incoming) > 0 || pres > 0 || debt > 0 || artifact != nil || loanContract != nil || len(beliefs) > 0 {
					s.heirs = append(s.heirs, heirData{
						DeadID:        ident.ID,
						FamilyID:      affil.FamilyID,
						Prestige:      pres,
						InheritedDebt: debt,
						OutgoingHooks: outgoing,
						IncomingHooks: incoming,
						Artifact:      artifact,
						LoanContract:  loanContract,
						Beliefs:       beliefs,
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

	// Phase 25.1: Execute Succession - Optimized to O(NPCs)
	if len(s.heirs) > 0 {
		// 1. Build a map of FamilyID -> Living NPC Entities
		familyMap := make(map[uint32][]ecs.Entity)

		npcID := ecs.ComponentID[components.NPC](world)
		npcQuery := world.Query(ecs.All(npcID, identityID, affilID, legacyID))

		// Map dying entities for fast lookup during family map build
		dyingMap := make(map[ecs.Entity]bool)
		for _, e := range s.toRemove {
			dyingMap[e] = true
		}

		for npcQuery.Next() {
			ent := npcQuery.Entity()
			if dyingMap[ent] {
				continue // Heirs must be alive
			}
			affil := (*components.Affiliation)(npcQuery.Get(affilID))
			familyMap[affil.FamilyID] = append(familyMap[affil.FamilyID], ent)
		}

		// 2. Process succession using the map
		for _, h := range s.heirs {
			if heirs, exists := familyMap[h.FamilyID]; exists && len(heirs) > 0 {
				// For now, pick the first available heir in the family
				heirEnt := heirs[0]

				legacy := (*components.Legacy)(world.Get(heirEnt, legacyID))
				ident := (*components.Identity)(world.Get(heirEnt, identityID))

				// Transfer Legacy
				legacy.Prestige += h.Prestige
				legacy.InheritedDebt += h.InheritedDebt

				// Transfer Hooks
				for target, points := range h.OutgoingHooks {
					if target != ident.ID {
						s.hookGraph.AddHook(ident.ID, target, points)
					}
				}
				for source, points := range h.IncomingHooks {
					if source != ident.ID {
						s.hookGraph.AddHook(source, ident.ID, points)
					}
				}

				// Phase 32.1: Transfer Artifact
				if h.Artifact != nil {
					if !world.Has(heirEnt, equipID) {
						world.Add(heirEnt, equipID)
					}
					equip := (*components.EquipmentComponent)(world.Get(heirEnt, equipID))
					equip.Weapon = *h.Artifact
					equip.Equipped = true

					// Remove from spawning items pool
					for j := 0; j < len(s.itemsToSpawn); j++ {
						if s.itemsToSpawn[j].prestige == h.Prestige && s.itemsToSpawn[j].nameID == h.Artifact.NameID {
							s.itemsToSpawn[j] = s.itemsToSpawn[len(s.itemsToSpawn)-1]
							s.itemsToSpawn = s.itemsToSpawn[:len(s.itemsToSpawn)-1]
							break
						}
					}
				}

				// Phase 46: Generational Debt Engine - Transfer Loan Contract directly
				if h.LoanContract != nil {
					if !world.Has(heirEnt, loanID) {
						world.Add(heirEnt, loanID)
					}
					loan := (*components.LoanContractComponent)(world.Get(heirEnt, loanID))
					*loan = *h.LoanContract
				}

				// Evolution: Phase 25.2 - The Ideological Succession Engine
				if len(h.Beliefs) > 0 {
					if !world.Has(heirEnt, beliefID) {
						world.Add(heirEnt, beliefID)
					}
					belComp := (*components.BeliefComponent)(world.Get(heirEnt, beliefID))

					for _, b := range h.Beliefs {
						decayedWeight := b.Weight / 2
						if decayedWeight > 0 {
							found := false
							for i := range belComp.Beliefs {
								if belComp.Beliefs[i].BeliefID == b.BeliefID {
									belComp.Beliefs[i].Weight += decayedWeight
									found = true
									break
								}
							}
							if !found {
								belComp.Beliefs = append(belComp.Beliefs, components.Belief{
									BeliefID: b.BeliefID,
									Weight:   decayedWeight,
								})
							}
						}
					}
				}
			}
		}

		// 3. Clean up the dead ID's hooks
		for _, h := range s.heirs {
			s.hookGraph.RemoveAllHooks(h.DeadID)
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
