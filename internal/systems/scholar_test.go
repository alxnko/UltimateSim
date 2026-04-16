package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

func TestScholarSystem_Unit(t *testing.T) {
	tests := []struct {
		name           string
		intellect      uint8
		wealth         float32
		secrets        []components.Secret
		ticks          int
		expectLedger   bool
		expectedWealth float32
	}{
		{
			name:           "Success - NPC meets all criteria",
			intellect:      150,
			wealth:         100.0,
			secrets:        []components.Secret{{SecretID: 1}},
			ticks:          100,
			expectLedger:   true,
			expectedWealth: 50.0,
		},
		{
			name:           "Failure - Low intellect",
			intellect:      149,
			wealth:         100.0,
			secrets:        []components.Secret{{SecretID: 1}},
			ticks:          100,
			expectLedger:   false,
			expectedWealth: 100.0,
		},
		{
			name:           "Failure - Low wealth",
			intellect:      150,
			wealth:         49.9,
			secrets:        []components.Secret{{SecretID: 1}},
			ticks:          100,
			expectLedger:   false,
			expectedWealth: 49.9,
		},
		{
			name:           "Failure - No secrets",
			intellect:      150,
			wealth:         100.0,
			secrets:        []components.Secret{},
			ticks:          100,
			expectLedger:   false,
			expectedWealth: 100.0,
		},
		{
			name:           "Failure - Wrong tick",
			intellect:      150,
			wealth:         100.0,
			secrets:        []components.Secret{{SecretID: 1}},
			ticks:          99,
			expectLedger:   false,
			expectedWealth: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// New world per test case for perfect isolation and to avoid Arche world lock issues
			world := ecs.NewWorld()

			// Component IDs
			npcID := ecs.ComponentID[components.NPC](&world)
			posID := ecs.ComponentID[components.Position](&world)
			genID := ecs.ComponentID[components.GenomeComponent](&world)
			needsID := ecs.ComponentID[components.Needs](&world)
			secretID := ecs.ComponentID[components.SecretComponent](&world)
			ledgerTagID := ecs.ComponentID[components.Ledger](&world)
			ledgerCompID := ecs.ComponentID[components.LedgerComponent](&world)

			// Setup entity
			entity := world.NewEntity(npcID, posID, genID, needsID, secretID)

			gen := (*components.GenomeComponent)(world.Get(entity, genID))
			gen.Intellect = tt.intellect

			needs := (*components.Needs)(world.Get(entity, needsID))
			needs.Wealth = tt.wealth

			sec := (*components.SecretComponent)(world.Get(entity, secretID))
			sec.Secrets = tt.secrets

			sys := NewScholarSystem(&world)
			for i := 0; i < tt.ticks; i++ {
				sys.Update(&world)
			}

			// Check for Ledger
			f := filter.All(ledgerTagID, ledgerCompID)
			query := world.Query(f)
			ledgerCount := 0
			for query.Next() {
				ledgerCount++
			}

			if tt.expectLedger {
				if ledgerCount == 0 {
					t.Errorf("Expected ledger to be created, but none found")
				}
			} else {
				if ledgerCount > 0 {
					t.Errorf("Expected no ledger to be created, but found %d", ledgerCount)
				}
			}

			if needs.Wealth != tt.expectedWealth {
				t.Errorf("Expected wealth to be %f, got %f", tt.expectedWealth, needs.Wealth)
			}
		})
	}
}
