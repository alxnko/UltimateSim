package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/ALXNKO/UltimateSim/internal/ui"
)

// Phase 01.4: Hardware Affinity & Rendering Bridging
// Phase 01.6: Telemetry & Profiling
// Phase 11.1: Raylib rendering architecture

func BuildSimulation(gridWidth, gridHeight int, seedVal byte, status *render.LoadingStatus) {
	update := func(progress float32, msg string) {
		status.Mutex.Lock()
		status.Progress = progress
		status.Message = msg
		status.Mutex.Unlock()
	}

	update(0.1, "Initializing Map Grid...")
	// Phase 02: Map Generation
	// Phase 02.1: Map Scaling support configurable grid sizes and scales
	grid := engine.NewMapGrid(gridWidth, gridHeight)

	update(0.2, "Initializing RNG & Seed...")
	seed := [32]byte{seedVal, seedVal + 1, seedVal + 2, seedVal + 3, seedVal + 4} // Deterministic seed based on input
	engine.InitializeRNG(seed)

	update(0.3, "Generating Tectonic & Biome Data...")
	engine.GenerateMap(grid, seed)

	update(0.5, "Building Oceanic Nav Mesh...")
	// Phase 17.2: Build Secondary Nav Mesh for Oceanic Pathfinding
	engine.BuildOceanicNavMesh(grid)

	update(0.6, "Starting Pathfinding Workers...")
	// Phase 04.2: Async Path Queue Pool
	pathQueue := engine.NewPathRequestQueue(1000, runtime.NumCPU())
	pathQueue.StartWorkers()

	update(0.7, "Initializing Social Hook Graph...")
	// Phase 06.3: Sparse Hook Graph
	hookGraph := engine.NewSparseHookGraph()

	update(0.8, "Assembling ECS Engine & Systems...")
	// Phase 01.3: Initialize the TickManager with 60 TPS
	tickManager := engine.NewTickManager(60)
	world := tickManager.World

	// Shell Phase: cross-tick player bridges shared with the UI layer.
	bridge := &systems.InputBridge{}
	playerEvents := &engine.PlayerEvents{}

	// Phase 13.4: The Seasonal Pulse
	calendar := engine.NewCalendar()
	tickManager.AddSystem(systems.NewCalendarSystem(calendar), engine.PhaseInput)

	// Phase 19.2: Ecological Drift
	tickManager.AddSystem(systems.NewGlobalWeatherSystem(world, grid), engine.PhaseInput)

	// Phase 09.2: Dynamic Attrition
	tickManager.AddSystem(systems.NewSpoilageSystem(), engine.PhaseResolution)
	// Phase 37.1: The Quarantine Engine
	tickManager.AddSystem(systems.NewQuarantineSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewNaturalDisasterSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewExposureSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRustSystem(), engine.PhaseResolution)
	// --- PHASE: AI ---
	tickManager.AddSystem(systems.NewDesperationSystem(world), engine.PhaseAI)
	tickManager.AddSystem(systems.NewBanditrySystem(world), engine.PhaseAI)
	tickManager.AddSystem(systems.NewCourierInterceptionSystem(world), engine.PhaseAI)
	tickManager.AddSystem(systems.NewWanderSystem(world, grid, pathQueue), engine.PhaseAI)
	// Grand Strategy P5 L1: employed citizens live visible daily routines;
	// WanderSystem keeps only the unemployed and the wilderness.
	tickManager.AddSystem(systems.NewDailyRoutineSystem(world, grid, calendar), engine.PhaseAI)
	tickManager.AddSystem(systems.NewNavalRoutingSystem(world, grid, pathQueue, calendar), engine.PhaseAI)

	// --- PHASE: MOVEMENT ---
	tickManager.AddSystem(systems.NewMovementSystem(world, grid, calendar), engine.PhaseMovement)
	tickManager.AddSystem(systems.NewInfrastructureWearSystem(grid), engine.PhaseMovement)

	// --- PHASE: RESOLUTION ---
	tickManager.AddSystem(systems.NewMetabolismSystem(world, calendar, tickManager), engine.PhaseResolution)
	// Phase 31.5: The Winter Heating Engine
	tickManager.AddSystem(systems.NewWinterHeatingSystem(world, calendar), engine.PhaseResolution)
	// Phase 17.3: Maritime Attrition & Piracy
	tickManager.AddSystem(systems.NewStormSystem(grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewNavalPiracySystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewBirthSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewDiseaseVectorSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCaravanSpawnerSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCareerChangeSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCityBinderSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewSettlementRuleSystem(grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPriceDiscoverySystem(), engine.PhaseResolution)
	// Grand Strategy P2.6: trade-route goods flow + route income.
	tickManager.AddSystem(systems.NewTradeRouteSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRuinTransformationSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewAdministrativeDecaySystem(), engine.PhaseResolution)

	// Phase 43: Organic Administration Engine
	tickManager.AddSystem(systems.NewLeadershipEmergenceSystem(hookGraph), engine.PhaseResolution)

	// Phase 16.1 & 42 & Grand Strategy P2.6: Taxation + Tax Evasion + Tax
	// Policy rate control. The wrapper owns and drives the base system —
	// registering both would double-collect.
	tickManager.AddSystem(systems.NewTaxPolicySystem(systems.NewTaxationSystem(world, hookGraph)), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewVassalSafetyValveSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 16.4: Administrative Reach & Friction
	tickManager.AddSystem(systems.NewAdministrativeFractureSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewLendingSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewDebtDefaultSystem(hookGraph), engine.PhaseResolution)

	// Phase 61: The Biological Sabotage Engine
	tickManager.AddSystem(systems.NewBiologicalSabotageSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 28.1: The Vassal Rebellion Engine
	tickManager.AddSystem(systems.NewVassalRebellionSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 33.1: Cultural Friction & Ideological Secession Engine
	tickManager.AddSystem(systems.NewCulturalFrictionSystem(), engine.PhaseResolution)

	// Phase 29.1: Geopolitical Resource Wars
	tickManager.AddSystem(systems.NewResourceWarSystem(world, hookGraph), engine.PhaseResolution)

	// Grand Strategy Phase: opinion/alliance/war-score diplomacy ledger,
	// synced two-way with the macro WarTracker model. Registered right after
	// ResourceWarSystem so macro declarations sync on the same cadence.
	tickManager.AddSystem(systems.NewDiplomacySystem(world), engine.PhaseResolution)

	// Register Gossip
	tickManager.AddSystem(systems.NewInformationTradeSystem(world, hookGraph), engine.PhaseResolution)

	tickManager.AddSystem(systems.NewGossipDistributionSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 39.1 & 39.2: The Epistemological Engine
	tickManager.AddSystem(systems.NewScholarSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewLedgerDiscoverySystem(world), engine.PhaseResolution)

	// Register Language Drift
	tickManager.AddSystem(systems.NewLanguageDriftSystem(world), engine.PhaseResolution)

	// Phase 30.1: Ideological Economy
	tickManager.AddSystem(systems.NewTitheSystem(world), engine.PhaseResolution)

	// Phase 18: Justice Engine
	tickManager.AddSystem(systems.NewJusticeSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 23 & 64: Blood Feud & Physical Combat Engine
	tickManager.AddSystem(systems.NewBloodFeudSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCombatSystem(world), engine.PhaseResolution)

	// Phase 41: The Ostracization Engine
	tickManager.AddSystem(systems.NewOstracizationSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 27.1: The Military Revolt Engine
	tickManager.AddSystem(systems.NewMilitaryRevoltSystem(world, hookGraph), engine.PhaseResolution)

	// Phase 24.1: The Labor Union Engine
	tickManager.AddSystem(systems.NewLaborUnionSystem(world, hookGraph), engine.PhaseResolution)

	// --- Previously-dormant simulation systems (woken for full depth) ---
	tickManager.AddSystem(systems.NewAgricultureSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewSanitationSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewMedicalSystem(world, pathQueue, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewMentalBreakSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewParasiteSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewWorkplaceSystem(pathQueue), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewBlackMarketSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewWarEconomySystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewAdministrationSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewLegitimacySystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPoliticalCoupSystem(hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewClassWarfareSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewXenophobiaSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewScapegoatSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPropagandaSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewTraumaticTraditionsSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPreacherSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewHolyWarSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewEchoChamberSystem(grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewJealousyVulnerabilitySystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewConscriptionSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewMercenarySystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewMaritimeLaborSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewMaritimeMigrationSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewNavalSpawningSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPenalLaborSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewLaborCrisisSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCastingSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRuinResettlementSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewRefugeeSystem(world, pathQueue), engine.PhaseResolution)
	// New evolution engines merged from main.
	tickManager.AddSystem(systems.NewMiningSystem(world, grid), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewUnderminingSystem(world), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCannibalismSystem(world, pathQueue), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewForgerySystem(world, hookGraph), engine.PhaseResolution)

	// --- Shell Phase: physical construction, crafting and player systems ---
	tickManager.AddSystem(systems.NewConstructionSystem(world, pathQueue), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCraftingSystem(world, pathQueue), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewStructureEffectSystem(), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewAmbitionSystem(hookGraph), engine.PhaseResolution)
	// Grand Strategy P2.4/P2.5: intrigue plots + council bonuses.
	tickManager.AddSystem(systems.NewPlotSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewCouncilSystem(world), engine.PhaseResolution)
	// Grand Strategy G1/G2/G6: interactive event director (drama engine).
	tickManager.AddSystem(systems.NewEventDirectorSystem(world, hookGraph), engine.PhaseResolution)
	tickManager.AddSystem(systems.NewPlayerOrderSystem(hookGraph, bridge), engine.PhaseAI)

	// --- PHASE: CLEANUP ---
	deathSys := systems.NewDeathSystem(world, hookGraph)
	deathSys.SetPlayerEvents(playerEvents) // Shell Phase: surface possessed death
	tickManager.AddSystem(deathSys, engine.PhaseCleanup)
	tickManager.AddSystem(systems.NewAgingSystem(world, tickManager), engine.PhaseResolution)

	// Phase 03.2: Genesis Spawner (Runs once at tick 0)
	tickManager.AddSystem(systems.NewNPCSpawnerSystem(world, grid), engine.PhaseCleanup)

	// Grand Strategy Phase: plant an already-political world — cities,
	// countries, capitals, rulers, employed families — before the first tick,
	// the way a grand-strategy start date works. The wilderness spawner above
	// still adds organic wanderers on tick 1.
	update(0.9, "Founding Civilizations...")
	systems.SeedCivilization(world, grid, hookGraph, systems.DefaultGenesis())

	update(1.0, "Engine Assembly Complete.")

	status.Mutex.Lock()
	status.TM = tickManager
	status.Grid = grid
	status.HookGraph = hookGraph
	status.Bridge = bridge
	status.Events = playerEvents
	status.Seed = seedVal
	status.Done = true
	status.Mutex.Unlock()
}

type Game struct {
	sm *ui.StateManager
}

func (g *Game) Update() error {
	if ui.QuitRequested() {
		return ebiten.Termination
	}
	return g.sm.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.sm.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Grand Strategy Phase: render at the real window size (crisp text at any
	// resolution). UI layouts read screen bounds each frame.
	if outsideWidth < 960 {
		outsideWidth = 960
	}
	if outsideHeight < 540 {
		outsideHeight = 540
	}
	return outsideWidth, outsideHeight
}

func main() {
	// Phase 01.6: Telemetry & Profiling
	// Use flags to control pprof for security
	pprofEnabled := flag.Bool("pprof", false, "Enable pprof profiling server")
	pprofPort := flag.Int("pprof-port", 6060, "Port for pprof profiling server")
	flag.Parse()

	if *pprofEnabled {
		// Boot net/http/pprof instance on localhost
		go func() {
			addr := fmt.Sprintf("localhost:%d", *pprofPort)
			log.Printf("Starting pprof server on %s\n", addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Fatalf("pprof server failed: %v", err)
			}
		}()
	}

	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Boundless Sovereigns")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	sm := ui.NewStateManager()
	
	status := &render.LoadingStatus{}
	
	// Create Playing State first (but don't push it yet)
	playingState := ui.NewStatePlaying(status)

	// Create Main Menu
	menuState := ui.NewStateMainMenu(func() {
		// When "Start" is clicked, push the playing state
		sm.Switch(playingState)
	})

	// Start the async engine build
	go func() {
		BuildSimulation(1024, 1024, 42, status)
		// Register the player input system once the world + bridge exist.
		status.Mutex.Lock()
		inputSys := systems.NewPlayerInputSystem(status.Bridge)
		inputSys.Initialize(status.TM.World)
		status.TM.AddSystem(inputSys, engine.PhaseInput)

		directorSys := systems.NewPlayerDirectorSystem(status.HookGraph)
		directorSys.Initialize(status.TM.World)
		status.TM.AddSystem(directorSys, engine.PhaseResolution)
		status.Mutex.Unlock()
	}()

	sm.Push(menuState)

	game := &Game{sm: sm}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
