package ui

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase: display-name helpers for political entities.

// CityName resolves a settlement's display name from its city ID.
func CityName(world *ecs.World, cityID uint32) string {
	if cityID == 0 {
		return "the wilds"
	}
	villageID := ecs.ComponentID[components.Village](world)
	idID := ecs.ComponentID[components.Identity](world)
	name := fmt.Sprintf("City %d", cityID)
	q := world.Query(ecs.All(villageID, idID))
	for q.Next() {
		ident := (*components.Identity)(q.Get(idID))
		if uint32(ident.ID) == cityID && ident.Name != "" {
			name = ident.Name
		}
	}
	return name
}
