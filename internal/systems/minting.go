package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 15.3: Currency & Debt

// coinSpawn records the parameters to spawn a physical CoinEntity
type coinSpawn struct {
	x, y     float32
	issuerID uint32
}

// MintingSystem manages the creation of physical coins by Capital cities.
type MintingSystem struct {
	world *ecs.World

	// Component IDs
	villageID   ecs.ID
	capitalID   ecs.ID
	storageID   ecs.ID
	affilID     ecs.ID
	posID       ecs.ID
	coinTagID   ecs.ID
	currencyID  ecs.ID

	tickStamp uint64
	toSpawn   []coinSpawn
}

// NewMintingSystem initializes the minting logic
func NewMintingSystem(world *ecs.World) *MintingSystem {
	return &MintingSystem{
		world:      world,
		villageID:  ecs.ComponentID[components.Village](world),
		capitalID:  ecs.ComponentID[components.CapitalComponent](world),
		storageID:  ecs.ComponentID[components.StorageComponent](world),
		affilID:    ecs.ComponentID[components.Affiliation](world),
		posID:      ecs.ComponentID[components.Position](world),
		coinTagID:  ecs.ComponentID[components.CoinEntity](world),
		currencyID: ecs.ComponentID[components.CurrencyComponent](world),
		toSpawn:    make([]coinSpawn, 0, 100),
	}
}

// Update evaluates Capital cities and mints currency if resources allow
func (s *MintingSystem) Update() {
	s.tickStamp++

	// Only process minting every 100 ticks to preserve processing time
	if s.tickStamp%100 != 0 {
		return
	}

	// Reset slice capacity without allocating new arrays to preserve GC constraints
	s.toSpawn = s.toSpawn[:0]

	// 1. Iterate over all Capital Villages with Storage and Position
	query := s.world.Query(filter.All(s.villageID, s.capitalID, s.storageID, s.affilID, s.posID))
	for query.Next() {
		storage := (*components.StorageComponent)(query.Get(s.storageID))
		affil := (*components.Affiliation)(query.Get(s.affilID))
		pos := (*components.Position)(query.Get(s.posID))

		// If the capital has sufficient raw material (e.g., Iron >= 100)
		if storage.Iron >= 100 {
			// Deduct the cost deterministically
			storage.Iron -= 100

			// Defer entity creation to avoid Arche-Go query nesting issues
			s.toSpawn = append(s.toSpawn, coinSpawn{
				x:        pos.X,
				y:        pos.Y,
				issuerID: affil.CityID,
			})
		}
	}

	// 2. Instantiate new physical coin entities outside the primary loop
	for _, spawn := range s.toSpawn {
		entity := s.world.NewEntity(s.coinTagID, s.posID, s.currencyID)

		// Apply Position matching the Mint location
		pos := (*components.Position)(s.world.Get(entity, s.posID))
		pos.X = spawn.x
		pos.Y = spawn.y

		// Apply baseline Currency value
		curr := (*components.CurrencyComponent)(s.world.Get(entity, s.currencyID))
		curr.IssuerID = spawn.issuerID
		curr.Value = 100.0 // Baseline nominal value before exchange rate modulation
	}
}
