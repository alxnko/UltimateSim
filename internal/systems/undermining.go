package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 36: The Sapper & Undermining Engine
// Bridges Geography, Warfare, and Construction by having JobMiner NPCs
// dynamically spawn TunnelComponents under StructureComponents during a siege.
// When the tunnel progresses, it mathematically degrades the StructureComponent's Integrity.
// Once Integrity hits 0, the Structure is destroyed, opening breaches for infantry.

type targetStructureData struct {
	Entity    ecs.Entity
	ID        uint64
	X         float32
	Y         float32
	Integrity float32
}

type structuralChange struct {
	Type         uint8
	TargetID     uint64
	TargetEntity ecs.Entity
	MinerEntity  ecs.Entity
	MinerX       float32
	MinerY       float32
}

const (
	changeSpawnTunnel   = 1
	changeDestroyStruct = 2
)

type UnderminingSystem struct {
	tickCounter uint64

	warID    ecs.ID
	jobID    ecs.ID
	posID    ecs.ID
	idID     ecs.ID
	structID ecs.ID
	tunnelID ecs.ID

	changes []structuralChange
}

func NewUnderminingSystem(world *ecs.World) *UnderminingSystem {
	return &UnderminingSystem{
		tickCounter: 0,
		warID:       ecs.ComponentID[components.WarTrackerComponent](world),
		jobID:       ecs.ComponentID[components.JobComponent](world),
		posID:       ecs.ComponentID[components.Position](world),
		idID:        ecs.ComponentID[components.Identity](world),
		structID:    ecs.ComponentID[components.StructureComponent](world),
		tunnelID:    ecs.ComponentID[components.TunnelComponent](world),
		changes:     make([]structuralChange, 0, 100),
	}
}

func (s *UnderminingSystem) Update(world *ecs.World) {
	s.tickCounter++

	if s.tickCounter%10 != 0 {
		return // Evaluate every 10 ticks for performance
	}

	s.changes = s.changes[:0]

	// 1. Check if there are any active wars
	warQuery := world.Query(ecs.All(s.warID))
	hasActiveWars := false
	for warQuery.Next() {
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warID))
		if war.Active {
			hasActiveWars = true
			// DO NOT BREAK to ensure proper full iteration so it auto-closes, or manually close before break
		}
	}

	if !hasActiveWars {
		return
	}

	// 2. Pre-cache potential structure targets for DOD performance
	structQuery := world.Query(ecs.All(s.structID, s.posID, s.idID))
	structures := make([]targetStructureData, 0, 500)
	for structQuery.Next() {
		pos := (*components.Position)(structQuery.Get(s.posID))
		id := (*components.Identity)(structQuery.Get(s.idID))
		st := (*components.StructureComponent)(structQuery.Get(s.structID))

		structures = append(structures, targetStructureData{
			Entity:    structQuery.Entity(),
			ID:        id.ID,
			X:         pos.X,
			Y:         pos.Y,
			Integrity: st.Integrity,
		})
	}

	if len(structures) == 0 {
		return
	}

	// 3. Pre-cache active tunnels to avoid duplicate spawns.
	// Cache values, not component pointers — GC corruption class, see
	// banditry.go. Entity handles are stored and the TunnelComponent is
	// re-fetched via world.Get at use time; tunnels deferred for spawn this
	// tick are tracked separately (their placeholder never had an entity).
	tunnelQuery := world.Query(ecs.All(s.tunnelID))
	tunnels := make(map[uint64]ecs.Entity)
	pendingTunnels := make(map[uint64]bool)
	for tunnelQuery.Next() {
		t := (*components.TunnelComponent)(tunnelQuery.Get(s.tunnelID))
		tunnels[t.TargetID] = tunnelQuery.Entity()
	}

	// 4. Iterate over Miners and execute undermining
	minerQuery := world.Query(ecs.All(s.jobID, s.posID))

	// Track structures to remove this tick to avoid multiple miners triggering destroy
	structuresToDestroy := make(map[uint64]bool)

	for minerQuery.Next() {
		job := (*components.JobComponent)(minerQuery.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		minerPos := (*components.Position)(minerQuery.Get(s.posID))

		var bestTarget *targetStructureData
		var bestDist float32 = 999999.0

		for i := 0; i < len(structures); i++ {
			st := &structures[i]

			if structuresToDestroy[st.ID] {
				continue
			}

			dx := minerPos.X - st.X
			dy := minerPos.Y - st.Y
			distSq := dx*dx + dy*dy

			if distSq < bestDist {
				bestDist = distSq
				bestTarget = st
			}
		}

		// If a structure is nearby (within 5 tiles)
		if bestTarget != nil && bestDist <= 25.0 {
			tunnelEnt, exists := tunnels[bestTarget.ID]
			if exists || pendingTunnels[bestTarget.ID] {
				// Tunnel exists (or is pending spawn this tick), progress it.
				// A pending tunnel has no entity yet; its progress write was
				// always discarded (the spawned tunnel starts at 0), so only
				// real tunnels are advanced.
				if exists && world.Alive(tunnelEnt) {
					t := (*components.TunnelComponent)(world.Get(tunnelEnt, s.tunnelID))
					t.Progress += 5.0
				}

				// Deduct structural integrity structurally or directly (if we have access)
				// Since we pre-cached structures, we need to modify the actual component.
				// We can just fetch it again safely because query on miners doesn't conflict directly
				// with fetching a specific entity's component, provided it's alive.
				if world.Alive(bestTarget.Entity) && world.Has(bestTarget.Entity, s.structID) {
					stComp := (*components.StructureComponent)(world.Get(bestTarget.Entity, s.structID))
					stComp.Integrity -= 5.0

					if stComp.Integrity <= 0 {
						structuresToDestroy[bestTarget.ID] = true
						s.changes = append(s.changes, structuralChange{
							Type:         changeDestroyStruct,
							TargetEntity: bestTarget.Entity,
						})
					}
				}
			} else {
				// Defer tunnel creation
				s.changes = append(s.changes, structuralChange{
					Type:     changeSpawnTunnel,
					TargetID: bestTarget.ID,
					MinerX:   minerPos.X,
					MinerY:   minerPos.Y,
				})
				// Optimistically mark as pending to prevent multi-spawns in same tick
				pendingTunnels[bestTarget.ID] = true
			}
		}
	}

	// 5. Apply deferred structural changes
	for _, change := range s.changes {
		if change.Type == changeSpawnTunnel {
			e := world.NewEntity()
			world.Add(e, s.posID, s.tunnelID)

			pos := (*components.Position)(world.Get(e, s.posID))
			pos.X = change.MinerX
			pos.Y = change.MinerY

			tunnel := (*components.TunnelComponent)(world.Get(e, s.tunnelID))
			tunnel.TargetID = change.TargetID
			tunnel.Progress = 0.0
		} else if change.Type == changeDestroyStruct {
			if world.Alive(change.TargetEntity) {
				world.RemoveEntity(change.TargetEntity)
			}
		}
	}
}
