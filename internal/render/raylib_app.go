package render

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/mlange-42/arche/ecs"
)

const TileSize = 16.0

// SelectionState tracks the currently selected entity.
type SelectionState struct {
	Entity   ecs.Entity
	Active   bool
	Name     string
	Type     string
	Details  string
}

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
	selection := SelectionState{}

	for !rl.WindowShouldClose() {
		world := tm.World
		// Drive the simulation before rendering
		tm.Tick()

		// Pause Toggle
		if rl.IsKeyPressed(rl.KeySpace) {
			tm.TogglePause()
		}

		// Toggle mode
		if rl.IsKeyPressed(rl.KeyP) || rl.IsKeyPressed(rl.KeyTab) {
			isPossessionMode = !isPossessionMode

			// Phase 12.2: Auto-Possession on Mode Switch
			if isPossessionMode && selection.Active {
				possessedID := ecs.ComponentID[components.Possessed](world)
				// Remove existing possessions first
				filter := ecs.All(possessedID)
				q := world.Query(filter)
				var toRemove []ecs.Entity
				for q.Next() {
					toRemove = append(toRemove, q.Entity())
				}
				for _, e := range toRemove {
					world.Remove(e, possessedID)
				}
				// Add to selected
				world.Add(selection.Entity, possessedID)
			}
		}

		// Optional: exit on ESC
		if rl.IsKeyPressed(rl.KeyEscape) {
			break
		}

		if isPossessionMode {
			// Phase 11.2: 3rd Person Controller Input Logic
			possessedID := ecs.ComponentID[components.Possessed](world)
			posID := ecs.ComponentID[components.Position](world)
			velID := ecs.ComponentID[components.Velocity](world)

			// Update First: Camera follow based on position ONLY
			filterPos := ecs.All(possessedID, posID)
			queryPos := world.Query(filterPos)
			for queryPos.Next() {
				pos := (*components.Position)(queryPos.Get(posID))
				// Update 3D Camera position to follow player cleanly
				cam3D.Position = rl.NewVector3(pos.X, 10.0, pos.Y+10.0)
				cam3D.Target = rl.NewVector3(pos.X, 0.0, pos.Y)
			}

			// Update Second: Movement logic (if entity has velocity)
			filterVel := ecs.All(possessedID, posID, velID)
			queryVel := world.Query(filterVel)

			for queryVel.Next() {
				vel := (*components.Velocity)(queryVel.Get(velID))

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
			}
		} else {
			// Selection Logic (Left Click)
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				mouseWorldPos := rl.GetScreenToWorld2D(rl.GetMousePosition(), cam2D)
				mapX := int(mouseWorldPos.X / TileSize)
				mapY := int(mouseWorldPos.Y / TileSize)

				if mapX >= 0 && mapX < mapGrid.Width && mapY >= 0 && mapY < mapGrid.Height {
					// Find nearest entity at mouse position
					selection.Active = false
					
					posID := ecs.ComponentID[components.Position](world)
					idID := ecs.ComponentID[components.Identity](world)
					query := world.Query(ecs.All(posID, idID))
					
					minDist := float32(1.0) // Selection radius
					for query.Next() {
						entityPos := (*components.Position)(query.Get(posID))
						dx := entityPos.X - float32(mapX)
						dy := entityPos.Y - float32(mapY)
						dist := dx*dx + dy*dy
						
						if dist < minDist {
							minDist = dist
							selection.Active = true
							selection.Entity = query.Entity()
							ident := (*components.Identity)(query.Get(idID))
							selection.Name = ident.Name
							selection.Type = "NPC"
							if world.Has(selection.Entity, ecs.ComponentID[components.Village](world)) {
								selection.Type = "Village"
							}
							
							// Details string
							details := fmt.Sprintf("ID: %d\nPos: (%.1f, %.1f)", ident.ID, entityPos.X, entityPos.Y)
							if world.Has(selection.Entity, ecs.ComponentID[components.Needs](world)) {
								needs := (*components.Needs)(world.Get(selection.Entity, ecs.ComponentID[components.Needs](world)))
								details += fmt.Sprintf("\nFood: %.1f", needs.Food)
							}
							selection.Details = details
						}
					}
				}
			}

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

			// Right-Click Drag Panning
			if rl.IsMouseButtonDown(rl.MouseRightButton) {
				delta := rl.GetMouseDelta()
				cam2D.Target.X -= delta.X / cam2D.Zoom
				cam2D.Target.Y -= delta.Y / cam2D.Zoom
			}

			// Handle Zooming via Mouse Wheel (Zoom towards Mouse Position)
			wheel := rl.GetMouseWheelMove()
			if wheel != 0 {
				mouseWorldPos := rl.GetScreenToWorld2D(rl.GetMousePosition(), cam2D)

				// Set offset to current mouse position (screen space)
				// This makes the zoom "pin" to the mouse cursor
				cam2D.Offset = rl.GetMousePosition()
				// Set target to the world position at this same point
				cam2D.Target = mouseWorldPos

				// Apply Zoom
				if wheel > 0 {
					cam2D.Zoom *= 1.1
				} else {
					cam2D.Zoom /= 1.1
				}

				// Clamp Zoom
				if cam2D.Zoom < 0.1 {
					cam2D.Zoom = 0.1
				} else if cam2D.Zoom > 10.0 {
					cam2D.Zoom = 10.0
				}
			}
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		if isPossessionMode {
			rl.BeginMode3D(cam3D)

			// Render 3D World Context (Terrain cubes)
			// Optimizing: only render around the camera area
			viewDist := 30
			startX := int(cam3D.Target.X) - viewDist
			endX := int(cam3D.Target.X) + viewDist
			startZ := int(cam3D.Target.Z) - viewDist
			endZ := int(cam3D.Target.Z) + viewDist

			for x := startX; x < endX; x++ {
				for z := startZ; z < endZ; z++ {
					if x >= 0 && x < mapGrid.Width && z >= 0 && z < mapGrid.Height {
						tile := mapGrid.GetTile(x, z)
						// In 3D mode, the "white block" might be due to skipping BiomeOcean poorly or just huge oceans.
						// We'll draw Ocean as a darker, lower plane.
						color := GetBiomeColorRL(tile.BiomeID)
						height := float32(0.2)
						yPos := float32(-0.1)
						
						if tile.BiomeID == engine.BiomeOcean {
							height = 0.05
							yPos = -0.15
						}

						rl.DrawCubeV(rl.NewVector3(float32(x), yPos, float32(z)), rl.NewVector3(1.0, height, 1.0), color)
					}
				}
			}

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
			foundAny := false
			for query.Next() {
				foundAny = true
				pos := (*components.Position)(query.Get(posID))
				rl.DrawCube(rl.NewVector3(pos.X, 0.5, pos.Y), 0.5, 1.0, 0.5, rl.Red)
				rl.DrawCubeWires(rl.NewVector3(pos.X, 0.5, pos.Y), 0.5, 1.0, 0.5, rl.Maroon)
			}

			rl.DrawGrid(100, 1.0)

			rl.EndMode3D()

			if !foundAny {
				rl.DrawText("NO ENTITY POSSESSED. Select one on Map first!", 400, 300, 20, rl.Red)
			}
			rl.DrawText("Possession Mode Active. Press 'TAB' to return to Map.", 10, 10, 20, rl.RayWhite)
			rl.DrawText("WASD to Move.", 10, 40, 20, rl.RayWhite)
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

			// Selection Highlight (2D)
			if selection.Active && world.Alive(selection.Entity) {
				if world.Has(selection.Entity, posID) {
					pos := (*components.Position)(world.Get(selection.Entity, posID))
					drawX := float64(pos.X)
					drawY := float64(pos.Y)
					rectSize := int32(12)
					rl.DrawRectangleLines(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.Yellow)
				}
			}

			rl.EndMode2D()

			rl.DrawText("Map Mode Active. Press 'P' to Possess an NPC.", 10, 10, 20, rl.RayWhite)
		}

		// --- HUD Overlay (Static) ---
		rl.DrawRectangle(0, int32(rl.GetScreenHeight())-100, int32(rl.GetScreenWidth()), 100, rl.Fade(rl.Black, 0.7))
		
		statusStr := "RUNNING"
		statusColor := rl.Green
		if tm.IsPaused {
			statusStr = "PAUSED"
			statusColor = rl.Red
		}
		
		rl.DrawText(fmt.Sprintf("STATUS: %s (SPACE to toggle)", statusStr), 20, int32(rl.GetScreenHeight())-80, 20, statusColor)
		rl.DrawText(fmt.Sprintf("FPS: %d", rl.GetFPS()), 20, int32(rl.GetScreenHeight())-50, 20, rl.RayWhite)
		
		// Selection Sidebar/Box
		if selection.Active {
			boxWidth := int32(250)
			rl.DrawRectangle(int32(rl.GetScreenWidth())-boxWidth, 0, boxWidth, int32(rl.GetScreenHeight()), rl.Fade(rl.DarkGray, 0.8))
			rl.DrawText("SELECTION", int32(rl.GetScreenWidth())-boxWidth+10, 20, 20, rl.Gold)
			rl.DrawText(selection.Name, int32(rl.GetScreenWidth())-boxWidth+10, 50, 18, rl.RayWhite)
			rl.DrawText(selection.Type, int32(rl.GetScreenWidth())-boxWidth+10, 75, 14, rl.Gray)
			rl.DrawText(selection.Details, int32(rl.GetScreenWidth())-boxWidth+10, 110, 16, rl.RayWhite)
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

// Map Biome IDs to Raylib colors (constants are 0-indexed iota)
func GetBiomeColorRL(biomeID uint8) rl.Color {
	switch biomeID {
	case engine.BiomeOcean:
		return rl.NewColor(0, 105, 148, 255)
	case engine.BiomeBeach:
		return rl.NewColor(238, 214, 175, 255)
	case engine.BiomeScorched:
		return rl.NewColor(85, 85, 85, 255)
	case engine.BiomeBare:
		return rl.NewColor(136, 136, 136, 255)
	case engine.BiomeTundra:
		return rl.NewColor(221, 221, 255, 255)
	case engine.BiomeSnow:
		return rl.NewColor(255, 255, 255, 255)
	case engine.BiomeTemperateDesert:
		return rl.NewColor(201, 210, 155, 255)
	case engine.BiomeShrubland:
		return rl.NewColor(136, 153, 119, 255)
	case engine.BiomeGrassland:
		return rl.NewColor(136, 170, 85, 255)
	case engine.BiomeTemperateDeciduousForest:
		return rl.NewColor(103, 148, 89, 255)
	case engine.BiomeTemperateRainForest:
		return rl.NewColor(70, 130, 80, 255) // Distinct from deciduous
	case engine.BiomeSubtropicalDesert:
		return rl.NewColor(210, 185, 139, 255)
	case engine.BiomeTropicalSeasonalForest:
		return rl.NewColor(85, 153, 68, 255)
	case engine.BiomeTropicalRainForest:
		return rl.NewColor(51, 119, 85, 255)
	case engine.BiomeMountain:
		return rl.NewColor(139, 137, 137, 255)
	default: // BiomeUnknown
		return rl.NewColor(255, 0, 255, 255) // Purple for actual unknown
	}
}
