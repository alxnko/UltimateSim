package render

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/mlange-42/arche/ecs"
)

const TileSize = 16.0

// RunRaylibApp is the unified rendering loop using Raylib for both 2D (Map Mode) and 3D (Possession Mode).
// Phase 11: Switch Pattern & Instanced 3D Control
func RunRaylibApp(tm *engine.TickManager, mapGrid *engine.MapGrid) {
	rl.InitWindow(1280, 720, "Boundless Sovereigns")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// 2D Camera for Map Mode
	cam2D := rl.Camera2D{
		Target:   rl.NewVector2(0, 0),
		Offset:   rl.NewVector2(0, 0),
		Rotation: 0,
		Zoom:     1.0,
	}

	// 3D Camera for Possession Mode
	cam3D := rl.Camera{}
	cam3D.Position = rl.NewVector3(0.0, 10.0, 10.0)
	cam3D.Target = rl.NewVector3(0.0, 0.0, 0.0)
	cam3D.Up = rl.NewVector3(0.0, 1.0, 0.0)
	cam3D.Fovy = 45.0
	cam3D.Projection = rl.CameraPerspective

	// Disable ESC key closing the window so we can handle it manually if needed, or map it to Unpossess
	rl.SetExitKey(0)

	isPossessionMode := false

	for !rl.WindowShouldClose() {
		// Toggle mode
		if rl.IsKeyPressed(rl.KeyP) {
			isPossessionMode = !isPossessionMode
		}

		// Optional: exit on ESC
		if rl.IsKeyPressed(rl.KeyEscape) {
			break
		}

		world := tm.World

		if isPossessionMode {
			// Phase 11.2: 3rd Person Controller Input Logic
			possessedID := ecs.ComponentID[components.Possessed](world)
			posID := ecs.ComponentID[components.Position](world)
			velID := ecs.ComponentID[components.Velocity](world)

			filter := ecs.All(possessedID, posID, velID)
			query := world.Query(filter)

			for query.Next() {
				vel := (*components.Velocity)(query.Get(velID))
				pos := (*components.Position)(query.Get(posID))

				// Clear velocity
				vel.X = 0
				vel.Y = 0

				speed := float32(0.5)

				if rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeyUp) {
					vel.Y = -speed
				}
				if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
					vel.Y = speed
				}
				if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
					vel.X = -speed
				}
				if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
					vel.X = speed
				}

				// Update 3D Camera position to follow player
				cam3D.Position = rl.NewVector3(pos.X, 10.0, pos.Y+10.0)
				cam3D.Target = rl.NewVector3(pos.X, 0.0, pos.Y)
			}
		} else {
			// Handle 2D Panning/Zooming
			panSpeed := float32(10.0) / cam2D.Zoom
			if rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeyUp) {
				cam2D.Target.Y -= panSpeed
			}
			if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
				cam2D.Target.Y += panSpeed
			}
			if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
				cam2D.Target.X -= panSpeed
			}
			if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
				cam2D.Target.X += panSpeed
			}

			// Handle Zooming via Mouse Wheel
			wheel := rl.GetMouseWheelMove()
			if wheel > 0 {
				cam2D.Zoom *= 1.1 // Zoom in
			} else if wheel < 0 {
				cam2D.Zoom /= 1.1 // Zoom out
			}

			// Prevent zooming too far out or in
			if cam2D.Zoom < 0.1 {
				cam2D.Zoom = 0.1
			} else if cam2D.Zoom > 10.0 {
				cam2D.Zoom = 10.0
			}
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		if isPossessionMode {
			rl.BeginMode3D(cam3D)

			// Phase 11.2: Instanced Rendering Placeholder
			// Draw one 3D generic "House Model" across the StorageComponent bounds
			// For now we'll draw basic cubes representing villages/entities
			posID := ecs.ComponentID[components.Position](world)
			villageID := ecs.ComponentID[components.Village](world)
			possessedID := ecs.ComponentID[components.Possessed](world)

			villageFilter := ecs.All(villageID, posID)
			villageQuery := world.Query(villageFilter)

			for villageQuery.Next() {
				vPos := (*components.Position)(villageQuery.Get(posID))

				// Draw simple house (cube)
				rl.DrawCube(rl.NewVector3(vPos.X, 0.5, vPos.Y), 1.0, 1.0, 1.0, rl.Blue)
				rl.DrawCubeWires(rl.NewVector3(vPos.X, 0.5, vPos.Y), 1.0, 1.0, 1.0, rl.DarkBlue)
			}

			// Draw possessed entity
			filter := ecs.All(possessedID, posID)
			query := world.Query(filter)
			for query.Next() {
				pos := (*components.Position)(query.Get(posID))
				rl.DrawCube(rl.NewVector3(pos.X, 0.5, pos.Y), 0.5, 1.0, 0.5, rl.Red)
				rl.DrawCubeWires(rl.NewVector3(pos.X, 0.5, pos.Y), 0.5, 1.0, 0.5, rl.Maroon)
			}

			// Draw simple ground
			rl.DrawGrid(100, 1.0)

			rl.EndMode3D()

			rl.DrawText("Possession Mode Active. Press 'P' to return to Map.", 10, 10, 20, rl.RayWhite)
		} else {
			rl.BeginMode2D(cam2D)

			// Phase 08.3: Map Rendering & Biomes using Raylib
			if mapGrid != nil {
				for y := 0; y < mapGrid.Height; y++ {
					for x := 0; x < mapGrid.Width; x++ {
						tile := mapGrid.GetTile(x, y)
						c := GetBiomeColorRL(tile.BiomeID)

						// Phase 08.5: Visualizing Desire Paths
						// Dynamic Floor Updates: Override base Biome color if FootTraffic is high
						state := mapGrid.TileStates[y*mapGrid.Width+x]
						if state.FootTraffic > 100 { // Example RenderThreshold
							// Blend towards dirt color #8B4513 (139, 69, 19) based on traffic
							c = rl.NewColor(139, 69, 19, 255)
						}

						// Draw rect
						rl.DrawRectangle(int32(float64(x)*TileSize), int32(float64(y)*TileSize), int32(TileSize), int32(TileSize), c)
					}
				}
			}

			// Phase 08.4: Entity rendering
			posID := ecs.ComponentID[components.Position](world)
			velID := ecs.ComponentID[components.Velocity](world)
			familyID := ecs.ComponentID[components.FamilyCluster](world)
			villageID := ecs.ComponentID[components.Village](world)
			ruinID := ecs.ComponentID[components.RuinComponent](world)

			alpha := tm.Alpha

			// Wandering AI Clusters (FamilyCluster)
			filterFamily := ecs.All(posID, familyID)
			queryFamily := world.Query(filterFamily)
			for queryFamily.Next() {
				pos := (*components.Position)(queryFamily.Get(posID))
				drawX := float64(pos.X)
				drawY := float64(pos.Y)

				if queryFamily.Has(velID) {
					vel := (*components.Velocity)(queryFamily.Get(velID))
					drawX += float64(vel.X) * alpha
					drawY += float64(vel.Y) * alpha
				}

				rectSize := int32(4)
				rl.DrawRectangle(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.Red)
			}

			// Villages
			filterVillage := ecs.All(posID, villageID)
			queryVillage := world.Query(filterVillage)
			for queryVillage.Next() {
				pos := (*components.Position)(queryVillage.Get(posID))
				drawX := float64(pos.X)
				drawY := float64(pos.Y)
				rectSize := int32(8)
				rl.DrawRectangle(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.Blue)
			}

			// Ruins
			filterRuin := ecs.All(posID, ruinID)
			queryRuin := world.Query(filterRuin)
			for queryRuin.Next() {
				pos := (*components.Position)(queryRuin.Get(posID))
				drawX := float64(pos.X)
				drawY := float64(pos.Y)
				rectSize := int32(8)
				rl.DrawRectangle(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.DarkGray)
			}

			rl.EndMode2D()

			rl.DrawText("Map Mode Active. Press 'P' to Possess an NPC.", 10, 10, 20, rl.RayWhite)
		}

		rl.EndDrawing()
	}

	// Clean up Possessed tag when exiting
	// ecs query to remove the Possessed tag from all entities
	world := tm.World
	possessedID := ecs.ComponentID[components.Possessed](world)
	filter := ecs.All(possessedID)
	query := world.Query(filter)
	var possessedEntities []ecs.Entity
	for query.Next() {
		possessedEntities = append(possessedEntities, query.Entity())
	}
	for _, e := range possessedEntities {
		world.Remove(e, possessedID)
	}
}

// Map Biome IDs to Raylib colors (copied from previous Ebiten colors map)
func GetBiomeColorRL(biomeID uint8) rl.Color {
	switch biomeID {
	case 1: // BiomeOcean
		return rl.NewColor(0, 105, 148, 255)
	case 2: // BiomeBeach
		return rl.NewColor(238, 214, 175, 255)
	case 3: // BiomeScorched
		return rl.NewColor(85, 85, 85, 255)
	case 4: // BiomeBare
		return rl.NewColor(136, 136, 136, 255)
	case 5: // BiomeTundra
		return rl.NewColor(221, 221, 255, 255)
	case 6: // BiomeSnow
		return rl.NewColor(255, 255, 255, 255)
	case 7: // BiomeTemperateDesert
		return rl.NewColor(201, 210, 155, 255)
	case 8: // BiomeShrubland
		return rl.NewColor(136, 153, 119, 255)
	case 9: // BiomeGrassland
		return rl.NewColor(136, 170, 85, 255)
	case 10: // BiomeDeciduousForest
		return rl.NewColor(103, 148, 89, 255)
	case 11: // BiomeSubtropicalDesert
		return rl.NewColor(210, 185, 139, 255)
	case 12: // BiomeTropicalSeasonalForest
		return rl.NewColor(85, 153, 68, 255)
	case 13: // BiomeTropicalRainforest
		return rl.NewColor(51, 119, 85, 255)
	default: // BiomeUnknown
		return rl.NewColor(255, 0, 255, 255)
	}
}
