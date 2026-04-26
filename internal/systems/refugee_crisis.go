package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 63: The Refugee Crisis Engine
// Bridges Entropy, Logistics, and Culture by routing NPCs with extreme desperation
// to prosperous cities, structurally triggering Xenophobia interactions.

type RefugeeSystem struct {
	world       *ecs.World
	pathQueue   *engine.PathRequestQueue
	tickCounter uint64

	pendingReqs map[uint64]ecs.Entity
}

// IsExpensive returns true as it does HPA* requests occasionally.
func (s *RefugeeSystem) IsExpensive() bool {
	return true
}

func NewRefugeeSystem(world *ecs.World, pathQueue *engine.PathRequestQueue) *RefugeeSystem {
	return &RefugeeSystem{
		world:       world,
		pathQueue:   pathQueue,
		pendingReqs: make(map[uint64]ecs.Entity),
	}
}

func (s *RefugeeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// 1. Drain incoming path results from the asynchronous queue
DrainLoop:
	for {
		select {
		case res := <-s.pathQueue.GetResultsChannel():
			// Re-fetch entity from pending map
			if entity, exists := s.pendingReqs[res.EntityID]; exists {
				// Ensure entity still exists in world
				if s.world.Alive(entity) {
					pathID := ecs.ComponentID[components.Path](s.world)
					if s.world.Has(entity, pathID) {
						path := (*components.Path)(s.world.Get(entity, pathID))
						if res.Success && len(res.Path) > 0 {
							// We successfully found a path
							path.Nodes = make([]components.Position, len(res.Path))
							for i, node := range res.Path {
								path.Nodes[i] = components.Position{X: node.X, Y: node.Y}
							}
						} else {
							// Path failed, allow retry
							path.HasPath = false
						}
					}
				}
				delete(s.pendingReqs, res.EntityID)
			}
		default:
			// No pending results
			break DrainLoop
		}
	}

	if s.tickCounter%10 != 0 {
		return
	}

	// Components
	vID := ecs.ComponentID[components.Village](s.world)
	tID := ecs.ComponentID[components.TreasuryComponent](s.world)
	aID := ecs.ComponentID[components.Affiliation](s.world)
	pID := ecs.ComponentID[components.Position](s.world)

	dID := ecs.ComponentID[components.DesperationComponent](s.world)
	identID := ecs.ComponentID[components.Identity](s.world)
	pathID := ecs.ComponentID[components.Path](s.world)
	asylumID := ecs.ComponentID[components.AsylumSeekerComponent](s.world)

	type wealthyCityData struct {
		cityID uint32
		x      float32
		y      float32
	}
	var prosperousCities []wealthyCityData

	// Pre-cache wealthy cities
	vFilter := filter.All(vID, tID, aID, pID)
	vQuery := s.world.Query(&vFilter)
	for vQuery.Next() {
		treas := (*components.TreasuryComponent)(vQuery.Get(tID))
		if treas.Wealth > 100.0 {
			aff := (*components.Affiliation)(vQuery.Get(aID))
			pos := (*components.Position)(vQuery.Get(pID))
			prosperousCities = append(prosperousCities, wealthyCityData{
				cityID: aff.CityID,
				x:      pos.X,
				y:      pos.Y,
			})
		}
	}

	// Process Desperate NPCs
	var toAddAsylum []ecs.Entity
	var targetCityIDs []uint32
	var reqs []engine.PathRequest

	despFilter := filter.All(dID, identID, pID, aID, pathID).Without(asylumID)
	dQuery := s.world.Query(&despFilter)
	for dQuery.Next() {
		desp := (*components.DesperationComponent)(dQuery.Get(dID))
		if desp.Level >= 80 {
			pos := (*components.Position)(dQuery.Get(pID))
			ident := (*components.Identity)(dQuery.Get(identID))

			// Find closest prosperous city
			var bestDist float32 = 9999999.0
			var bestCityID uint32 = 0
			var bestX, bestY float32

			for _, city := range prosperousCities {
				dx := city.x - pos.X
				dy := city.y - pos.Y
				distSq := dx*dx + dy*dy
				if distSq < bestDist {
					bestDist = distSq
					bestCityID = city.cityID
					bestX = city.x
					bestY = city.y
				}
			}

			if bestCityID != 0 {
				toAddAsylum = append(toAddAsylum, dQuery.Entity())
				targetCityIDs = append(targetCityIDs, bestCityID)

				reqs = append(reqs, engine.PathRequest{
					EntityID: ident.ID,
					StartX:   pos.X,
					StartY:   pos.Y,
					TargetX:  bestX,
					TargetY:  bestY,
				})
			}
		}
	}

	// Process NPCs already seeking asylum
	var toAssimilate []ecs.Entity
	var assimilateCityIDs []uint32

	asylumFilter := filter.All(asylumID, pID, aID, dID)
	aQuery := s.world.Query(&asylumFilter)
	for aQuery.Next() {
		asylum := (*components.AsylumSeekerComponent)(aQuery.Get(asylumID))
		pos := (*components.Position)(aQuery.Get(pID))

		// Find city coords from our cache
		var cx, cy float32
		found := false
		for _, city := range prosperousCities {
			if city.cityID == asylum.TargetCityID {
				cx = city.x
				cy = city.y
				found = true
				break
			}
		}

		if found {
			dx := cx - pos.X
			dy := cy - pos.Y
			distSq := dx*dx + dy*dy

			if distSq < 4.0 {
				toAssimilate = append(toAssimilate, aQuery.Entity())
				assimilateCityIDs = append(assimilateCityIDs, asylum.TargetCityID)
			}
		}
	}

	// Structural modifications outside query loops
	for i, entity := range toAddAsylum {
		if s.world.Alive(entity) {
			s.world.Add(entity, asylumID)
			asylum := (*components.AsylumSeekerComponent)(s.world.Get(entity, asylumID))
			asylum.TargetCityID = targetCityIDs[i]

			path := (*components.Path)(s.world.Get(entity, pathID))
			path.HasPath = true
			path.TargetX = reqs[i].TargetX
			path.TargetY = reqs[i].TargetY

			s.pathQueue.Enqueue(reqs[i])

			ident := (*components.Identity)(s.world.Get(entity, identID))
			s.pendingReqs[ident.ID] = entity
		}
	}

	for i, entity := range toAssimilate {
		if s.world.Alive(entity) {
			aff := (*components.Affiliation)(s.world.Get(entity, aID))
			aff.CityID = assimilateCityIDs[i]

			desp := (*components.DesperationComponent)(s.world.Get(entity, dID))
			desp.Level = 0

			s.world.Remove(entity, asylumID)
		}
	}
}
