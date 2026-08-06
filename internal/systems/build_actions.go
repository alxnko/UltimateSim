package systems

import (
	"errors"
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: player-driven construction. The player funds a site upfront with
// carried materials; existing JobBuilder NPCs (or the player via HammerSite)
// raise its progress. Reuses the ConstructionSystem completion pipeline.

const buildHammerRange = 1.5

// BuildCosts returns the wood/stone cost of a structure type.
func BuildCosts(buildType uint8) (wood, stone uint32) {
	switch buildType {
	case components.StructureHouse:
		return 50, 50
	case components.StructureWorkshop:
		return 80, 40
	case components.StructureStorehouse:
		return 120, 80
	case components.StructureShrine:
		return 0, 100
	case components.StructureFarm:
		return 40, 0
	case components.StructureTavern:
		return 100, 60
	}
	return 50, 50
}

// CanPlaceReason validates placement of a structure at a world point. It
// returns ok=true when placement is allowed; on failure the string is a short
// human-readable reason ("water", "too close to a building", "not enough
// materials", ...) fit for the build-ghost overlay. Pass the zero Entity as
// player to run the geometry checks only (bounds, water, proximity) and skip
// the material-affordability check. This is the single source of placement
// truth: CanPlace and PlaceSite both defer to it.
func CanPlaceReason(world *ecs.World, grid *engine.MapGrid, player ecs.Entity, buildType uint8, wx, wy float32) (string, bool) {
	tx, ty := int(wx), int(wy)
	if tx < 0 || tx >= grid.Width || ty < 0 || ty >= grid.Height {
		return "out of bounds", false
	}
	if grid.GetTile(tx, ty).BiomeID == engine.BiomeOcean {
		return "water", false
	}

	posID := ecs.ComponentID[components.Position](world)
	tooClose := func(compID ecs.ID) bool {
		mask := ecs.All(compID, posID)
		q := world.Query(&mask)
		blocked := false
		for q.Next() { // iterate to exhaustion: no early break, no Close
			pos := (*components.Position)(q.Get(posID))
			dx := pos.X - wx
			dy := pos.Y - wy
			if dx*dx+dy*dy < 1.0 {
				blocked = true
			}
		}
		return blocked
	}
	if tooClose(ecs.ComponentID[components.StructureComponent](world)) {
		return "too close to a building", false
	}
	if tooClose(ecs.ComponentID[components.ConstructionSiteComponent](world)) {
		return "too close to a construction site", false
	}
	if tooClose(ecs.ComponentID[components.Village](world)) {
		return "too close to a settlement", false
	}

	// Affordability (only when a live player entity is supplied).
	if world.Alive(player) {
		wood, stone := BuildCosts(buildType)
		storID := ecs.ComponentID[components.StorageComponent](world)
		if !world.Has(player, storID) {
			return "not enough materials", false
		}
		stor := (*components.StorageComponent)(world.Get(player, storID))
		if stor.Wood < wood || stor.Stone < stone {
			return "not enough materials", false
		}
	}
	return "", true
}

// CanPlace validates that a structure may be placed at the world point.
// Geometry-only wrapper over CanPlaceReason (no material check).
func CanPlace(world *ecs.World, grid *engine.MapGrid, wx, wy float32) error {
	if reason, ok := CanPlaceReason(world, grid, ecs.Entity{}, 0, wx, wy); !ok {
		return errors.New(reason)
	}
	return nil
}

// PlaceSite validates, deducts player materials, and spawns a funded site.
func PlaceSite(world *ecs.World, grid *engine.MapGrid, player ecs.Entity, buildType uint8, wx, wy float32) (ecs.Entity, error) {
	if reason, ok := CanPlaceReason(world, grid, player, buildType, wx, wy); !ok {
		return ecs.Entity{}, errors.New(reason)
	}
	wood, stone := BuildCosts(buildType)

	storID := ecs.ComponentID[components.StorageComponent](world)
	if !world.Alive(player) || !world.Has(player, storID) {
		// CanPlaceReason already rejects live players without materials; this
		// guards the zero/dead-entity path, which skips the material check.
		return ecs.Entity{}, fmt.Errorf("insufficient materials: need %d wood, %d stone", wood, stone)
	}
	stor := (*components.StorageComponent)(world.Get(player, storID))
	stor.Wood -= wood
	stor.Stone -= stone

	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)

	var cityID uint32
	if world.Has(player, affID) {
		cityID = (*components.Affiliation)(world.Get(player, affID)).CityID
	}

	ent := world.NewEntity(posID, affID, siteID)
	pos := (*components.Position)(world.Get(ent, posID))
	pos.X, pos.Y = wx, wy
	aff := (*components.Affiliation)(world.Get(ent, affID))
	aff.CityID = cityID
	site := (*components.ConstructionSiteComponent)(world.Get(ent, siteID))
	site.WoodRequired = wood
	site.StoneRequired = stone
	site.WoodGathered = wood // player supplied materials upfront
	site.StoneGathered = stone
	site.MaxProgress = 100
	site.Progress = 0
	site.SiteType = uint32(buildType)
	site.BuilderID = 0

	return ent, nil
}

// NearestSite returns the closest construction site within radius of a point.
func NearestSite(world *ecs.World, wx, wy float32, radius float32) (ecs.Entity, bool) {
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	var best ecs.Entity
	bestD := radius * radius
	found := false
	mask := ecs.All(siteID, posID)
	q := world.Query(&mask)
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		dx := pos.X - wx
		dy := pos.Y - wy
		d := dx*dx + dy*dy
		if d <= bestD {
			bestD = d
			best = q.Entity()
			found = true
		}
	}
	return best, found
}

// HammerSite lets the player personally advance a nearby site.
func HammerSite(world *ecs.World, player, site ecs.Entity) error {
	posID := ecs.ComponentID[components.Position](world)
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)

	if !world.Has(player, posID) || !world.Has(site, siteID) {
		return fmt.Errorf("invalid hammer target")
	}
	pp := (*components.Position)(world.Get(player, posID))
	sp := (*components.Position)(world.Get(site, posID))
	dx := pp.X - sp.X
	dy := pp.Y - sp.Y
	if dx*dx+dy*dy > buildHammerRange*buildHammerRange {
		return fmt.Errorf("too far from the site")
	}
	if world.Has(player, vitalsID) {
		v := (*components.VitalsComponent)(world.Get(player, vitalsID))
		if v.Stamina <= 1 {
			return fmt.Errorf("too exhausted to work")
		}
		v.Stamina -= 1
	}
	s := (*components.ConstructionSiteComponent)(world.Get(site, siteID))
	s.Progress += 2
	if s.Progress > s.MaxProgress {
		s.Progress = s.MaxProgress
	}
	return nil
}
