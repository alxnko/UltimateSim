package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func BenchmarkPenalLaborSystem_Update(b *testing.B) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	villID := ecs.ComponentID[components.Village](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	penalID := ecs.ComponentID[components.PenalLaborComponent](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
    _ = jobID
    _ = needsID

	// Create City
	cityEnt := world.NewEntity()
	world.Add(cityEnt, villID, posID, storID, affID, idID)
	cAff := (*components.Affiliation)(world.Get(cityEnt, affID))
	cAff.CityID = 1
	cIdent := (*components.Identity)(world.Get(cityEnt, idID))
	cIdent.ID = 100

	// Create 1000 Convicts at (10, 10)
	for i := 0; i < 1000; i++ {
		convict := world.NewEntity()
		world.Add(convict, penalID, posID, idID)
		cvPos := (*components.Position)(world.Get(convict, posID))
		cvPos.X, cvPos.Y = 10.0, 10.0
		cvPenal := (*components.PenalLaborComponent)(world.Get(convict, penalID))
		cvPenal.StateCityID = 1
		cvPenal.RemainingSentence = 1000
	}

	// Create 100 Abolitionists at (12, 12) - close enough to witness (distSq = 2^2 + 2^2 = 8 < 100)
	for i := 0; i < 100; i++ {
		abol := world.NewEntity()
		world.Add(abol, posID, idID)
		abPos := (*components.Position)(world.Get(abol, posID))
		abPos.X, abPos.Y = 12.0, 12.0
		abIdent := (*components.Identity)(world.Get(abol, idID))
		abIdent.ID = uint64(1000 + i)
		abIdent.BaseTraits = components.TraitAbolitionist
	}

	sys := NewPenalLaborSystem(&world, hooks)
	// Trigger the backlash logic (tick % 10 == 0)
	sys.tickCounter = 9

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sys.Update(&world)
        // Reset tickCounter to ensure laborEvents are generated in each iteration if needed,
        // though we want to measure the backlash loop which depends on laborEvents.
        sys.tickCounter = 9
	}
}
