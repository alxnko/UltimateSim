package main
import (
	"fmt"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

func main() {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	parentEnt := world.NewEntity(needsID, legacyID, identID, affilID, npcID)
	pId := (*components.Identity)(world.Get(parentEnt, identID))
	pId.ID = 101

	ruinID := ecs.ComponentID[components.RuinComponent](&world)
	mask := ecs.All(needsID).Without(ruinID)

	query := world.Query(&mask)
	for query.Next() {
		fmt.Printf("Has Ident: %v\n", query.Has(identID))
		fmt.Printf("Has Affil: %v\n", query.Has(affilID))

		if query.Has(identID) {
			ident := (*components.Identity)(query.Get(identID))
			fmt.Printf("Ident ID inside loop: %v\n", ident.ID)
		}
	}
}
