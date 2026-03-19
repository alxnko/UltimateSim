package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.1: Maritime Labor Market
// MaritimeLaborSystem iterates over ships and hires unemployed or desperate NPCs to work as JobSailor.
// It also transfers wealth from the Ship's Treasury to the NPC's Needs.Wealth as wages.
// If the ship cannot pay, the crew quits.

type shipLaborData struct {
	entity      ecs.Entity
	identID     uint64
	shipComp    *components.ShipComponent
	treasury    *components.TreasuryComponent
	pos         *components.Position
}

type MaritimeLaborSystem struct {
	tickStamp uint64
	ships     []shipLaborData
}

func NewMaritimeLaborSystem() *MaritimeLaborSystem {
	return &MaritimeLaborSystem{
		ships: make([]shipLaborData, 0, 20),
	}
}

func (s *MaritimeLaborSystem) Update(world *ecs.World) {
	s.tickStamp++
	isWageTick := s.tickStamp%100 == 0
	isHiringTick := s.tickStamp%50 == 0

	if !isWageTick && !isHiringTick {
		return
	}

	shipID := ecs.ComponentID[components.ShipComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	identID := ecs.ComponentID[components.Identity](world)

	s.ships = s.ships[:0]

	shipFilter := ecs.All(shipID, posID, treasuryID, identID)
	shipQuery := world.Query(shipFilter)

	for shipQuery.Next() {
		shipComp := (*components.ShipComponent)(shipQuery.Get(shipID))
		treasury := (*components.TreasuryComponent)(shipQuery.Get(treasuryID))
		pos := (*components.Position)(shipQuery.Get(posID))
		ident := (*components.Identity)(shipQuery.Get(identID))

		s.ships = append(s.ships, shipLaborData{
			entity:   shipQuery.Entity(),
			identID:  ident.ID,
			shipComp: shipComp,
			treasury: treasury,
			pos:      pos,
		})
	}

	npcID := ecs.ComponentID[components.NPC](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)

	npcFilter := ecs.All(npcID, posID, jobID, needsID)
	npcQuery := world.Query(npcFilter)

	for npcQuery.Next() {
		job := (*components.JobComponent)(npcQuery.Get(jobID))
		needs := (*components.Needs)(npcQuery.Get(needsID))
		pos := (*components.Position)(npcQuery.Get(posID))

		// 1. Process Wages
		justQuit := false
		if isWageTick && job.JobID == components.JobSailor {
			// Find employer ship
			foundShip := false
			for i := 0; i < len(s.ships); i++ {
				if s.ships[i].identID == job.EmployerID {
					foundShip = true
					wage := float32(5.0)
					if s.ships[i].treasury.Wealth >= wage {
						s.ships[i].treasury.Wealth -= wage
						needs.Wealth += wage
					} else {
						// Bankrupt! Crew quits.
						job.JobID = components.JobNone
						job.EmployerID = 0
						if s.ships[i].shipComp.CrewCurrent > 0 {
							s.ships[i].shipComp.CrewCurrent--
						}
						justQuit = true
					}
					break
				}
			}
			if !foundShip {
				// Ship doesn't exist anymore, crew quits.
				job.JobID = components.JobNone
				job.EmployerID = 0
				justQuit = true
			}
		}

		// If they just quit this same tick because of bankruptcy, they shouldn't instantly rehire.
		if justQuit {
			continue
		}

		// 2. Process Hiring
		if isHiringTick && job.JobID == components.JobNone && needs.Wealth < 50.0 {
			bestDistSq := float32(25.0) // Must be close to port
			bestShipIdx := -1

			for i := 0; i < len(s.ships); i++ {
				if s.ships[i].shipComp.CrewCurrent < s.ships[i].shipComp.CrewRequirements {
					dx := s.ships[i].pos.X - pos.X
					dy := s.ships[i].pos.Y - pos.Y
					distSq := dx*dx + dy*dy

					if distSq < bestDistSq {
						bestDistSq = distSq
						bestShipIdx = i
					}
				}
			}

			if bestShipIdx != -1 {
				// Hire NPC
				job.JobID = components.JobSailor
				job.EmployerID = s.ships[bestShipIdx].identID
				s.ships[bestShipIdx].shipComp.CrewCurrent++
			}
		}
	}
}
