package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 33: The Refugee Crisis
// RefugeeMigrationSystem evaluates displaced populations seeking a new village.

type RefugeeMigrationSystem struct {
	hooks *engine.SparseHookGraph
	toRemove []ecs.Entity
	toSpawnBandits []banditSpawnData
}

type banditSpawnData struct {
	x float32
	y float32
	targetID uint64
}

func NewRefugeeMigrationSystem(hooks *engine.SparseHookGraph) *RefugeeMigrationSystem {
	return &RefugeeMigrationSystem{
		hooks: hooks,
		toRemove: make([]ecs.Entity, 0, 100),
		toSpawnBandits: make([]banditSpawnData, 0, 100),
	}
}

func (s *RefugeeMigrationSystem) Update(world *ecs.World) {
	s.toRemove = s.toRemove[:0]
	s.toSpawnBandits = s.toSpawnBandits[:0]

	refClusterID := ecs.ComponentID[components.RefugeeCluster](world)
	refDataID := ecs.ComponentID[components.RefugeeData](world)
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)

	villageID := ecs.ComponentID[components.Village](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	idID := ecs.ComponentID[components.Identity](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	velID := ecs.ComponentID[components.Velocity](world)
	needsID := ecs.ComponentID[components.Needs](world)

	refFilter := ecs.All(refClusterID, refDataID, posID, pathID)
	refQuery := world.Query(refFilter)

	for refQuery.Next() {
		refEnt := refQuery.Entity()
		pos := (*components.Position)(refQuery.Get(posID))
		path := (*components.Path)(refQuery.Get(pathID))
		refData := (*components.RefugeeData)(refQuery.Get(refDataID))

		dx := pos.X - path.TargetX
		dy := pos.Y - path.TargetY
		distSq := dx*dx + dy*dy

		// Arrived at destination
		if distSq < 1.0 {
			// Find the village at these coordinates
			var targetVillage ecs.Entity
			var targetPop *components.PopulationComponent
			var targetStorage *components.StorageComponent
			var targetJur *components.JurisdictionComponent
			var targetID uint64

			vFilter := ecs.All(villageID, posID, popID, storageID)
			vQuery := world.Query(vFilter)
			for vQuery.Next() {
				vPos := (*components.Position)(vQuery.Get(posID))
				if vPos.X == path.TargetX && vPos.Y == path.TargetY {
					targetVillage = vQuery.Entity()
					targetPop = (*components.PopulationComponent)(vQuery.Get(popID))
					targetStorage = (*components.StorageComponent)(vQuery.Get(storageID))

					if world.Has(targetVillage, jurID) {
						targetJur = (*components.JurisdictionComponent)(world.Get(targetVillage, jurID))
					}
					if world.Has(targetVillage, idID) {
						targetID = (*components.Identity)(world.Get(targetVillage, idID)).ID
					}
					vQuery.Close()
					break
				}
			}

			if !world.Alive(targetVillage) {
				s.toRemove = append(s.toRemove, refEnt)
				continue
			}

			// Determine integration success
			rejected := false

			// Rejection due to starvation limit (Needs 1 Food per citizen roughly to integrate)
			if targetStorage.Food < refData.Count {
				rejected = true
			}

			// Rejection due to Xenophobia/Trauma
			if targetJur != nil && targetJur.Trauma >= 50 {
				if world.Has(targetVillage, cultureID) {
					vCulture := (*components.CultureComponent)(world.Get(targetVillage, cultureID))
					if vCulture.LanguageID != refData.Culture.LanguageID {
						rejected = true
					}
				} else {
					rejected = true
				}
			}

			if rejected {
				// The Systemic Emergence hook:
				// Refugees are rejected, turning them into bandits against the target.
				if s.hooks != nil && targetID != 0 {
					for _, citizen := range refData.Citizens {
						_ = citizen
						s.toSpawnBandits = append(s.toSpawnBandits, banditSpawnData{
							x: pos.X,
							y: pos.Y,
							targetID: targetID,
						})
					}
				}
			} else {
				// Accepted! Integrate population.
				targetPop.Count += refData.Count
				targetPop.Citizens = append(targetPop.Citizens, refData.Citizens...)

				// Cultural Drift via integration
				if world.Has(targetVillage, cultureID) {
					vCulture := (*components.CultureComponent)(world.Get(targetVillage, cultureID))
					if vCulture.LanguageID != refData.Culture.LanguageID {
						vCulture.ForeignInteractionTicks += 1000 // Boost drift
						vCulture.ForeignLanguageID = refData.Culture.LanguageID
					}
				}
			}
			s.toRemove = append(s.toRemove, refEnt)
		}
	}

	for _, e := range s.toRemove {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}

	for _, data := range s.toSpawnBandits {
		banditEnt := world.NewEntity()
		world.Add(banditEnt, npcID, posID, idID, jobID, velID, pathID, needsID)

		bPos := (*components.Position)(world.Get(banditEnt, posID))
		bPos.X = data.x
		bPos.Y = data.y

		bJob := (*components.JobComponent)(world.Get(banditEnt, jobID))
		bJob.JobID = components.JobBandit

		bID := (*components.Identity)(world.Get(banditEnt, idID))
		bID.ID = uint64(engine.GetRandomInt()) // Simplified ID generation

		bNeeds := (*components.Needs)(world.Get(banditEnt, needsID))
		bNeeds.Food = 0 // Desperate

		// Massive Grudge against destination
		s.hooks.AddHook(bID.ID, data.targetID, -50)
	}
}
