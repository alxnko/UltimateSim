package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 47 - The Mercenary Engine
// Bridging the Economy (Wealth) directly with Justice and Blood Feuds (Violence).
// Wealthy NPCs with deep negative hooks (-50) against a target use their wealth to
// hire desperate/unemployed NPCs in close physical proximity to execute the hit.
// The mercenary absorbs the negative hook natively, triggering BloodFeudSystem.

type wealthyClientData struct {
	Entity ecs.Entity
	ID     uint64
	X      float32
	Y      float32
	Wealth float32
	Needs  *components.Needs
}

type desperateMercData struct {
	Entity ecs.Entity
	ID     uint64
	X      float32
	Y      float32
	Job    *components.JobComponent
}

type MercenarySystem struct {
	hooks       *engine.SparseHookGraph
	tickCounter uint64

	// Component IDs mapped once during NewMercenarySystem
	npcID      ecs.ID
	identID    ecs.ID
	needsID    ecs.ID
	jobID      ecs.ID
	posID      ecs.ID
	despID     ecs.ID
	mercCompID ecs.ID
}

func NewMercenarySystem(world *ecs.World, hooks *engine.SparseHookGraph) *MercenarySystem {
	return &MercenarySystem{
		hooks:       hooks,
		tickCounter: 0,
		npcID:       ecs.ComponentID[components.NPC](world),
		identID:     ecs.ComponentID[components.Identity](world),
		needsID:     ecs.ComponentID[components.Needs](world),
		jobID:       ecs.ComponentID[components.JobComponent](world),
		posID:       ecs.ComponentID[components.Position](world),
		despID:      ecs.ComponentID[components.DesperationComponent](world),
		mercCompID:  ecs.ComponentID[components.MercenaryContractComponent](world),
	}
}

func (s *MercenarySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle execution to preserve 60 FPS
	if s.tickCounter%100 != 0 {
		return
	}

	if s.hooks == nil {
		return
	}

	// 1. Identify Wealthy Clients (Need > 500 Wealth)
	// We extract to flat arrays to avoid O(N^2) nested arche-go queries
	clientQuery := world.Query(filter.All(s.npcID, s.identID, s.needsID, s.posID))
	var clients []wealthyClientData

	for clientQuery.Next() {
		needs := (*components.Needs)(clientQuery.Get(s.needsID))
		if needs.Wealth > 500.0 {
			ident := (*components.Identity)(clientQuery.Get(s.identID))
			pos := (*components.Position)(clientQuery.Get(s.posID))

			clients = append(clients, wealthyClientData{
				Entity: clientQuery.Entity(),
				ID:     ident.ID,
				X:      pos.X,
				Y:      pos.Y,
				Wealth: needs.Wealth,
				Needs:  needs,
			})
		}
	}
	// Need to manually close early termination queries (if we did break out of the loop)
	// Not applicable here since we read the entire query

	if len(clients) == 0 {
		return
	}

	// 2. Identify Desperate/Unemployed NPCs who can act as Mercenaries
	mercQuery := world.Query(filter.All(s.npcID, s.identID, s.jobID, s.despID, s.posID))
	var potentialMercs []desperateMercData

	for mercQuery.Next() {
		job := (*components.JobComponent)(mercQuery.Get(s.jobID))
		// Only hire unemployed
		if job.JobID == components.JobNone {
			desp := (*components.DesperationComponent)(mercQuery.Get(s.despID))
			// Must be sufficiently desperate to take a hit contract
			if desp.Level >= 30 {
				ident := (*components.Identity)(mercQuery.Get(s.identID))
				pos := (*components.Position)(mercQuery.Get(s.posID))

				potentialMercs = append(potentialMercs, desperateMercData{
					Entity: mercQuery.Entity(),
					ID:     ident.ID,
					X:      pos.X,
					Y:      pos.Y,
					Job:    job,
				})
			}
		}
	}

	if len(potentialMercs) == 0 {
		return
	}

	// Track assigned mercs to prevent double hiring in the same tick
	hiredMercs := make(map[uint64]bool)
	var newMercenaryStructs []struct {
		Entity ecs.Entity
		Target uint64
		Bribe  float32
	}

	// 3. Match Clients to Mercenaries based on severe negative hooks
	for _, client := range clients {
		// A client will only hire a hitman if they have wealth to spare.
		// Bribe cost is 200 wealth.
		if client.Needs.Wealth < 200.0 {
			continue
		}

		outgoingHooks := s.hooks.GetAllHooks(client.ID)
		var targetID uint64 = 0
		var worstHook int = -49 // Must be <= -50

		// Find the target the client hates the most (deterministic evaluation)
		for destID, hookVal := range outgoingHooks {
			if hookVal < worstHook {
				worstHook = hookVal
				targetID = destID
			} else if hookVal == worstHook && destID > targetID {
				// Tie-breaker using targetID to guarantee determinism regardless of map iteration order
				targetID = destID
			}
		}

		if targetID == 0 {
			continue // No one they hate enough to kill
		}

		// Find the closest desperate merc
		var bestMerc *desperateMercData
		var minSq float32 = 10000.0 // Spatial limit (100 tiles max distance)

		for i := 0; i < len(potentialMercs); i++ {
			merc := &potentialMercs[i]
			if hiredMercs[merc.ID] {
				continue
			}

			dx := client.X - merc.X
			dy := client.Y - merc.Y
			distSq := dx*dx + dy*dy

			if distSq < minSq {
				minSq = distSq
				bestMerc = merc
			}
		}

		if bestMerc != nil {
			// Contract Executed. Transfer Wealth.
			bribe := float32(200.0)
			client.Needs.Wealth -= bribe

			// Update the merc's job natively
			bestMerc.Job.JobID = components.JobMercenary
			bestMerc.Job.EmployerID = client.ID

			hiredMercs[bestMerc.ID] = true

			newMercenaryStructs = append(newMercenaryStructs, struct {
				Entity ecs.Entity
				Target uint64
				Bribe  float32
			}{
				Entity: bestMerc.Entity,
				Target: targetID,
				Bribe:  bribe,
			})

			// Most Importantly: The Mercenary inherits the negative hook against the target.
			// This structurally triggers the BloodFeudSystem (Phase 23) to execute the hit natively.
			s.hooks.AddHook(bestMerc.ID, targetID, -100)
		}
	}

	// 4. Structural additions outside the main arche-go iteration lock
	for _, mercData := range newMercenaryStructs {
		if world.Alive(mercData.Entity) {
			world.Add(mercData.Entity, s.mercCompID)
			contract := (*components.MercenaryContractComponent)(world.Get(mercData.Entity, s.mercCompID))
			contract.TargetID = mercData.Target
			contract.WealthBribe = mercData.Bribe

			// Give the merc the money
			if world.Has(mercData.Entity, s.needsID) {
				mercNeeds := (*components.Needs)(world.Get(mercData.Entity, s.needsID))
				mercNeeds.Wealth += mercData.Bribe
			}
		}
	}
}
