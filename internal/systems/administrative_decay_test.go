package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 10.2: Bureaucratic Delay (Administrative Entropy)
// AdministrativeDecaySystem Tests

func TestAdministrativeDecaySystem(t *testing.T) {
	world := ecs.NewWorld()
	tm := engine.NewTickManager(60)

	sys := NewAdministrativeDecaySystem()
	tm.AddSystem(sys, engine.PhaseResolution)

	orderEntityID := ecs.ComponentID[components.OrderEntity](&world)
	orderCompID := ecs.ComponentID[components.OrderComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)

	villageID := ecs.ComponentID[components.Village](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)

	// Create Capital City (Optional, mostly for conceptual testing, we rely on TargetCityID)
	// Capital won't be explicitly queried by the decay system, only the Target City.

	// Create Target City (City 2) with Loyalty 5
	city2 := world.NewEntity()
	world.Add(city2, villageID, identID, loyaltyID)

	ident2 := (*components.Identity)(world.Get(city2, identID))
	ident2.ID = 2 // CityID = 2

	loyalty2 := (*components.LoyaltyComponent)(world.Get(city2, loyaltyID))
	loyalty2.Value = 5

	// Create Order Entity targeting City 2 at tick 0
	order1 := world.NewEntity()
	world.Add(order1, orderEntityID, orderCompID, posID)

	orderComp1 := (*components.OrderComponent)(world.Get(order1, orderCompID))
	orderComp1.CreationTick = 0
	orderComp1.TargetCityID = 2

	// Run ticks 1 to 5
	for i := 1; i <= 5; i++ {
		sys.Update(&world)

		// Verify Order survives while Decay <= Loyalty
		if !world.Alive(order1) {
			t.Errorf("OrderEntity despawned prematurely at tick %d (Decay: %d, Loyalty: 5)", i, i)
		}
	}

	// Run tick 6
	sys.Update(&world)

	// Verify Order despawns when Decay > Loyalty
	if world.Alive(order1) {
		t.Errorf("OrderEntity survived at tick 6 (Decay: 6, Loyalty: 5), expected it to despawn")
	}

	// Test missing target city handling
	// Create Order Entity targeting non-existent City 99
	order2 := world.NewEntity()
	world.Add(order2, orderEntityID, orderCompID, posID)

	orderComp2 := (*components.OrderComponent)(world.Get(order2, orderCompID))
	orderComp2.CreationTick = sys.Tick // Current tick
	orderComp2.TargetCityID = 99

	sys.Update(&world)

	if world.Alive(order2) {
		t.Errorf("OrderEntity targeting missing CityID 99 should have despawned immediately")
	}
}

// Test Deterministic execution
func TestAdministrativeDecaySystem_Deterministic(t *testing.T) {
	world1 := ecs.NewWorld()
	sys1 := NewAdministrativeDecaySystem()

	world2 := ecs.NewWorld()
	sys2 := NewAdministrativeDecaySystem()

	// Both worlds identical setup
	setupWorld := func(w *ecs.World, sys *AdministrativeDecaySystem) {
		orderEntityID := ecs.ComponentID[components.OrderEntity](w)
		orderCompID := ecs.ComponentID[components.OrderComponent](w)
		posID := ecs.ComponentID[components.Position](w)
		villageID := ecs.ComponentID[components.Village](w)
		identID := ecs.ComponentID[components.Identity](w)
		loyaltyID := ecs.ComponentID[components.LoyaltyComponent](w)

		// Create 100 cities with varying loyalty
		for i := 0; i < 100; i++ {
			c := w.NewEntity()
			w.Add(c, villageID, identID, loyaltyID)

			ident := (*components.Identity)(w.Get(c, identID))
			ident.ID = uint64(i)

			loyalty := (*components.LoyaltyComponent)(w.Get(c, loyaltyID))
			loyalty.Value = uint32(i % 10) // 0 to 9 loyalty
		}

		// Create 100 orders targeting those cities
		for i := 0; i < 100; i++ {
			o := w.NewEntity()
			w.Add(o, orderEntityID, orderCompID, posID)

			comp := (*components.OrderComponent)(w.Get(o, orderCompID))
			comp.CreationTick = 0
			comp.TargetCityID = uint32(i)
		}

		// Run for 5 ticks
		for i := 0; i < 5; i++ {
			sys.Update(w)
		}
	}

	setupWorld(&world1, sys1)
	setupWorld(&world2, sys2)

	// Compare active order entity counts
	orderEntityID1 := ecs.ComponentID[components.OrderEntity](&world1)
	orderEntityID2 := ecs.ComponentID[components.OrderEntity](&world2)

	filter1 := ecs.All(orderEntityID1)
	q1 := world1.Query(filter1)
	count1 := 0
	for q1.Next() {
		count1++
	}

	filter2 := ecs.All(orderEntityID2)
	q2 := world2.Query(filter2)
	count2 := 0
	for q2.Next() {
		count2++
	}

	if count1 != count2 {
		t.Errorf("Deterministic check failed: World1 orders = %d, World2 orders = %d", count1, count2)
	}
}
