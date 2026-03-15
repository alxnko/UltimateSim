package systems

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 18.2: Detection & The Guard System
// JusticeSystem evaluates MemoryEvents and assigns CrimeMarkers to entities committing illegal actions within a Jurisdiction bounds.
// It also directs Guards towards entities tagged with a CrimeMarker to enforce punishments.

type adminJurisdictionData struct {
	Entity   ecs.Entity
	CityID   uint32
	X        float32
	Y        float32
	Radius   float32
	Laws     uint32
	Treasury *components.TreasuryComponent // Pre-cached for Phase 18.3 Fines
}

type JusticeSystem struct {
	jurisdictions []adminJurisdictionData
	guardFilter   ecs.Filter
	crimeFilter   ecs.Filter
	targetMapping map[uint64]bool // To prevent multiple guards targeting the same criminal instantly
	hooks         *engine.SparseHookGraph
}

func NewJusticeSystem(world *ecs.World, hooks *engine.SparseHookGraph) *JusticeSystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)
	velID := ecs.ComponentID[components.Velocity](world)

	gMask := ecs.All(jobID, posID, pathID, velID)

	crimeID := ecs.ComponentID[components.CrimeMarker](world)
	cMask := ecs.All(crimeID, posID)

	return &JusticeSystem{
		jurisdictions: make([]adminJurisdictionData, 0, 100),
		guardFilter:   &gMask,
		crimeFilter:   &cMask,
		targetMapping: make(map[uint64]bool),
		hooks:         hooks,
	}
}

func (s *JusticeSystem) Update(world *ecs.World) {
	// Step 1: Pre-cache all Jurisdiction boundaries to dodge nested queries
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	jurQuery := world.Query(ecs.All(jurID, posID, affID))
	s.jurisdictions = s.jurisdictions[:0]

	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))
		pos := (*components.Position)(jurQuery.Get(posID))
		aff := (*components.Affiliation)(jurQuery.Get(affID))

		var treasury *components.TreasuryComponent
		if world.Has(jurQuery.Entity(), treasuryID) {
			treasury = (*components.TreasuryComponent)(jurQuery.Get(treasuryID))
		}

		s.jurisdictions = append(s.jurisdictions, adminJurisdictionData{
			Entity:   jurQuery.Entity(),
			CityID:   aff.CityID,
			X:        pos.X,
			Y:        pos.Y,
			Radius:   jur.RadiusSquared,
			Laws:     jur.IllegalActionIDs,
			Treasury: treasury,
		})
	}

	if len(s.jurisdictions) == 0 {
		return // No laws to enforce
	}

	// Step 2: Detection - iterate over all entities with a Memory buffer that could be committing crimes
	memID := ecs.ComponentID[components.Memory](world)
	idID := ecs.ComponentID[components.Identity](world)
	crimeID := ecs.ComponentID[components.CrimeMarker](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
		contraID := ecs.ComponentID[components.ContrabandComponent](world)

	npcQuery := world.Query(ecs.All(memID, posID, affID))

	// Temporarily gather entities to tag to avoid modifying arche structure mid-query
	var criminalsToTag []ecs.Entity

	for npcQuery.Next() {
		entity := npcQuery.Entity()
		if world.Has(entity, crimeID) {
			continue // Already marked
		}

		pos := (*components.Position)(npcQuery.Get(posID))
		mem := (*components.Memory)(npcQuery.Get(memID))

		// Find which jurisdiction they are in
		var activeJur *adminJurisdictionData
		for i := 0; i < len(s.jurisdictions); i++ {
			j := &s.jurisdictions[i]
			dx := pos.X - j.X
			dy := pos.Y - j.Y
			distSq := (dx * dx) + (dy * dy)
			if distSq <= j.Radius {
				activeJur = j
				break // We assume first hit jurisdiction bounds applies
			}
		}

		if activeJur != nil {
			isCriminal := false

			// Evaluate recent memory events for illegal actions
			// Realistically we should evaluate against a tick window, but for DOD speed we check the buffer
			for i := 0; i < len(mem.Events); i++ {
				ev := &mem.Events[i]
				if ev.InteractionType > 0 { // Skip zeroed empty slots
					bitmaskCheck := uint32(1 << ev.InteractionType)
					if (activeJur.Laws & bitmaskCheck) != 0 {
						isCriminal = true
						break
					}
				}
			}

			// Phase 18.1: Contraband Check
			if !isCriminal && world.Has(entity, storageID) && world.Has(activeJur.Entity, contraID) {
				store := (*components.StorageComponent)(npcQuery.Get(storageID))
				contra := (*components.ContrabandComponent)(world.Get(activeJur.Entity, contraID))

				if contra.Contraband > 0 {
					// Check bits
					if store.Wood > 0 && (contra.Contraband&(1<<components.ItemWood) != 0) { isCriminal = true }
					if store.Stone > 0 && (contra.Contraband&(1<<components.ItemStone) != 0) { isCriminal = true }
					if store.Iron > 0 && (contra.Contraband&(1<<components.ItemIron) != 0) { isCriminal = true }
				}
			}

			if isCriminal {
				criminalsToTag = append(criminalsToTag, entity)
			}
		}
	}

	for _, e := range criminalsToTag {
		if !world.Has(e, crimeID) { // Double check alive/has
			world.Add(e, crimeID)
			cm := (*components.CrimeMarker)(world.Get(e, crimeID))
			cm.CrimeLevel = 1
			cm.Bounty = 100 // Abstract wealth bounty
		}
	}

	// Clear target map for this frame
	for k := range s.targetMapping {
		delete(s.targetMapping, k)
	}

	// Step 3: Enforcement - evaluate Guards vs Criminals (O(G*C))
	crimeQuery := world.Query(s.crimeFilter)

	type cData struct {
		Entity ecs.Entity
		X      float32
		Y      float32
		ID     uint64
	}
	criminals := make([]cData, 0, 50)
	for crimeQuery.Next() {
		pos := (*components.Position)(crimeQuery.Get(posID))
		var identID uint64
		if crimeQuery.Has(idID) {
			ident := (*components.Identity)(crimeQuery.Get(idID))
			identID = ident.ID
		}
		criminals = append(criminals, cData{
			Entity: crimeQuery.Entity(),
			X:      pos.X,
			Y:      pos.Y,
			ID:     identID,
		})
	}

	if len(criminals) > 0 {
		// ecs-arche: we cannot request componentIDs while a query is active if those IDs are new.
		// Actually, query locks the world. We must fetch IDs before Query or rely on earlier IDs.
		jobID := ecs.ComponentID[components.JobComponent](world)
		pathID := ecs.ComponentID[components.Path](world)
		velID := ecs.ComponentID[components.Velocity](world)
		needsID := ecs.ComponentID[components.Needs](world)
		jurID := ecs.ComponentID[components.JurisdictionComponent](world)
		identID := ecs.ComponentID[components.Identity](world)
		secID := ecs.ComponentID[components.SecretComponent](world)

		guardQuery := world.Query(s.guardFilter)

		var punishedEntities []ecs.Entity

		for guardQuery.Next() {
			job := (*components.JobComponent)(guardQuery.Get(jobID))
			if job.JobID != components.JobGuard {
				continue
			}

			gPos := (*components.Position)(guardQuery.Get(posID))
			path := (*components.Path)(guardQuery.Get(pathID))

			var gAff *components.Affiliation
			if guardQuery.Has(affID) {
				gAff = (*components.Affiliation)(guardQuery.Get(affID))
			}

			var gIdent *components.Identity
			if guardQuery.Has(identID) {
				gIdent = (*components.Identity)(guardQuery.Get(identID))
			}

			// Find closest non-targeted criminal
			var best *cData
			var bestDist float32 = 9999999.0

			for i := 0; i < len(criminals); i++ {
				c := &criminals[i]

				// Optional: Only target if they are within the same jurisdiction (omitted for speed unless req)

				// Phase 18.2: Target tracking
				// Do not target if another Guard is already targeting them
				if s.targetMapping[c.ID] {
					continue
				}

				dx := gPos.X - c.X
				dy := gPos.Y - c.Y
				distSq := (dx * dx) + (dy * dy)

				// Phase 18.3: Sentencing & Prisons
				// If Guard is extremely close to Criminal (e.g. adjacent tile) -> Execute Punishment
				if distSq < 2.0 && world.Alive(c.Entity) {
					bribed := false

					// Check for Phase 22.1 Bribery / Corruption Engine
					if world.Has(c.Entity, needsID) {
						cNeeds := (*components.Needs)(world.Get(c.Entity, needsID))
						crimeMarker := (*components.CrimeMarker)(world.Get(c.Entity, crimeID))
						bribeAmount := float32(crimeMarker.Bounty) * 2.0

						if cNeeds.Wealth >= bribeAmount {
							// Execute bribe
							cNeeds.Wealth -= bribeAmount
							bribed = true

							// Increment Corruption in Guard's active Jurisdiction
							if gAff != nil {
								for jIdx := 0; jIdx < len(s.jurisdictions); jIdx++ {
									if s.jurisdictions[jIdx].CityID == gAff.CityID {
										jurEnt := s.jurisdictions[jIdx].Entity
										if world.Alive(jurEnt) && world.Has(jurEnt, jurID) {
											jurComp := (*components.JurisdictionComponent)(world.Get(jurEnt, jurID))
											jurComp.Corruption += 1
										}
										break
									}
								}
							}

							// Phase 30: Blackmail Engine
							if gIdent != nil && s.hooks != nil {
								s.hooks.AddHook(c.ID, gIdent.ID, 50)

								// Generate Rumor
								if world.Has(c.Entity, secID) {
									sec := (*components.SecretComponent)(world.Get(c.Entity, secID))
									registry := engine.GetSecretRegistry()
									if registry != nil {
										rumorText := fmt.Sprintf("guard_%d_corrupted", gIdent.ID)
										secretID := registry.RegisterSecret(rumorText)
										sec.Secrets = append(sec.Secrets, components.Secret{
											OriginID: c.ID,
											SecretID: secretID,
											Virality: 255,
										})
									}
								}
							}
						}
					}

					if !bribed {
						// Apply standard Punishment (Banishment & Fines)
						var collectedFine float32 = 0.0
						if world.Has(c.Entity, needsID) {
							cNeeds := (*components.Needs)(world.Get(c.Entity, needsID))
							crimeMarker := (*components.CrimeMarker)(world.Get(c.Entity, crimeID))

							fine := float32(crimeMarker.Bounty)
							if cNeeds.Wealth >= fine {
								cNeeds.Wealth -= fine
								collectedFine = fine
							} else {
								collectedFine = cNeeds.Wealth
								cNeeds.Wealth = 0
							}
						}

						// Phase 18.3: Sentencing & Prisons (Fines transfer wealth to the enforcing CityID)
						if collectedFine > 0.0 && gAff != nil {
							// Find the pre-cached TreasuryComponent of the Guard's City
							for jIdx := 0; jIdx < len(s.jurisdictions); jIdx++ {
								if s.jurisdictions[jIdx].CityID == gAff.CityID {
									if s.jurisdictions[jIdx].Treasury != nil {
										s.jurisdictions[jIdx].Treasury.Wealth += collectedFine
									}
									break
								}
							}
						}

						// Banishment
						if world.Has(c.Entity, affID) && gAff != nil {
							cAff := (*components.Affiliation)(world.Get(c.Entity, affID))
							// Remove CityID if it belongs to the enforcing Guard's city
							if cAff.CityID == gAff.CityID {
								cAff.CityID = 0
							}
						}

						// Force fleeing behavior
						if world.Has(c.Entity, velID) {
							cVel := (*components.Velocity)(world.Get(c.Entity, velID))
							// Send flying outward
							if dx == 0 && dy == 0 { dx = 1 } // prevent div by zero
							cVel.X = -dx * 2.0
							cVel.Y = -dy * 2.0
						}

						// Phase 30: Carceral Resentment
						if gIdent != nil && s.hooks != nil {
							s.hooks.AddHook(c.ID, gIdent.ID, -50)
						}
					}

					// Queue CrimeMarker for removal (cannot remove component while query is active)
					punishedEntities = append(punishedEntities, c.Entity)
					best = nil // Punished, no need to target anymore
					break // Done with this guard
				}

				if distSq < bestDist {
					bestDist = distSq
					best = c
				}
			}

			if best != nil {
				// Target the criminal's position (Phase 18.2)
				if path.TargetX != best.X || path.TargetY != best.Y {
					path.TargetX = best.X
					path.TargetY = best.Y
					path.HasPath = false // Trigger repathing in MovementSystem/WanderSystem
				}
				s.targetMapping[best.ID] = true
			}
		}

		// Clean up markers outside query
		for _, e := range punishedEntities {
			if world.Alive(e) && world.Has(e, crimeID) {
				world.Remove(e, crimeID)
			}
		}
	}
}
