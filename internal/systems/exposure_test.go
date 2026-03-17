package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 38.1: The Exposure Engine E2E Test (Butterfly Effect)
func TestExposureSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	grid := engine.NewMapGrid(10, 10)

	// Inject extreme temperature at coordinate (5, 5) -> index 55
	grid.Tiles[55].Temperature = 255

	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	// Entity 1: Exposed and Vulnerable
	e1 := world.NewEntity(posID, vitalsID, needsID)
	pos1 := (*components.Position)(world.Get(e1, posID))
	vitals1 := (*components.VitalsComponent)(world.Get(e1, vitalsID))
	needs1 := (*components.Needs)(world.Get(e1, needsID))

	pos1.X = 5.0
	pos1.Y = 5.0
	vitals1.Pain = 0.0
	needs1.Safety = 0.0

	// Entity 2: Protected
	e2 := world.NewEntity(posID, vitalsID, needsID)
	pos2 := (*components.Position)(world.Get(e2, posID))
	vitals2 := (*components.VitalsComponent)(world.Get(e2, vitalsID))
	needs2 := (*components.Needs)(world.Get(e2, needsID))

	pos2.X = 5.0
	pos2.Y = 5.0
	vitals2.Pain = 0.0
	needs2.Safety = 100.0 // Full protection

	// Entity 3: Safe Temperature, Exposed
	e3 := world.NewEntity(posID, vitalsID, needsID)
	pos3 := (*components.Position)(world.Get(e3, posID))
	vitals3 := (*components.VitalsComponent)(world.Get(e3, vitalsID))
	needs3 := (*components.Needs)(world.Get(e3, needsID))

	// Position 0,0 has default Temperature 0 (which is < 50, so let's set it to 100 to be safe)
	grid.Tiles[0].Temperature = 100
	pos3.X = 0.0
	pos3.Y = 0.0
	vitals3.Pain = 0.0
	needs3.Safety = 0.0

	sys := NewExposureSystem(&world, grid)

	// Run system
	sys.Update(&world)

	// Verify Entity 1 took pain damage (Heatstroke)
	if vitals1.Pain <= 0.0 {
		t.Errorf("Expected Entity 1 (exposed) to suffer pain from extreme heat, but Pain is %f", vitals1.Pain)
	}

	// Verify Entity 2 is protected
	if vitals2.Pain > 0.0 {
		t.Errorf("Expected Entity 2 (protected) to have 0 Pain, but got %f", vitals2.Pain)
	}

	// Verify Entity 3 is safe
	if vitals3.Pain > 0.0 {
		t.Errorf("Expected Entity 3 (safe temperature) to have 0 Pain, but got %f", vitals3.Pain)
	}
}
