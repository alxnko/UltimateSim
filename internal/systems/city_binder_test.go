package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 06.1: Societal Hierarchies Test

func TestCityBinderSystem(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	famClusterID := ecs.ComponentID[components.FamilyCluster](&world)

	// Spawn Villages
	v1 := world.NewEntity()
	world.Add(v1, posID, villageID, identID)
	posV1 := (*components.Position)(world.Get(v1, posID))
	identV1 := (*components.Identity)(world.Get(v1, identID))
	posV1.X, posV1.Y = 10, 10
	identV1.ID = 101

	v2 := world.NewEntity()
	world.Add(v2, posID, villageID, identID)
	posV2 := (*components.Position)(world.Get(v2, posID))
	identV2 := (*components.Identity)(world.Get(v2, identID))
	posV2.X, posV2.Y = 100, 100
	identV2.ID = 202

	// Spawn wandering clusters
	c1 := world.NewEntity()
	world.Add(c1, posID, affID, famClusterID)
	posC1 := (*components.Position)(world.Get(c1, posID))
	affC1 := (*components.Affiliation)(world.Get(c1, affID))
	posC1.X, posC1.Y = 12, 12 // Closer to v1 (101)

	c2 := world.NewEntity()
	world.Add(c2, posID, affID, famClusterID)
	posC2 := (*components.Position)(world.Get(c2, posID))
	affC2 := (*components.Affiliation)(world.Get(c2, affID))
	posC2.X, posC2.Y = 90, 95 // Closer to v2 (202)

	sys := &CityBinderSystem{TicksElapsed: 9999}
	sys.Update(&world) // Tick 10000

	// Verify
	if affC1.CityID != 101 {
		t.Errorf("Expected cluster 1 to bind to CityID 101, got %d", affC1.CityID)
	}
	if affC2.CityID != 202 {
		t.Errorf("Expected cluster 2 to bind to CityID 202, got %d", affC2.CityID)
	}
}

func TestCityBinderDeterminism(t *testing.T) {
	tm := engine.NewTickManager(60)
	tm.AddSystem(&CityBinderSystem{}, engine.PhaseResolution)

	world := tm.World

	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	identID := ecs.ComponentID[components.Identity](world)
	villageID := ecs.ComponentID[components.Village](world)
	famClusterID := ecs.ComponentID[components.FamilyCluster](world)

	// Add Villages
	v1 := world.NewEntity()
	world.Add(v1, posID, villageID, identID)
	posV1 := (*components.Position)(world.Get(v1, posID))
	identV1 := (*components.Identity)(world.Get(v1, identID))
	posV1.X, posV1.Y = 50, 50
	identV1.ID = 555

	v2 := world.NewEntity()
	world.Add(v2, posID, villageID, identID)
	posV2 := (*components.Position)(world.Get(v2, posID))
	identV2 := (*components.Identity)(world.Get(v2, identID))
	posV2.X, posV2.Y = 200, 200
	identV2.ID = 777

	// Add Wanderers
	numClusters := 50
	for i := 0; i < numClusters; i++ {
		c := world.NewEntity()
		world.Add(c, posID, affID, famClusterID)
		posC := (*components.Position)(world.Get(c, posID))
		posC.X = float32(i * 5)
		posC.Y = float32(i * 5)
	}

	tm.Run(10000)

	// Collect final states
	query := world.Query(filter.All(affID))
	count := 0
	var sumCityIDs uint32 = 0
	for query.Next() {
		aff := (*components.Affiliation)(query.Get(affID))
		sumCityIDs += aff.CityID
		count++
	}

	// Simple check, count should be numClusters and sum should be deterministic
	if count != numClusters {
		t.Errorf("Expected %d clusters, got %d", numClusters, count)
	}

	// Determinism output verification
	if sumCityIDs != 33078 {
		t.Errorf("Deterministic sum mismatch: expected 33078, got %d", sumCityIDs)
	}
}
