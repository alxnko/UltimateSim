package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 13.2: Labor Rebalancing Test
func TestCareerChangeSystem(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	identityID := ecs.ComponentID[components.Identity](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	affiliationID := ecs.ComponentID[components.Affiliation](&world)

	// Setup Village with severe Food shortage
	villageEntity := world.NewEntity(villageID, marketID, identityID)

	villageIdentity := (*components.Identity)(world.Get(villageEntity, identityID))
	villageIdentity.ID = 100 // CityID

	villageMarket := (*components.MarketComponent)(world.Get(villageEntity, marketID))
	villageMarket.FoodPrice = 15.0 // High price > 10.0
	villageMarket.WoodPrice = 5.0

	// Setup NPC 1: Artisan in Village 100 (Should change to Farmer)
	npc1Entity := world.NewEntity(jobID, affiliationID)

	npc1Job := (*components.JobComponent)(world.Get(npc1Entity, jobID))
	npc1Job.JobID = components.JobArtisan

	npc1Affiliation := (*components.Affiliation)(world.Get(npc1Entity, affiliationID))
	npc1Affiliation.CityID = 100

	// Setup NPC 2: Artisan in Village 100 with normal Food but severe Wood shortage
	// Wait, villageMarket applies to both, so let's make another Village
	village2Entity := world.NewEntity(villageID, marketID, identityID)

	village2Identity := (*components.Identity)(world.Get(village2Entity, identityID))
	village2Identity.ID = 200 // CityID

	village2Market := (*components.MarketComponent)(world.Get(village2Entity, marketID))
	village2Market.FoodPrice = 5.0
	village2Market.WoodPrice = 12.0 // High price > 10.0

	npc2Entity := world.NewEntity(jobID, affiliationID)

	npc2Job := (*components.JobComponent)(world.Get(npc2Entity, jobID))
	npc2Job.JobID = components.JobArtisan

	npc2Affiliation := (*components.Affiliation)(world.Get(npc2Entity, affiliationID))
	npc2Affiliation.CityID = 200

	// Run System
	system := NewCareerChangeSystem()
	system.Update(&world)

	// Assertions
	npc1JobAfter := (*components.JobComponent)(world.Get(npc1Entity, jobID))
	if npc1JobAfter.JobID != components.JobFarmer {
		t.Errorf("Expected NPC1 JobID to be JobFarmer (1), got %d", npc1JobAfter.JobID)
	}

	npc2JobAfter := (*components.JobComponent)(world.Get(npc2Entity, jobID))
	if npc2JobAfter.JobID != components.JobLumberjack {
		t.Errorf("Expected NPC2 JobID to be JobLumberjack (2), got %d", npc2JobAfter.JobID)
	}
}
