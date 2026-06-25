package engine

import (
	"os"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestSaveLoadWorld(t *testing.T) {
	// Setup db file
	dbPath := "test_save.db"
	os.Remove(dbPath)
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Setup world
	world := ecs.NewWorld()
	idID := ecs.ComponentID[components.Identity](&world)
	posID := ecs.ComponentID[components.Position](&world)

	// Add an entity
	ent := world.NewEntity(idID, posID)

	ident := (*components.Identity)(world.Get(ent, idID))
	ident.ID = 42
	ident.Name = "TestEntity"
	ident.BaseTraits = 100
	ident.Age = 35

	pos := (*components.Position)(world.Get(ent, posID))
	pos.X = 10.5
	pos.Y = 20.5

	// Phase: Playable Shell - add all new shell components to a single entity
	equipID := ecs.ComponentID[components.EquipmentComponent](&world)
	sanityID := ecs.ComponentID[components.SanityComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	jurisID := ecs.ComponentID[components.JurisdictionComponent](&world)
	structID := ecs.ComponentID[components.StructureComponent](&world)
	constrID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	demoID := ecs.ComponentID[components.DemographicsComponent](&world)
	workID := ecs.ComponentID[components.WorkbenchComponent](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	adminID := ecs.ComponentID[components.AdministrationMarker](&world)

	world.Add(ent, equipID, sanityID, treasuryID, marketID, legitID, loyaltyID,
		jurisID, structID, constrID, demoID, workID, cultureID,
		capitalID, countryID, adminID)

	eq := (*components.EquipmentComponent)(world.Get(ent, equipID))
	eq.Weapon.NameID = 7
	eq.Weapon.Prestige = 250
	eq.Weapon.History = []uint32{11, 22, 33}
	eq.Equipped = true

	san := (*components.SanityComponent)(world.Get(ent, sanityID))
	san.Stress = 12.5
	san.MaxStress = 100.0
	san.BreakState = components.BreakBerserk
	san.BreakCooldown = 9

	treasury := (*components.TreasuryComponent)(world.Get(ent, treasuryID))
	treasury.Wealth = 1234.5

	market := (*components.MarketComponent)(world.Get(ent, marketID))
	market.WoodPrice = 1.1
	market.StonePrice = 2.2
	market.IronPrice = 3.3
	market.FoodPrice = 4.4
	market.WageRate = 5.5

	legit := (*components.LegitimacyComponent)(world.Get(ent, legitID))
	legit.Score = 88

	loyalty := (*components.LoyaltyComponent)(world.Get(ent, loyaltyID))
	loyalty.Value = 73

	juris := (*components.JurisdictionComponent)(world.Get(ent, jurisID))
	juris.RadiusSquared = 400.0
	juris.IllegalActionIDs = 1 << components.InteractionTheft
	juris.Corruption = 17
	juris.BannedSecretID = 99
	juris.Trauma = 5

	st := (*components.StructureComponent)(world.Get(ent, structID))
	st.StructureType = uint32(components.StructureHouse)
	st.Integrity = 77.7

	cs := (*components.ConstructionSiteComponent)(world.Get(ent, constrID))
	cs.WoodRequired = 50
	cs.WoodGathered = 20
	cs.StoneRequired = 30
	cs.StoneGathered = 10
	cs.Progress = 5
	cs.MaxProgress = 100
	cs.BuilderID = 4242

	demo := (*components.DemographicsComponent)(world.Get(ent, demoID))
	demo.PeakPopulation = 500
	demo.LaborCrisisActive = true

	work := (*components.WorkbenchComponent)(world.Get(ent, workID))
	work.EmployerID = 555
	work.X = 64.0
	work.Y = 128.0

	culture := (*components.CultureComponent)(world.Get(ent, cultureID))
	culture.DialectTickStamp = 9000
	culture.ForeignInteractionTicks = 42
	culture.LanguageID = 3
	culture.ForeignLanguageID = 4

	ctry := (*components.CountryComponent)(world.Get(ent, countryID))
	ctry.StandardCurrencyID = 12
	ctry.Debasement = 0.25

	// Setup TM and Grid wrappers
	tm := &TickManager{World: &world, Ticks: 500}
	grid := NewMapGrid(10, 10)

	// Save
	err = SaveWorld(tm, grid, byte(1), db)
	if err != nil {
		t.Fatalf("Failed to save world: %v", err)
	}

	// Load into a new world
	newWorld := ecs.NewWorld()
	newTm := &TickManager{World: &newWorld}

	newIdID := ecs.ComponentID[components.Identity](&newWorld)
	newPosID := ecs.ComponentID[components.Position](&newWorld)
	_ = newIdID // Ensure registered
	_ = newPosID

	newEquipID := ecs.ComponentID[components.EquipmentComponent](&newWorld)
	newSanityID := ecs.ComponentID[components.SanityComponent](&newWorld)
	newTreasuryID := ecs.ComponentID[components.TreasuryComponent](&newWorld)
	newMarketID := ecs.ComponentID[components.MarketComponent](&newWorld)
	newLegitID := ecs.ComponentID[components.LegitimacyComponent](&newWorld)
	newLoyaltyID := ecs.ComponentID[components.LoyaltyComponent](&newWorld)
	newJurisID := ecs.ComponentID[components.JurisdictionComponent](&newWorld)
	newStructID := ecs.ComponentID[components.StructureComponent](&newWorld)
	newConstrID := ecs.ComponentID[components.ConstructionSiteComponent](&newWorld)
	newDemoID := ecs.ComponentID[components.DemographicsComponent](&newWorld)
	newWorkID := ecs.ComponentID[components.WorkbenchComponent](&newWorld)
	newCultureID := ecs.ComponentID[components.CultureComponent](&newWorld)
	newCapitalID := ecs.ComponentID[components.CapitalComponent](&newWorld)
	newCountryID := ecs.ComponentID[components.CountryComponent](&newWorld)
	newAdminID := ecs.ComponentID[components.AdministrationMarker](&newWorld)

	err = LoadWorld(newTm, db)
	if err != nil {
		t.Fatalf("Failed to load world: %v", err)
	}

	if newTm.Ticks != 500 {
		t.Errorf("Expected 500 ticks, got %d", newTm.Ticks)
	}

	// Verify
	query := newWorld.Query(ecs.All(newIdID))
	count := 0
	for query.Next() {
		count++
		newEnt := query.Entity()
		newIdent := (*components.Identity)(newWorld.Get(newEnt, newIdID))

		if newIdent.ID != 42 || newIdent.Name != "TestEntity" || newIdent.BaseTraits != 100 || newIdent.Age != 35 {
			t.Errorf("Loaded identity mismatch: %+v", newIdent)
		}

		if newWorld.Has(newEnt, newPosID) {
			newPos := (*components.Position)(newWorld.Get(newEnt, newPosID))
			if newPos.X != 10.5 || newPos.Y != 20.5 {
				t.Errorf("Loaded position mismatch: %+v", newPos)
			}
		} else {
			t.Errorf("Expected entity to have position component")
		}

		// Equipment
		if !newWorld.Has(newEnt, newEquipID) {
			t.Errorf("Expected entity to have equipment component")
		} else {
			eq := (*components.EquipmentComponent)(newWorld.Get(newEnt, newEquipID))
			if eq.Weapon.NameID != 7 || eq.Weapon.Prestige != 250 || !eq.Equipped {
				t.Errorf("Loaded equipment mismatch: %+v", eq)
			}
			if len(eq.Weapon.History) != 3 || eq.Weapon.History[0] != 11 || eq.Weapon.History[1] != 22 || eq.Weapon.History[2] != 33 {
				t.Errorf("Loaded equipment history mismatch: %+v", eq.Weapon.History)
			}
		}

		// Sanity
		if !newWorld.Has(newEnt, newSanityID) {
			t.Errorf("Expected entity to have sanity component")
		} else {
			san := (*components.SanityComponent)(newWorld.Get(newEnt, newSanityID))
			if san.Stress != 12.5 || san.MaxStress != 100.0 || san.BreakState != components.BreakBerserk || san.BreakCooldown != 9 {
				t.Errorf("Loaded sanity mismatch: %+v", san)
			}
		}

		// Treasury
		if !newWorld.Has(newEnt, newTreasuryID) {
			t.Errorf("Expected entity to have treasury component")
		} else {
			tr := (*components.TreasuryComponent)(newWorld.Get(newEnt, newTreasuryID))
			if tr.Wealth != 1234.5 {
				t.Errorf("Loaded treasury mismatch: %+v", tr)
			}
		}

		// Market
		if !newWorld.Has(newEnt, newMarketID) {
			t.Errorf("Expected entity to have market component")
		} else {
			mk := (*components.MarketComponent)(newWorld.Get(newEnt, newMarketID))
			if mk.WoodPrice != 1.1 || mk.StonePrice != 2.2 || mk.IronPrice != 3.3 || mk.FoodPrice != 4.4 || mk.WageRate != 5.5 {
				t.Errorf("Loaded market mismatch: %+v", mk)
			}
		}

		// Legitimacy
		if !newWorld.Has(newEnt, newLegitID) {
			t.Errorf("Expected entity to have legitimacy component")
		} else {
			l := (*components.LegitimacyComponent)(newWorld.Get(newEnt, newLegitID))
			if l.Score != 88 {
				t.Errorf("Loaded legitimacy mismatch: %+v", l)
			}
		}

		// Loyalty
		if !newWorld.Has(newEnt, newLoyaltyID) {
			t.Errorf("Expected entity to have loyalty component")
		} else {
			l := (*components.LoyaltyComponent)(newWorld.Get(newEnt, newLoyaltyID))
			if l.Value != 73 {
				t.Errorf("Loaded loyalty mismatch: %+v", l)
			}
		}

		// Jurisdiction
		if !newWorld.Has(newEnt, newJurisID) {
			t.Errorf("Expected entity to have jurisdiction component")
		} else {
			j := (*components.JurisdictionComponent)(newWorld.Get(newEnt, newJurisID))
			if j.RadiusSquared != 400.0 || j.IllegalActionIDs != (1<<components.InteractionTheft) || j.Corruption != 17 || j.BannedSecretID != 99 || j.Trauma != 5 {
				t.Errorf("Loaded jurisdiction mismatch: %+v", j)
			}
		}

		// Structure
		if !newWorld.Has(newEnt, newStructID) {
			t.Errorf("Expected entity to have structure component")
		} else {
			st := (*components.StructureComponent)(newWorld.Get(newEnt, newStructID))
			if st.StructureType != uint32(components.StructureHouse) || st.Integrity != 77.7 {
				t.Errorf("Loaded structure mismatch: %+v", st)
			}
		}

		// ConstructionSite
		if !newWorld.Has(newEnt, newConstrID) {
			t.Errorf("Expected entity to have construction site component")
		} else {
			cs := (*components.ConstructionSiteComponent)(newWorld.Get(newEnt, newConstrID))
			if cs.WoodRequired != 50 || cs.WoodGathered != 20 || cs.StoneRequired != 30 || cs.StoneGathered != 10 || cs.Progress != 5 || cs.MaxProgress != 100 || cs.BuilderID != 4242 {
				t.Errorf("Loaded construction site mismatch: %+v", cs)
			}
		}

		// Demographics
		if !newWorld.Has(newEnt, newDemoID) {
			t.Errorf("Expected entity to have demographics component")
		} else {
			dm := (*components.DemographicsComponent)(newWorld.Get(newEnt, newDemoID))
			if dm.PeakPopulation != 500 || !dm.LaborCrisisActive {
				t.Errorf("Loaded demographics mismatch: %+v", dm)
			}
		}

		// Workbench
		if !newWorld.Has(newEnt, newWorkID) {
			t.Errorf("Expected entity to have workbench component")
		} else {
			wb := (*components.WorkbenchComponent)(newWorld.Get(newEnt, newWorkID))
			if wb.EmployerID != 555 || wb.X != 64.0 || wb.Y != 128.0 {
				t.Errorf("Loaded workbench mismatch: %+v", wb)
			}
		}

		// Culture
		if !newWorld.Has(newEnt, newCultureID) {
			t.Errorf("Expected entity to have culture component")
		} else {
			cul := (*components.CultureComponent)(newWorld.Get(newEnt, newCultureID))
			if cul.DialectTickStamp != 9000 || cul.ForeignInteractionTicks != 42 || cul.LanguageID != 3 || cul.ForeignLanguageID != 4 {
				t.Errorf("Loaded culture mismatch: %+v", cul)
			}
		}

		// Extra tags
		if !newWorld.Has(newEnt, newCapitalID) {
			t.Errorf("Expected entity to have capital tag")
		}
		if !newWorld.Has(newEnt, newAdminID) {
			t.Errorf("Expected entity to have administration marker")
		}
		if !newWorld.Has(newEnt, newCountryID) {
			t.Errorf("Expected entity to have country component")
		} else {
			ctry := (*components.CountryComponent)(newWorld.Get(newEnt, newCountryID))
			if ctry.StandardCurrencyID != 12 || ctry.Debasement != 0.25 {
				t.Errorf("Loaded country mismatch: %+v", ctry)
			}
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 entity loaded, got %d", count)
	}
}
