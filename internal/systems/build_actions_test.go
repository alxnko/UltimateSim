package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: player-driven construction action tests.
// Deterministic, headless coverage of BuildCosts, CanPlace, PlaceSite,
// NearestSite, and HammerSite. No RNG is involved, so runs are reproducible.

// fillLand sets every tile of the grid to a buildable (non-ocean) biome so
// CanPlace's water check does not reject placement everywhere. A fresh
// NewMapGrid has zero-value tiles, and BiomeOcean == 0, so without this the
// entire grid would read as water.
func fillLand(grid *engine.MapGrid) {
	for i := range grid.Tiles {
		grid.Tiles[i].BiomeID = engine.BiomeGrassland
	}
}

// spawnPlayerEntity creates a player with a stocked StorageComponent and an
// Affiliation, mirroring the funded-construction flow used by PlaceSite.
func spawnPlayerEntity(world *ecs.World, wood, stone, cityID uint32) ecs.Entity {
	storID := ecs.ComponentID[components.StorageComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	e := world.NewEntity(storID, affID)
	stor := (*components.StorageComponent)(world.Get(e, storID))
	stor.Wood = wood
	stor.Stone = stone
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.CityID = cityID
	return e
}

// spawnExistingStructure spawns a StructureComponent+Position entity to test
// CanPlace overlap rejection.
func spawnExistingStructure(world *ecs.World, x, y float32) ecs.Entity {
	structID := ecs.ComponentID[components.StructureComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	e := world.NewEntity(structID, posID)
	pos := (*components.Position)(world.Get(e, posID))
	pos.X, pos.Y = x, y
	return e
}

// TestBuildCosts verifies the documented wood/stone cost per structure type,
// including the default fallback for an unknown type.
func TestBuildCosts(t *testing.T) {
	cases := []struct {
		name      string
		buildType uint8
		wantWood  uint32
		wantStone uint32
	}{
		{"House", components.StructureHouse, 50, 50},
		{"Workshop", components.StructureWorkshop, 80, 40},
		{"Storehouse", components.StructureStorehouse, 120, 80},
		{"Shrine", components.StructureShrine, 0, 100},
		{"Farm", components.StructureFarm, 40, 0},
		{"Tavern", components.StructureTavern, 100, 60},
		{"UnknownDefault", uint8(200), 50, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wood, stone := BuildCosts(tc.buildType)
			if wood != tc.wantWood || stone != tc.wantStone {
				t.Errorf("BuildCosts(%d) = (%d, %d), want (%d, %d)",
					tc.buildType, wood, stone, tc.wantWood, tc.wantStone)
			}
		})
	}
}

// TestCanPlace covers the land OK path, ocean rejection, overlap rejection, and
// out-of-bounds rejection.
func TestCanPlace(t *testing.T) {
	world := ecs.NewWorld()
	// Register the component IDs CanPlace queries against so the world knows them.
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.StructureComponent](&world)
	ecs.ComponentID[components.ConstructionSiteComponent](&world)
	ecs.ComponentID[components.Village](&world)

	grid := engine.NewMapGrid(16, 16)
	fillLand(grid)

	// Land placement on an empty grassland tile succeeds.
	if err := CanPlace(&world, grid, 5, 5); err != nil {
		t.Fatalf("CanPlace on land = %v, want nil", err)
	}

	// Ocean rejection: flip the target tile to BiomeOcean.
	grid.Tiles[8*grid.Width+8].BiomeID = engine.BiomeOcean
	if err := CanPlace(&world, grid, 8, 8); err == nil {
		t.Errorf("CanPlace on ocean tile = nil, want error")
	}

	// Out-of-bounds rejection (negative and beyond extent).
	if err := CanPlace(&world, grid, -1, 5); err == nil {
		t.Errorf("CanPlace at x=-1 = nil, want out-of-bounds error")
	}
	if err := CanPlace(&world, grid, 16, 5); err == nil {
		t.Errorf("CanPlace at x=16 = nil, want out-of-bounds error")
	}
	if err := CanPlace(&world, grid, 5, 16); err == nil {
		t.Errorf("CanPlace at y=16 = nil, want out-of-bounds error")
	}

	// Overlap rejection: an existing structure at (5,5) blocks placement there.
	spawnExistingStructure(&world, 5, 5)
	if err := CanPlace(&world, grid, 5, 5); err == nil {
		t.Errorf("CanPlace overlapping a structure = nil, want error")
	}
	// A tile far from the structure (distance^2 >= 1.0) is still placeable.
	if err := CanPlace(&world, grid, 10, 10); err != nil {
		t.Errorf("CanPlace away from structure = %v, want nil", err)
	}
}

// TestPlaceSite verifies a funded site spawn: materials deducted, site fully
// funded, affiliation carried over, and the insufficient-materials error path.
func TestPlaceSite(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.StructureComponent](&world)
	ecs.ComponentID[components.ConstructionSiteComponent](&world)
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.StorageComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)

	grid := engine.NewMapGrid(16, 16)
	fillLand(grid)

	player := spawnPlayerEntity(&world, 200, 200, 3)

	ent, err := PlaceSite(&world, grid, player, components.StructureHouse, 5, 5)
	if err != nil {
		t.Fatalf("PlaceSite returned error: %v", err)
	}

	// Materials deducted by the House cost (50/50): 200 -> 150.
	storID := ecs.ComponentID[components.StorageComponent](&world)
	stor := (*components.StorageComponent)(world.Get(player, storID))
	if stor.Wood != 150 || stor.Stone != 150 {
		t.Errorf("player storage after PlaceSite = (wood %d, stone %d), want (150, 150)", stor.Wood, stor.Stone)
	}

	// Site is fully funded at the House cost with MaxProgress 100 and zero start.
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	site := (*components.ConstructionSiteComponent)(world.Get(ent, siteID))
	if site.WoodRequired != 50 || site.WoodGathered != 50 {
		t.Errorf("site wood = (req %d, gathered %d), want (50, 50)", site.WoodRequired, site.WoodGathered)
	}
	if site.StoneRequired != 50 || site.StoneGathered != 50 {
		t.Errorf("site stone = (req %d, gathered %d), want (50, 50)", site.StoneRequired, site.StoneGathered)
	}
	if site.MaxProgress != 100 {
		t.Errorf("site MaxProgress = %d, want 100", site.MaxProgress)
	}
	if site.Progress != 0 {
		t.Errorf("site Progress = %d, want 0", site.Progress)
	}
	if site.SiteType != uint32(components.StructureHouse) {
		t.Errorf("site SiteType = %d, want %d", site.SiteType, components.StructureHouse)
	}

	// Position placed at the requested world point.
	posID := ecs.ComponentID[components.Position](&world)
	pos := (*components.Position)(world.Get(ent, posID))
	if pos.X != 5 || pos.Y != 5 {
		t.Errorf("site position = (%v, %v), want (5, 5)", pos.X, pos.Y)
	}

	// Affiliation CityID carried over from the player (3).
	affID := ecs.ComponentID[components.Affiliation](&world)
	aff := (*components.Affiliation)(world.Get(ent, affID))
	if aff.CityID != 3 {
		t.Errorf("site Affiliation.CityID = %d, want 3", aff.CityID)
	}
}

// TestPlaceSiteInsufficientMaterials confirms placement fails (with no
// deduction) when the player cannot afford the structure.
func TestPlaceSiteInsufficientMaterials(t *testing.T) {
	world := ecs.NewWorld()
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.StructureComponent](&world)
	ecs.ComponentID[components.ConstructionSiteComponent](&world)
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.StorageComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)

	grid := engine.NewMapGrid(16, 16)
	fillLand(grid)

	// Only 10 wood / 10 stone: a House (50/50) is unaffordable.
	player := spawnPlayerEntity(&world, 10, 10, 1)

	if _, err := PlaceSite(&world, grid, player, components.StructureHouse, 5, 5); err == nil {
		t.Fatalf("PlaceSite with insufficient materials = nil error, want error")
	}

	// Storage must be untouched on the failure path.
	storID := ecs.ComponentID[components.StorageComponent](&world)
	stor := (*components.StorageComponent)(world.Get(player, storID))
	if stor.Wood != 10 || stor.Stone != 10 {
		t.Errorf("player storage after failed PlaceSite = (wood %d, stone %d), want (10, 10)", stor.Wood, stor.Stone)
	}
}

// TestNearestSite verifies the closest in-radius site is returned, and that a
// site outside the radius is not found.
func TestNearestSite(t *testing.T) {
	world := ecs.NewWorld()
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)

	spawnSiteAt := func(x, y float32) ecs.Entity {
		e := world.NewEntity(siteID, posID)
		p := (*components.Position)(world.Get(e, posID))
		p.X, p.Y = x, y
		return e
	}

	near := spawnSiteAt(5, 5) // dist 0 from query
	spawnSiteAt(7, 5)         // dist 2 from query (farther)

	got, found := NearestSite(&world, 5, 5, 3)
	if !found {
		t.Fatalf("NearestSite found = false, want true")
	}
	if got != near {
		t.Errorf("NearestSite returned the wrong (non-nearest) entity")
	}

	// Nothing within a tight radius around a far-off point.
	if _, found := NearestSite(&world, 0, 0, 1); found {
		t.Errorf("NearestSite at (0,0) r=1 found = true, want false")
	}
}

// TestHammerSite covers the in-range progress+stamina path, the too-far
// rejection, and the exhausted rejection.
func TestHammerSite(t *testing.T) {
	world := ecs.NewWorld()
	posID := ecs.ComponentID[components.Position](&world)
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	newSite := func(x, y float32) ecs.Entity {
		e := world.NewEntity(siteID, posID)
		p := (*components.Position)(world.Get(e, posID))
		p.X, p.Y = x, y
		s := (*components.ConstructionSiteComponent)(world.Get(e, siteID))
		s.MaxProgress = 100
		return e
	}
	newPlayer := func(x, y, stamina float32) ecs.Entity {
		e := world.NewEntity(posID, vitalsID)
		p := (*components.Position)(world.Get(e, posID))
		p.X, p.Y = x, y
		v := (*components.VitalsComponent)(world.Get(e, vitalsID))
		v.Stamina = stamina
		return e
	}

	// In-range (distance 1.0 < buildHammerRange 1.5), healthy stamina.
	site := newSite(5, 5)
	player := newPlayer(6, 5, 10)
	if err := HammerSite(&world, player, site); err != nil {
		t.Fatalf("HammerSite in range = %v, want nil", err)
	}
	s := (*components.ConstructionSiteComponent)(world.Get(site, siteID))
	if s.Progress != 2 {
		t.Errorf("Progress after one hammer = %d, want 2", s.Progress)
	}
	v := (*components.VitalsComponent)(world.Get(player, vitalsID))
	if v.Stamina != 9 {
		t.Errorf("Stamina after one hammer = %v, want 9", v.Stamina)
	}

	// Too far: distance 2.0 > 1.5 range. Progress and stamina unchanged.
	farPlayer := newPlayer(7, 5, 10)
	if err := HammerSite(&world, farPlayer, site); err == nil {
		t.Errorf("HammerSite too far = nil, want error")
	}
	if s.Progress != 2 {
		t.Errorf("Progress after a failed (too far) hammer = %d, want 2", s.Progress)
	}
	if fv := (*components.VitalsComponent)(world.Get(farPlayer, vitalsID)); fv.Stamina != 10 {
		t.Errorf("far player stamina = %v, want 10 (unchanged)", fv.Stamina)
	}

	// Exhausted: stamina <= 1 is rejected with no progress and no stamina drain.
	tiredPlayer := newPlayer(6, 5, 1)
	if err := HammerSite(&world, tiredPlayer, site); err == nil {
		t.Errorf("HammerSite while exhausted = nil, want error")
	}
	if s.Progress != 2 {
		t.Errorf("Progress after exhausted hammer = %d, want 2", s.Progress)
	}
	if tv := (*components.VitalsComponent)(world.Get(tiredPlayer, vitalsID)); tv.Stamina != 1 {
		t.Errorf("exhausted player stamina = %v, want 1 (unchanged)", tv.Stamina)
	}
}

// TestHammerSiteProgressClamp verifies Progress never exceeds MaxProgress.
func TestHammerSiteProgressClamp(t *testing.T) {
	world := ecs.NewWorld()
	posID := ecs.ComponentID[components.Position](&world)
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	site := world.NewEntity(siteID, posID)
	sp := (*components.Position)(world.Get(site, posID))
	sp.X, sp.Y = 5, 5
	s := (*components.ConstructionSiteComponent)(world.Get(site, siteID))
	s.MaxProgress = 5
	s.Progress = 4

	player := world.NewEntity(posID, vitalsID)
	pp := (*components.Position)(world.Get(player, posID))
	pp.X, pp.Y = 5, 5
	pv := (*components.VitalsComponent)(world.Get(player, vitalsID))
	pv.Stamina = 10

	if err := HammerSite(&world, player, site); err != nil {
		t.Fatalf("HammerSite = %v, want nil", err)
	}
	// 4 + 2 = 6 would exceed MaxProgress 5; must clamp to 5.
	if s.Progress != 5 {
		t.Errorf("Progress after clamping hammer = %d, want 5", s.Progress)
	}
}
