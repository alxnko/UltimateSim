package render

import (
	"fmt"
	"sync"

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

// GameState tracks the UI state of the application.
type GameState int

const (
	StateMainMenu GameState = iota
	StateWorldGen
	StatePlaying
	StatePaused
)

// BuildSimFunc is the function type to construct the engine.
type BuildSimFunc func(gridWidth, gridHeight int, seedVal byte) (*engine.TickManager, *engine.MapGrid)

// RunRaylibApp is the unified rendering loop using Raylib for both 2D (Map Mode) and 3D (Possession Mode).
// Phase 11: Switch Pattern & Instanced 3D Control
func RunRaylibApp(buildSim BuildSimFunc) {
	rl.InitWindow(1280, 720, "Boundless Sovereigns")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	currentState := StateMainMenu
	simulateYears := 10 // Default to 10 years of simulation in WorldGen

	// World Gen settings
	gridSizeStr := "Medium (256x256)"
	gridSize := 256
	seedVal := byte(1)

	var tm *engine.TickManager
	var mapGrid *engine.MapGrid

	// Simulation state variables
	isSimulating := false
	ticksSimulated := 0
	ticksToSimulate := 0

	// Map Lenses
	type LensType int
	const (
		LensPhysical LensType = iota
		LensPolitical
		LensEconomic
		LensSocial
	)
	mapLenses := []string{"Physical", "Political", "Economic", "Social"}
	currentLens := LensPhysical

	// Precomputed structures for complex lenses
	type voronoiCell struct { ctryID uint32 }
	var voronoiMap []voronoiCell
	var lastVoronoiTick uint64 = 0
	isBuildingVoronoi := false
	var voronoiMutex sync.RWMutex

	// Heatmap array
	var socialHeatmap []uint8
	var lastHeatmapTick uint64 = 0

	// Extract calendar from tick manager (hack: approximate based on ticks if no direct access)
	// We'll calculate it inside the loop.

	// Pre-load a basic Mesh and Material for Instanced Rendering of Villages
	cubeMesh := rl.GenMeshCube(1.0, 1.0, 1.0)
	cubeMaterial := rl.LoadMaterialDefault()

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
		var world *ecs.World
		if tm != nil {
			world = tm.World
		}

		// --- STATE MANAGEMENT ---
		if currentState == StateMainMenu {
			rl.BeginDrawing()
			rl.ClearBackground(rl.DarkBlue)

			rl.DrawText("BOUNDLESS SOVEREIGNS", 350, 150, 50, rl.RayWhite)

			newGameBtn := rl.Rectangle{X: 540, Y: 300, Width: 200, Height: 50}
			loadGameBtn := rl.Rectangle{X: 540, Y: 380, Width: 200, Height: 50}
			settingsBtn := rl.Rectangle{X: 540, Y: 460, Width: 200, Height: 50}

			mousePos := rl.GetMousePosition()

			// Draw Buttons
			drawButton(newGameBtn, "NEW GAME", mousePos)
			drawButton(loadGameBtn, "LOAD GAME", mousePos)
			drawButton(settingsBtn, "SETTINGS", mousePos)

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				if rl.CheckCollisionPointRec(mousePos, newGameBtn) {
					currentState = StateWorldGen
				} else if rl.CheckCollisionPointRec(mousePos, loadGameBtn) {
					// Load Game logic
					db, err := engine.InitDB("test_save.db")
					if err == nil {
						// Extract saved map parameters
						_, savedW, savedH, savedSeed, errState := engine.LoadGameState(db)
						if errState != nil {
							savedW, savedH, savedSeed = 256, 256, 1
						}

						// Setup base engine with matching dimensions and seed
						tm, mapGrid = buildSim(savedW, savedH, savedSeed)

						// Clear world and load entities
						err = engine.LoadWorld(tm, db)
						if err == nil {
							world = tm.World
							currentState = StatePlaying
							fmt.Println("Game Loaded Successfully")
						} else {
							fmt.Println("Failed to load world:", err)
						}
						db.Close()
					} else {
						fmt.Println("Failed to open DB:", err)
					}
				} else if rl.CheckCollisionPointRec(mousePos, settingsBtn) {
					// Placeholder for settings
					fmt.Println("Settings Clicked")
				}
			}

			rl.EndDrawing()
			continue
		} else if currentState == StateWorldGen {
			rl.BeginDrawing()
			rl.ClearBackground(rl.Black)

			rl.DrawText("WORLD GENERATION", 400, 100, 40, rl.RayWhite)

			mousePos := rl.GetMousePosition()

			// Map Size UI
			rl.DrawText(fmt.Sprintf("Map Size: %s", gridSizeStr), 450, 180, 20, rl.LightGray)
			sizeBtn := rl.Rectangle{X: 750, Y: 175, Width: 100, Height: 30}
			drawButton(sizeBtn, "Toggle", mousePos)

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, sizeBtn) {
				if gridSize == 256 {
					gridSize = 512
					gridSizeStr = "Large (512x512)"
				} else if gridSize == 512 {
					gridSize = 1024
					gridSizeStr = "Huge (1024x1024)"
				} else if gridSize == 1024 {
					gridSize = 2048
					gridSizeStr = "Massive (2048x2048)"
				} else if gridSize == 2048 {
					gridSize = 128
					gridSizeStr = "Small (128x128)"
				} else {
					gridSize = 256
					gridSizeStr = "Medium (256x256)"
				}
			}

			// Conceptual Scale UI
			rl.DrawText("Conceptual Scale: 1 Tile = 100m", 450, 210, 14, rl.DarkGray)

			// Seed UI
			rl.DrawText(fmt.Sprintf("Seed: %d", seedVal), 450, 230, 20, rl.LightGray)
			seedBtn := rl.Rectangle{X: 750, Y: 225, Width: 100, Height: 30}
			drawButton(seedBtn, "Random", mousePos)

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, seedBtn) {
				seedVal = byte(rl.GetRandomValue(1, 255))
			}

			// Simulate Years UI
			rl.DrawText(fmt.Sprintf("Simulate History: %d Years", simulateYears), 450, 280, 20, rl.LightGray)

			minusBtn := rl.Rectangle{X: 400, Y: 275, Width: 30, Height: 30}
			plusBtn := rl.Rectangle{X: 750, Y: 275, Width: 30, Height: 30}

			drawButton(minusBtn, "-", mousePos)
			drawButton(plusBtn, "+", mousePos)

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				if rl.CheckCollisionPointRec(mousePos, minusBtn) && simulateYears > 0 {
					simulateYears--
				} else if rl.CheckCollisionPointRec(mousePos, plusBtn) && simulateYears < 500 {
					simulateYears++
				}
			}

			if !isSimulating {
				startBtn := rl.Rectangle{X: 540, Y: 380, Width: 200, Height: 50}
				drawButton(startBtn, "GENERATE", mousePos)

				if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, startBtn) {
					rl.DrawText("Building Engine... Please wait.", 500, 450, 20, rl.Yellow)
					rl.EndDrawing() // Force draw before blocking

					tm, mapGrid = buildSim(gridSize, gridSize, seedVal)

					// Re-fetch world after build
					world = tm.World

					if simulateYears > 0 {
						// Fast-forward simulation: 1 year = TicksPerDay * 6 (days) * 4 (months)
						ticksToSimulate = simulateYears * engine.TicksPerDay * 6 * 4
						ticksSimulated = 0
						isSimulating = true
					} else {
						currentState = StatePlaying
					}
					continue // Skip the rest of the loop for this frame
				}
			} else {
				// Process a chunk of ticks per frame to prevent freezing
				chunkSize := 10000
				for i := 0; i < chunkSize; i++ {
					if ticksSimulated < ticksToSimulate {
						tm.Tick()
						ticksSimulated++
					} else {
						break
					}
				}

				// Draw Progress Bar
				rl.DrawText("Simulating History...", 500, 400, 20, rl.RayWhite)
				progress := float32(ticksSimulated) / float32(ticksToSimulate)
				rl.DrawRectangle(440, 430, int32(400*progress), 30, rl.Green)
				rl.DrawRectangleLines(440, 430, 400, 30, rl.RayWhite)
				rl.DrawText(fmt.Sprintf("%d %%", int(progress*100)), 600, 435, 20, rl.Black)

				if ticksSimulated >= ticksToSimulate {
					isSimulating = false
					currentState = StatePlaying
				}
			}

			rl.EndDrawing()
			continue
		} else if currentState == StatePaused {
			if rl.IsKeyPressed(rl.KeySpace) {
				tm.IsPaused = false
				currentState = StatePlaying
			}
		} else if currentState == StatePlaying {
			// Drive the simulation before rendering
			tm.Tick()

			// Auto-save logic
			if tm.Ticks % 600 == 0 && tm.Ticks > 0 { // Prevent instant save on tick 0
				db, err := engine.InitDB("test_save.db")
				if err == nil {
					engine.SaveWorld(tm, mapGrid, seedVal, db)
					db.Close()
					fmt.Println("Auto-saved game state at tick:", tm.Ticks)
				} else {
					fmt.Println("Auto-save failed:", err)
				}
			}

			// Pause Toggle
			if rl.IsKeyPressed(rl.KeySpace) {
				tm.IsPaused = true
				currentState = StatePaused
			}
		}

		// Map Lenses Toggle
		if rl.IsKeyPressed(rl.KeyL) {
			currentLens = (currentLens + 1) % LensType(len(mapLenses))
		}

		// Precompute Maps if needed and stale
		if currentLens == LensPolitical && tm != nil && (tm.Ticks - lastVoronoiTick > 60 || lastVoronoiTick == 0) && mapGrid != nil && !isBuildingVoronoi {
			isBuildingVoronoi = true

			voronoiMutex.RLock()
			if len(voronoiMap) != mapGrid.Width * mapGrid.Height {
				voronoiMutex.RUnlock()
				voronoiMutex.Lock()
				voronoiMap = make([]voronoiCell, mapGrid.Width * mapGrid.Height)
				voronoiMutex.Unlock()
			} else {
				voronoiMutex.RUnlock()
			}

			// Collect villages synchronously to avoid ECS locking panics
			type vData struct { x, y float32; ctry uint32 }
			var villages []vData

			vID := ecs.ComponentID[components.Village](world)
			pID := ecs.ComponentID[components.Position](world)
			aID := ecs.ComponentID[components.Affiliation](world)
			vq := world.Query(ecs.All(vID, pID, aID))
			for vq.Next() {
				p := (*components.Position)(vq.Get(pID))
				a := (*components.Affiliation)(vq.Get(aID))
				villages = append(villages, vData{x: p.X, y: p.Y, ctry: a.CountryID})
			}

			// Offload heavy distance calculations to goroutine
			mapW := mapGrid.Width
			mapH := mapGrid.Height
			go func(vList []vData, w, h int, tick uint64) {
				newVoronoi := make([]voronoiCell, w*h)
				for vy := 0; vy < h; vy++ {
					for vx := 0; vx < w; vx++ {
						closestDist := float32(9999999.0)
						closestCtry := uint32(0)

						for _, v := range vList {
							dx := float32(vx) - v.x
							dy := float32(vy) - v.y
							distSq := dx*dx + dy*dy
							if distSq < closestDist {
								closestDist = distSq
								closestCtry = v.ctry
							}
						}

						// Limit influence radius to 30 tiles
						if closestDist > 900.0 {
							closestCtry = 0
						}
						newVoronoi[vy*w+vx].ctryID = closestCtry
					}
				}
				voronoiMutex.Lock()
				voronoiMap = newVoronoi
				lastVoronoiTick = tick
				isBuildingVoronoi = false
				voronoiMutex.Unlock()
			}(villages, mapW, mapH, tm.Ticks)
		}

		if currentLens == LensSocial && tm != nil && (tm.Ticks - lastHeatmapTick > 60 || lastHeatmapTick == 0) && mapGrid != nil {
			if len(socialHeatmap) != mapGrid.Width * mapGrid.Height {
				socialHeatmap = make([]uint8, mapGrid.Width * mapGrid.Height)
			} else {
				// Clear
				for i := range socialHeatmap { socialHeatmap[i] = 0 }
			}

			popID := ecs.ComponentID[components.PopulationComponent](world)
			pID := ecs.ComponentID[components.Position](world)
			pq := world.Query(ecs.All(popID, pID))
			for pq.Next() {
				p := (*components.Position)(pq.Get(pID))
				pop := (*components.PopulationComponent)(pq.Get(popID))

				// Apply heat locally
				px := int(p.X)
				py := int(p.Y)
				radius := int(2 + pop.Count/10)

				for hy := py - radius; hy <= py + radius; hy++ {
					for hx := px - radius; hx <= px + radius; hx++ {
						if hx >= 0 && hx < mapGrid.Width && hy >= 0 && hy < mapGrid.Height {
							dx := hx - px
							dy := hy - py
							if dx*dx + dy*dy <= radius*radius {
								idx := hy * mapGrid.Width + hx
								heat := int(socialHeatmap[idx]) + int(pop.Count*5)
								if heat > 255 { heat = 255 }
								socialHeatmap[idx] = uint8(heat)
							}
						}
					}
				}
			}
			lastHeatmapTick = tm.Ticks
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
			// Optimizing: aggressive Frustum Culling around camera target
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

			// Phase 11.2: Frustum Culled 3D Entities & Instanced Rendering
			// For high density, we map coordinates to transformation matrices and draw them via GPU instances
			posID := ecs.ComponentID[components.Position](world)
			villageID := ecs.ComponentID[components.Village](world)
			possessedID := ecs.ComponentID[components.Possessed](world)

			villageFilter := ecs.All(villageID, posID)
			villageQuery := world.Query(villageFilter)

			var transforms []rl.Matrix
			for villageQuery.Next() {
				vPos := (*components.Position)(villageQuery.Get(posID))

				// Frustum culling check
				if vPos.X >= float32(startX) && vPos.X <= float32(endX) && vPos.Y >= float32(startZ) && vPos.Y <= float32(endZ) {
					mat := rl.MatrixTranslate(vPos.X, 0.5, vPos.Y)
					transforms = append(transforms, mat)
				}
			}

			if len(transforms) > 0 {
				cubeMaterial.Maps.Color = rl.Blue
				rl.DrawMeshInstanced(cubeMesh, cubeMaterial, transforms, len(transforms))
			}

			// Draw possessed entity (drawn normally to easily distinct it)
			filter := ecs.All(possessedID, posID)
			query := world.Query(filter)
			foundAny := false
			for query.Next() {
				foundAny = true
				pos := (*components.Position)(query.Get(posID))
				rl.DrawCube(rl.NewVector3(pos.X, 0.5, pos.Y), 0.5, 1.0, 0.5, rl.Red)
				rl.DrawCubeWires(rl.NewVector3(pos.X, 0.5, pos.Y), 0.5, 1.0, 0.5, rl.Maroon)
			}

			rl.EndMode3D()

			if !foundAny {
				rl.DrawText("NO ENTITY POSSESSED. Select one on Map first!", 400, 300, 20, rl.Red)
			}
			rl.DrawText("Possession Mode Active. Press 'TAB' to return to Map.", 10, 10, 20, rl.RayWhite)
			rl.DrawText("WASD to Move.", 10, 40, 20, rl.RayWhite)
		} else {
			rl.BeginMode2D(cam2D)

			// Phase 08.3: Map Rendering & Biomes using Raylib
			// Frustum Culling calculation for 2D map
			screenWidth := float32(rl.GetScreenWidth())
			screenHeight := float32(rl.GetScreenHeight())

			// Calculate the top-left and bottom-right world coordinates visible on screen
			// We expand the bounds slightly to prevent edge pop-in
			topLeft := rl.GetScreenToWorld2D(rl.NewVector2(0, 0), cam2D)
			bottomRight := rl.GetScreenToWorld2D(rl.NewVector2(screenWidth, screenHeight), cam2D)

			startX := int(topLeft.X/TileSize) - 2
			startY := int(topLeft.Y/TileSize) - 2
			endX := int(bottomRight.X/TileSize) + 2
			endY := int(bottomRight.Y/TileSize) + 2

			if startX < 0 { startX = 0 }
			if startY < 0 { startY = 0 }
			if endX > mapGrid.Width { endX = mapGrid.Width }
			if endY > mapGrid.Height { endY = mapGrid.Height }

			if mapGrid != nil {
				for y := startY; y < endY; y++ {
					for x := startX; x < endX; x++ {
						tile := mapGrid.GetTile(x, y)

						var c rl.Color

						// Lens Logic
						switch currentLens {
						case LensPhysical:
							c = GetBiomeColorRL(tile.BiomeID)
							// Visualizing Desire Paths
							state := mapGrid.TileStates[y*mapGrid.Width+x]
							if state.FootTraffic > 100 {
								c = rl.NewColor(139, 69, 19, 255) // Dirt color
							}
						case LensPolitical:
							c = rl.DarkGray
							voronoiMutex.RLock()
							if len(voronoiMap) > 0 {
								ctry := voronoiMap[y*mapGrid.Width+x].ctryID
								if ctry > 0 {
									// Generate stable color from ctry ID
									r := uint8((ctry * 43) % 255)
									g := uint8((ctry * 79) % 255)
									b := uint8((ctry * 101) % 255)
									c = rl.NewColor(r, g, b, 255)
								}
							}
							voronoiMutex.RUnlock()
						case LensEconomic:
							res := mapGrid.Resources[y*mapGrid.Width+x]
							// Heatmap based on resource values
							c = rl.NewColor(res.FoodValue, res.WoodValue, res.StoneValue, 255)
						case LensSocial:
							c = rl.Black
							if len(socialHeatmap) > 0 {
								heat := socialHeatmap[y*mapGrid.Width+x]
								c = rl.NewColor(heat, 0, 0, 255) // Red heatmap
							}
						}

						// Draw rect
						rl.DrawRectangle(int32(float64(x)*TileSize), int32(float64(y)*TileSize), int32(TileSize), int32(TileSize), c)
					}
				}
			}

			// Phase 08.4: Entity rendering
			posID := ecs.ComponentID[components.Position](world)
			velID := ecs.ComponentID[components.Velocity](world)
			npcID := ecs.ComponentID[components.NPC](world)
			villageID := ecs.ComponentID[components.Village](world)
			ruinID := ecs.ComponentID[components.RuinComponent](world)

			alpha := tm.Alpha

			// Wandering AI NPCs
			filterFamily := ecs.All(posID, npcID)
			queryFamily := world.Query(filterFamily)
			for queryFamily.Next() {
				pos := (*components.Position)(queryFamily.Get(posID))
				// Frustum culling check
				if pos.X >= float32(startX) && pos.X <= float32(endX) && pos.Y >= float32(startY) && pos.Y <= float32(endY) {
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
			}

			// Villages
			filterVillage := ecs.All(posID, villageID)
			queryVillage := world.Query(filterVillage)
			for queryVillage.Next() {
				pos := (*components.Position)(queryVillage.Get(posID))
				// Frustum culling check
				if pos.X >= float32(startX) && pos.X <= float32(endX) && pos.Y >= float32(startY) && pos.Y <= float32(endY) {
					drawX := float64(pos.X)
					drawY := float64(pos.Y)
					rectSize := int32(8)
					rl.DrawRectangle(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.Blue)
				}
			}

			// Ruins
			filterRuin := ecs.All(posID, ruinID)
			queryRuin := world.Query(filterRuin)
			for queryRuin.Next() {
				pos := (*components.Position)(queryRuin.Get(posID))
				// Frustum culling check
				if pos.X >= float32(startX) && pos.X <= float32(endX) && pos.Y >= float32(startY) && pos.Y <= float32(endY) {
					drawX := float64(pos.X)
					drawY := float64(pos.Y)
					rectSize := int32(8)
					rl.DrawRectangle(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.DarkGray)
				}
			}

			// Selection Highlight (2D)
			if selection.Active && world.Alive(selection.Entity) {
				if world.Has(selection.Entity, posID) {
					pos := (*components.Position)(world.Get(selection.Entity, posID))
					// Always draw selection if it exists, or check cull:
					if pos.X >= float32(startX) && pos.X <= float32(endX) && pos.Y >= float32(startY) && pos.Y <= float32(endY) {
						drawX := float64(pos.X)
						drawY := float64(pos.Y)
						rectSize := int32(12)
						rl.DrawRectangleLines(int32(drawX*TileSize)-rectSize/2, int32(drawY*TileSize)-rectSize/2, rectSize, rectSize, rl.Yellow)
					}
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
		
		// HUD: Calendar calculated from tm.Ticks
		ticks := tm.Ticks
		days := (ticks / engine.TicksPerDay) % 6 + 1
		months := (ticks / (engine.TicksPerDay * 6)) % 4 + 1
		years := (ticks / (engine.TicksPerDay * 6 * 4)) + 1

		rl.DrawText(fmt.Sprintf("Year %d | Month %d | Day %d", years, months, days), 300, int32(rl.GetScreenHeight())-50, 20, rl.RayWhite)

		// Status Dashboard
		if isPossessionMode {
			possessedID := ecs.ComponentID[components.Possessed](world)
			needsID := ecs.ComponentID[components.Needs](world)
			vitalsID := ecs.ComponentID[components.VitalsComponent](world)
			jobID := ecs.ComponentID[components.JobComponent](world)
			affID := ecs.ComponentID[components.Affiliation](world)

			q := world.Query(ecs.All(possessedID))
			for q.Next() {
				ent := q.Entity()
				rl.DrawText("POSSESSED DASHBOARD", 600, int32(rl.GetScreenHeight())-80, 16, rl.Gold)

				yPos := int32(rl.GetScreenHeight()) - 60

				// Needs
				if world.Has(ent, needsID) {
					needs := (*components.Needs)(world.Get(ent, needsID))
					rl.DrawText(fmt.Sprintf("Food: %.1f | Wealth: %.1f", needs.Food, needs.Wealth), 600, yPos, 14, rl.RayWhite)
					yPos += 20
				}

				// Vitals
				if world.Has(ent, vitalsID) {
					vitals := (*components.VitalsComponent)(world.Get(ent, vitalsID))
					rl.DrawText(fmt.Sprintf("Health: %.1f | Pain: %.1f | Blood: %.1f", vitals.Stamina, vitals.Pain, vitals.Blood), 600, yPos, 14, rl.RayWhite)
					yPos += 20
				}

				// Job & Affiliation
				jobStr := "None"
				if world.Has(ent, jobID) {
					job := (*components.JobComponent)(world.Get(ent, jobID))
					jobStr = fmt.Sprintf("%d", job.JobID)
				}
				affStr := "None"
				if world.Has(ent, affID) {
					aff := (*components.Affiliation)(world.Get(ent, affID))
					affStr = fmt.Sprintf("City: %d | Ctry: %d", aff.CityID, aff.CountryID)
				}
				rl.DrawText(fmt.Sprintf("Job: %s | %s", jobStr, affStr), 600, yPos, 14, rl.RayWhite)
			}
		}

		// Map Lenses Display
		rl.DrawText(fmt.Sprintf("LENS: %s (L to cycle)", mapLenses[currentLens]), 300, int32(rl.GetScreenHeight())-80, 20, rl.Gold)

		// Selection Sidebar/Box & Deep Inspection
		if selection.Active {
			boxWidth := int32(250)
			startX := int32(rl.GetScreenWidth()) - boxWidth
			rl.DrawRectangle(startX, 0, boxWidth, int32(rl.GetScreenHeight()), rl.Fade(rl.DarkGray, 0.8))
			rl.DrawText("SELECTION", startX+10, 20, 20, rl.Gold)
			rl.DrawText(selection.Name, startX+10, 50, 18, rl.RayWhite)
			rl.DrawText(selection.Type, startX+10, 75, 14, rl.Gray)
			rl.DrawText(selection.Details, startX+10, 110, 16, rl.RayWhite)

			// Deep Inspection additions
			if world.Alive(selection.Entity) {
				yOffset := int32(200)

				// Genetics
				genID := ecs.ComponentID[components.GenomeComponent](world)
				if world.Has(selection.Entity, genID) {
					gen := (*components.GenomeComponent)(world.Get(selection.Entity, genID))
					rl.DrawText("GENETICS", startX+10, yOffset, 16, rl.Gold)
					rl.DrawText(fmt.Sprintf("STR: %d  INT: %d", gen.Strength, gen.Intellect), startX+10, yOffset+20, 14, rl.RayWhite)
					rl.DrawText(fmt.Sprintf("BEA: %d  HLT: %d", gen.Beauty, gen.Health), startX+10, yOffset+40, 14, rl.RayWhite)
					yOffset += 70
				}

				// Affiliation
				affID := ecs.ComponentID[components.Affiliation](world)
				if world.Has(selection.Entity, affID) {
					aff := (*components.Affiliation)(world.Get(selection.Entity, affID))
					rl.DrawText("AFFILIATION", startX+10, yOffset, 16, rl.Gold)
					rl.DrawText(fmt.Sprintf("Fam: %d Clan: %d", aff.FamilyID, aff.ClanID), startX+10, yOffset+20, 14, rl.RayWhite)
					rl.DrawText(fmt.Sprintf("City: %d Ctry: %d", aff.CityID, aff.CountryID), startX+10, yOffset+40, 14, rl.RayWhite)
					yOffset += 70
				}

				// Beliefs
				beliefID := ecs.ComponentID[components.BeliefComponent](world)
				if world.Has(selection.Entity, beliefID) {
					beliefs := (*components.BeliefComponent)(world.Get(selection.Entity, beliefID))
					rl.DrawText(fmt.Sprintf("BELIEFS: %d", len(beliefs.Beliefs)), startX+10, yOffset, 16, rl.Gold)
					yOffset += 20
					for i, b := range beliefs.Beliefs {
						if i > 2 { break } // Max display 3
						rl.DrawText(fmt.Sprintf(" ID: %d W: %d", b.BeliefID, b.Weight), startX+10, yOffset, 14, rl.RayWhite)
						yOffset += 20
					}
				}
			}

			// Context Interaction Menu (if in Possession mode and close enough)
			if isPossessionMode {
				// Check distance to possessed entity
				possessedID := ecs.ComponentID[components.Possessed](world)
				posID := ecs.ComponentID[components.Position](world)
				identID := ecs.ComponentID[components.Identity](world)

				// Find player pos and identity
				var playerPos *components.Position
				var playerUID uint64
				playerEnt := ecs.Entity{}
				pQuery := world.Query(ecs.All(possessedID, posID, identID))
				for pQuery.Next() {
					playerPos = (*components.Position)(pQuery.Get(posID))
					pIdent := (*components.Identity)(pQuery.Get(identID))
					playerUID = pIdent.ID
					playerEnt = pQuery.Entity()
					break
				}

				if playerPos != nil && world.Has(selection.Entity, posID) && world.Has(selection.Entity, identID) {
					targetPos := (*components.Position)(world.Get(selection.Entity, posID))
					targetIdent := (*components.Identity)(world.Get(selection.Entity, identID))

					dx := playerPos.X - targetPos.X
					dy := playerPos.Y - targetPos.Y
					distSq := dx*dx + dy*dy

					if distSq < 10.0 && selection.Entity != playerEnt {
						// Render interaction menu over selection box
						menuY := int32(rl.GetScreenHeight()) - 200
						rl.DrawText("INTERACTIONS", startX+10, menuY, 16, rl.Gold)

						gossipBtn := rl.Rectangle{X: float32(startX + 10), Y: float32(menuY + 30), Width: 100, Height: 30}
						tradeBtn := rl.Rectangle{X: float32(startX + 120), Y: float32(menuY + 30), Width: 100, Height: 30}
						assaultBtn := rl.Rectangle{X: float32(startX + 10), Y: float32(menuY + 70), Width: 210, Height: 30}

						mousePos := rl.GetMousePosition()
						drawButton(gossipBtn, "Gossip", mousePos)
						drawButton(tradeBtn, "Trade", mousePos)
						drawButton(assaultBtn, "Assault (Crime)", mousePos)

						if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
							if rl.CheckCollisionPointRec(mousePos, gossipBtn) {
								// Interaction: Gossip
								// Add a generic secret to both via registry
								secReg := engine.GetSecretRegistry()
								secID := secReg.RegisterSecret(fmt.Sprintf("Player gossiped with UID %d", targetIdent.ID))

								// Player gains secret
								secCompID := ecs.ComponentID[components.SecretComponent](world)
								if world.Has(playerEnt, secCompID) {
									sc := (*components.SecretComponent)(world.Get(playerEnt, secCompID))
									sc.Secrets = append(sc.Secrets, components.Secret{OriginID: playerUID, SecretID: secID, Virality: 10, BeliefID: 0})
								}
								// Target gains secret
								if world.Has(selection.Entity, secCompID) {
									sc := (*components.SecretComponent)(world.Get(selection.Entity, secCompID))
									sc.Secrets = append(sc.Secrets, components.Secret{OriginID: playerUID, SecretID: secID, Virality: 10, BeliefID: 0})
								}
								fmt.Println("Gossiped!")
							} else if rl.CheckCollisionPointRec(mousePos, tradeBtn) {
								// Interaction: Trade (give 10 wealth for 1 food if possible)
								needsID := ecs.ComponentID[components.Needs](world)
								storageID := ecs.ComponentID[components.StorageComponent](world)
								if world.Has(playerEnt, needsID) && world.Has(selection.Entity, storageID) {
									pNeeds := (*components.Needs)(world.Get(playerEnt, needsID))
									tStore := (*components.StorageComponent)(world.Get(selection.Entity, storageID))
									if pNeeds.Wealth >= 10 && tStore.Food >= 1 {
										pNeeds.Wealth -= 10
										pNeeds.Food += 1
										tStore.Food -= 1
										fmt.Println("Traded 10 wealth for 1 food!")
									} else {
										fmt.Println("Not enough wealth or target has no food.")
									}
								}
							} else if rl.CheckCollisionPointRec(mousePos, assaultBtn) {
								// Interaction: Assault
								// Use Memory buffer directly (JusticeSystem reads this to apply CrimeMarker)
								memID := ecs.ComponentID[components.Memory](world)
								if world.Has(playerEnt, memID) {
									// Target records assault from player
									if world.Has(selection.Entity, memID) {
										tMem := (*components.Memory)(world.Get(selection.Entity, memID))

										// Add to head
										tMem.Events[tMem.Head] = components.MemoryEvent{
											TargetID: playerUID,
											TickStamp: tm.Ticks,
											InteractionType: components.InteractionAssault,
											LanguageID: 0,
											Value: -50,
										}
										tMem.Head = (tMem.Head + 1) % 50
									}

									// Player desperation + crime logging
									despID := ecs.ComponentID[components.DesperationComponent](world)
									if world.Has(playerEnt, despID) {
										d := (*components.DesperationComponent)(world.Get(playerEnt, despID))
										if d.Level < 100 { d.Level += 20 }
									}

									fmt.Println("Assault committed!")
								}
							}
						}
					}
				}
			}
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

// drawButton is a simple helper function to draw interactive UI buttons.
func drawButton(rect rl.Rectangle, text string, mousePos rl.Vector2) {
	color := rl.DarkGray
	if rl.CheckCollisionPointRec(mousePos, rect) {
		color = rl.Gray
	}
	rl.DrawRectangleRec(rect, color)
	rl.DrawRectangleLinesEx(rect, 2, rl.RayWhite)

	textWidth := rl.MeasureText(text, 20)
	textX := int32(rect.X + (rect.Width - float32(textWidth)) / 2)
	textY := int32(rect.Y + (rect.Height - 20) / 2)
	rl.DrawText(text, textX, textY, 20, rl.RayWhite)
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
