package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func BenchmarkDesperationSystem_Update(b *testing.B) {
	world := ecs.NewWorld()

	needsID := ecs.ComponentID[components.Needs](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)

	// Create 100 Villages with a Market and Storage
	for i := 0; i < 100; i++ {
		villageEntity := world.NewEntity(posID, affID, marketID, villageID, storageID)
		vPos := (*components.Position)(world.Get(villageEntity, posID))
		vPos.X, vPos.Y = float32(i*10), float32(i*10)

		vAff := (*components.Affiliation)(world.Get(villageEntity, affID))
		vAff.CityID = uint32(i + 1)

		vMarket := (*components.MarketComponent)(world.Get(villageEntity, marketID))
		vMarket.FoodPrice = 50.0

		vStorage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
		vStorage.Food = 100
	}

	// Create 1000 Desperate NPCs
	for i := 0; i < 1000; i++ {
		npcEntity := world.NewEntity(needsID, affID, posID, despID, memID, idID)
		nPos := (*components.Position)(world.Get(npcEntity, posID))
		nPos.X, nPos.Y = float32(i), float32(i)

		nAff := (*components.Affiliation)(world.Get(npcEntity, affID))
		nAff.CityID = uint32((i % 100) + 1)

		nNeeds := (*components.Needs)(world.Get(npcEntity, needsID))
		nNeeds.Food = 20.0
		nNeeds.Wealth = 10.0

		nDesp := (*components.DesperationComponent)(world.Get(npcEntity, despID))
		nDesp.Level = 50 // Trigger crime
	}

	despSys := NewDesperationSystem(&world)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		despSys.Update(&world)
	}
}
