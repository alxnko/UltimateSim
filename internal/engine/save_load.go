package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	_ "modernc.org/sqlite"
)

// InitDB initializes SQLite schemas for persistence.
func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Create tables for core components
	schema := `
	CREATE TABLE IF NOT EXISTS entities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uid INTEGER UNIQUE
	);

	CREATE TABLE IF NOT EXISTS identity (
		uid INTEGER PRIMARY KEY,
		name TEXT,
		basetraits INTEGER,
		age INTEGER
	);

	CREATE TABLE IF NOT EXISTS position (
		uid INTEGER PRIMARY KEY,
		x REAL,
		y REAL
	);

	CREATE TABLE IF NOT EXISTS needs (
		uid INTEGER PRIMARY KEY,
		food REAL,
		rest REAL,
		safety REAL,
		wealth REAL
	);

	CREATE TABLE IF NOT EXISTS affiliation (
		uid INTEGER PRIMARY KEY,
		family_id INTEGER,
		clan_id INTEGER,
		city_id INTEGER,
		country_id INTEGER
	);

	CREATE TABLE IF NOT EXISTS tags (
		uid INTEGER PRIMARY KEY,
		is_village BOOLEAN,
		is_npc BOOLEAN,
		is_possessed BOOLEAN
	);

	CREATE TABLE IF NOT EXISTS storage (
		uid INTEGER PRIMARY KEY,
		wood INTEGER,
		stone INTEGER,
		iron INTEGER,
		food INTEGER
	);

	CREATE TABLE IF NOT EXISTS velocity (
		uid INTEGER PRIMARY KEY,
		x REAL,
		y REAL
	);

	CREATE TABLE IF NOT EXISTS job (
		uid INTEGER PRIMARY KEY,
		job_id INTEGER,
		employer_id INTEGER
	);

	CREATE TABLE IF NOT EXISTS memory (
		uid INTEGER PRIMARY KEY,
		events_json TEXT,
		head INTEGER
	);

	CREATE TABLE IF NOT EXISTS beliefs (
		uid INTEGER PRIMARY KEY,
		beliefs_json TEXT
	);

	CREATE TABLE IF NOT EXISTS genome (
		uid INTEGER PRIMARY KEY,
		str INTEGER,
		bea INTEGER,
		hlt INTEGER,
		itl INTEGER,
		dom INTEGER,
		rec INTEGER
	);

	CREATE TABLE IF NOT EXISTS vitals (
		uid INTEGER PRIMARY KEY,
		stamina REAL,
		blood REAL,
		pain REAL,
		consciousness REAL
	);

	CREATE TABLE IF NOT EXISTS population (
		uid INTEGER PRIMARY KEY,
		count INTEGER,
		citizens_json TEXT
	);

	CREATE TABLE IF NOT EXISTS desperation (
		uid INTEGER PRIMARY KEY,
		level INTEGER
	);

	CREATE TABLE IF NOT EXISTS secrets (
		uid INTEGER PRIMARY KEY,
		secrets_json TEXT
	);

	CREATE TABLE IF NOT EXISTS game_state (
		id INTEGER PRIMARY KEY,
		ticks INTEGER,
		grid_width INTEGER,
		grid_height INTEGER,
		seed_val INTEGER
	);

	CREATE TABLE IF NOT EXISTS equipment (
		uid INTEGER PRIMARY KEY,
		weapon_nameid INTEGER,
		weapon_prestige INTEGER,
		weapon_history_json TEXT,
		equipped BOOLEAN
	);

	CREATE TABLE IF NOT EXISTS sanity (
		uid INTEGER PRIMARY KEY,
		stress REAL,
		max_stress REAL,
		break_state INTEGER,
		break_cooldown INTEGER
	);

	CREATE TABLE IF NOT EXISTS treasury (
		uid INTEGER PRIMARY KEY,
		wealth REAL
	);

	CREATE TABLE IF NOT EXISTS market (
		uid INTEGER PRIMARY KEY,
		wood_price REAL,
		stone_price REAL,
		iron_price REAL,
		food_price REAL,
		wage_rate REAL
	);

	CREATE TABLE IF NOT EXISTS legitimacy (
		uid INTEGER PRIMARY KEY,
		score INTEGER
	);

	CREATE TABLE IF NOT EXISTS loyalty (
		uid INTEGER PRIMARY KEY,
		value INTEGER
	);

	CREATE TABLE IF NOT EXISTS jurisdiction (
		uid INTEGER PRIMARY KEY,
		radius_sq REAL,
		illegal_action_ids INTEGER,
		corruption INTEGER,
		banned_secret_id INTEGER,
		trauma INTEGER
	);

	CREATE TABLE IF NOT EXISTS structure (
		uid INTEGER PRIMARY KEY,
		structure_type INTEGER,
		integrity REAL,
		data_a INTEGER,
		owner_id INTEGER
	);

	CREATE TABLE IF NOT EXISTS construction_site (
		uid INTEGER PRIMARY KEY,
		wood_req INTEGER,
		wood_gathered INTEGER,
		stone_req INTEGER,
		stone_gathered INTEGER,
		progress INTEGER,
		max_progress INTEGER,
		builder_id INTEGER,
		site_type INTEGER
	);

	CREATE TABLE IF NOT EXISTS ambitions (
		uid INTEGER PRIMARY KEY,
		ambitions_json TEXT,
		offers_json TEXT,
		built_count INTEGER,
		family_base INTEGER
	);

	CREATE TABLE IF NOT EXISTS demographics (
		uid INTEGER PRIMARY KEY,
		peak_population INTEGER,
		labor_crisis BOOLEAN
	);

	CREATE TABLE IF NOT EXISTS dynasty (
		uid INTEGER PRIMARY KEY,
		spouse_id INTEGER,
		children INTEGER,
		married BOOLEAN
	);

	CREATE TABLE IF NOT EXISTS plot (
		uid INTEGER PRIMARY KEY,
		target_id INTEGER,
		start_tick INTEGER,
		progress INTEGER,
		power INTEGER,
		kind INTEGER,
		exposed BOOLEAN
	);

	CREATE TABLE IF NOT EXISTS council (
		uid INTEGER PRIMARY KEY,
		steward INTEGER,
		marshal INTEGER,
		diplomat INTEGER,
		spymaster INTEGER
	);

	CREATE TABLE IF NOT EXISTS diplomacy (
		uid INTEGER PRIMARY KEY,
		relations_json TEXT
	);

	CREATE TABLE IF NOT EXISTS tax_policy (
		uid INTEGER PRIMARY KEY,
		rate INTEGER
	);

	CREATE TABLE IF NOT EXISTS trade_routes (
		from_city INTEGER,
		to_city INTEGER,
		volume INTEGER
	);

	CREATE TABLE IF NOT EXISTS workbench (
		uid INTEGER PRIMARY KEY,
		employer_id INTEGER,
		x REAL,
		y REAL
	);

	CREATE TABLE IF NOT EXISTS culture (
		uid INTEGER PRIMARY KEY,
		dialect_tick INTEGER,
		foreign_ticks INTEGER,
		language_id INTEGER,
		foreign_language_id INTEGER
	);

	CREATE TABLE IF NOT EXISTS extra_tags (
		uid INTEGER PRIMARY KEY,
		is_capital BOOLEAN,
		is_country BOOLEAN,
		is_admin BOOLEAN,
		country_currency_id INTEGER,
		country_debasement REAL
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return db, nil
}

var tableDeleteQueries = map[string]string{
	"entities":          "DELETE FROM entities",
	"identity":          "DELETE FROM identity",
	"position":          "DELETE FROM position",
	"needs":             "DELETE FROM needs",
	"affiliation":       "DELETE FROM affiliation",
	"tags":              "DELETE FROM tags",
	"storage":           "DELETE FROM storage",
	"velocity":          "DELETE FROM velocity",
	"job":               "DELETE FROM job",
	"memory":            "DELETE FROM memory",
	"beliefs":           "DELETE FROM beliefs",
	"genome":            "DELETE FROM genome",
	"vitals":            "DELETE FROM vitals",
	"population":        "DELETE FROM population",
	"desperation":       "DELETE FROM desperation",
	"secrets":           "DELETE FROM secrets",
	"equipment":         "DELETE FROM equipment",
	"sanity":            "DELETE FROM sanity",
	"treasury":          "DELETE FROM treasury",
	"market":            "DELETE FROM market",
	"legitimacy":        "DELETE FROM legitimacy",
	"loyalty":           "DELETE FROM loyalty",
	"jurisdiction":      "DELETE FROM jurisdiction",
	"structure":         "DELETE FROM structure",
	"construction_site": "DELETE FROM construction_site",
	"demographics":      "DELETE FROM demographics",
	"workbench":         "DELETE FROM workbench",
	"culture":           "DELETE FROM culture",
	"extra_tags":        "DELETE FROM extra_tags",
	"ambitions":         "DELETE FROM ambitions",
}

// SaveWorld serializes the core ECS state into SQLite.
func SaveWorld(tm *TickManager, mapGrid *MapGrid, seedVal byte, db *sql.DB) error {
	world := tm.World

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Save game_state
	stmtState, err := tx.Prepare("INSERT OR REPLACE INTO game_state (id, ticks, grid_width, grid_height, seed_val) VALUES (1, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtState.Close()
	if _, err := stmtState.Exec(tm.Ticks, mapGrid.Width, mapGrid.Height, int(seedVal)); err != nil {
		return err
	}

	// Clear out old entity rows to prevent resurrecting dead entities
	tables := []string{"entities", "identity", "position", "needs", "affiliation", "tags", "storage", "velocity", "job", "memory", "beliefs", "genome", "vitals", "population", "desperation", "secrets", "equipment", "sanity", "treasury", "market", "legitimacy", "loyalty", "jurisdiction", "structure", "construction_site", "demographics", "workbench", "culture", "extra_tags", "ambitions"}
	for _, table := range tables {
		query, ok := tableDeleteQueries[table]
		if !ok {
			return fmt.Errorf("unauthorized table delete attempt: %s", table)
		}
		if _, err := tx.Exec(query); err != nil {
			return err
		}
	}

	// Prepare statements
	stmtEnt, err := tx.Prepare("INSERT OR REPLACE INTO entities (uid) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmtEnt.Close()
	stmtId, err := tx.Prepare("INSERT OR REPLACE INTO identity (uid, name, basetraits, age) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtId.Close()
	stmtPos, err := tx.Prepare("INSERT OR REPLACE INTO position (uid, x, y) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtPos.Close()
	stmtNeeds, err := tx.Prepare("INSERT OR REPLACE INTO needs (uid, food, rest, safety, wealth) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtNeeds.Close()
	stmtAff, err := tx.Prepare("INSERT OR REPLACE INTO affiliation (uid, family_id, clan_id, city_id, country_id) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtAff.Close()
	stmtTags, err := tx.Prepare("INSERT OR REPLACE INTO tags (uid, is_village, is_npc, is_possessed) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtTags.Close()
	stmtStorage, err := tx.Prepare("INSERT OR REPLACE INTO storage (uid, wood, stone, iron, food) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtStorage.Close()
	stmtVel, err := tx.Prepare("INSERT OR REPLACE INTO velocity (uid, x, y) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtVel.Close()
	stmtJob, err := tx.Prepare("INSERT OR REPLACE INTO job (uid, job_id, employer_id) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtJob.Close()
	stmtMem, err := tx.Prepare("INSERT OR REPLACE INTO memory (uid, events_json, head) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtMem.Close()
	stmtBeliefs, err := tx.Prepare("INSERT OR REPLACE INTO beliefs (uid, beliefs_json) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtBeliefs.Close()
	stmtGen, err := tx.Prepare("INSERT OR REPLACE INTO genome (uid, str, bea, hlt, itl, dom, rec) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtGen.Close()
	stmtVitals, err := tx.Prepare("INSERT OR REPLACE INTO vitals (uid, stamina, blood, pain, consciousness) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtVitals.Close()
	stmtPop, err := tx.Prepare("INSERT OR REPLACE INTO population (uid, count, citizens_json) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtPop.Close()
	stmtDesp, err := tx.Prepare("INSERT OR REPLACE INTO desperation (uid, level) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtDesp.Close()
	stmtSec, err := tx.Prepare("INSERT OR REPLACE INTO secrets (uid, secrets_json) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtSec.Close()
	stmtEquip, err := tx.Prepare("INSERT OR REPLACE INTO equipment (uid, weapon_nameid, weapon_prestige, weapon_history_json, equipped) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtEquip.Close()
	stmtSanity, err := tx.Prepare("INSERT OR REPLACE INTO sanity (uid, stress, max_stress, break_state, break_cooldown) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtSanity.Close()
	stmtTreasury, err := tx.Prepare("INSERT OR REPLACE INTO treasury (uid, wealth) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtTreasury.Close()
	stmtMarket, err := tx.Prepare("INSERT OR REPLACE INTO market (uid, wood_price, stone_price, iron_price, food_price, wage_rate) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtMarket.Close()
	stmtLegit, err := tx.Prepare("INSERT OR REPLACE INTO legitimacy (uid, score) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtLegit.Close()
	stmtLoyalty, err := tx.Prepare("INSERT OR REPLACE INTO loyalty (uid, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtLoyalty.Close()
	stmtJuris, err := tx.Prepare("INSERT OR REPLACE INTO jurisdiction (uid, radius_sq, illegal_action_ids, corruption, banned_secret_id, trauma) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtJuris.Close()
	stmtStruct, err := tx.Prepare("INSERT OR REPLACE INTO structure (uid, structure_type, integrity, data_a, owner_id) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtStruct.Close()
	stmtConstr, err := tx.Prepare("INSERT OR REPLACE INTO construction_site (uid, wood_req, wood_gathered, stone_req, stone_gathered, progress, max_progress, builder_id, site_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtConstr.Close()
	stmtDemo, err := tx.Prepare("INSERT OR REPLACE INTO demographics (uid, peak_population, labor_crisis) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtDemo.Close()
	stmtWork, err := tx.Prepare("INSERT OR REPLACE INTO workbench (uid, employer_id, x, y) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtWork.Close()
	stmtCulture, err := tx.Prepare("INSERT OR REPLACE INTO culture (uid, dialect_tick, foreign_ticks, language_id, foreign_language_id) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtCulture.Close()
	stmtExtra, err := tx.Prepare("INSERT OR REPLACE INTO extra_tags (uid, is_capital, is_country, is_admin, country_currency_id, country_debasement) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtExtra.Close()
	stmtAmb, err := tx.Prepare("INSERT OR REPLACE INTO ambitions (uid, ambitions_json, offers_json, built_count, family_base) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtAmb.Close()
	stmtDynasty, err := tx.Prepare("INSERT OR REPLACE INTO dynasty (uid, spouse_id, children, married) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtDynasty.Close()
	stmtPlot, err := tx.Prepare("INSERT OR REPLACE INTO plot (uid, target_id, start_tick, progress, power, kind, exposed) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtPlot.Close()
	stmtCouncil, err := tx.Prepare("INSERT OR REPLACE INTO council (uid, steward, marshal, diplomat, spymaster) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmtCouncil.Close()
	stmtDiplomacy, err := tx.Prepare("INSERT OR REPLACE INTO diplomacy (uid, relations_json) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtDiplomacy.Close()
	stmtTaxPolicy, err := tx.Prepare("INSERT OR REPLACE INTO tax_policy (uid, rate) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmtTaxPolicy.Close()

	// Extract components
	ambitionsID := ecs.ComponentID[components.AmbitionsComponent](world)
	dynastyCompID := ecs.ComponentID[components.DynastyComponent](world)
	plotCompID := ecs.ComponentID[components.PlotComponent](world)
	councilCompID := ecs.ComponentID[components.CouncilComponent](world)
	diplomacyCompID := ecs.ComponentID[components.DiplomacyComponent](world)
	taxPolicyCompID := ecs.ComponentID[components.TaxPolicyComponent](world)
	idID := ecs.ComponentID[components.Identity](world)
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	velID := ecs.ComponentID[components.Velocity](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	npcID := ecs.ComponentID[components.NPC](world)
	possessedID := ecs.ComponentID[components.Possessed](world)
	memID := ecs.ComponentID[components.Memory](world)
	beliefID := ecs.ComponentID[components.BeliefComponent](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	vitID := ecs.ComponentID[components.VitalsComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	secID := ecs.ComponentID[components.SecretComponent](world)
	equipID := ecs.ComponentID[components.EquipmentComponent](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	jurisID := ecs.ComponentID[components.JurisdictionComponent](world)
	structID := ecs.ComponentID[components.StructureComponent](world)
	constrID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	demoID := ecs.ComponentID[components.DemographicsComponent](world)
	workID := ecs.ComponentID[components.WorkbenchComponent](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	countryID := ecs.ComponentID[components.CountryComponent](world)
	adminID := ecs.ComponentID[components.AdministrationMarker](world)

	// We query entities with Identity
	query := world.Query(ecs.All(idID))
	for query.Next() {
		ent := query.Entity()
		ident := (*components.Identity)(query.Get(idID))
		uid := ident.ID

		if _, err := stmtEnt.Exec(uid); err != nil {
			query.Close()
			return err
		}
		if _, err := stmtId.Exec(uid, ident.Name, ident.BaseTraits, ident.Age); err != nil {
			query.Close()
			return err
		}

		if world.Has(ent, posID) {
			pos := (*components.Position)(world.Get(ent, posID))
			if _, err := stmtPos.Exec(uid, pos.X, pos.Y); err != nil {
				query.Close()
				return err
			}
		}

		if world.Has(ent, needsID) {
			needs := (*components.Needs)(world.Get(ent, needsID))
			if _, err := stmtNeeds.Exec(uid, needs.Food, needs.Rest, needs.Safety, needs.Wealth); err != nil {
				query.Close()
				return err
			}
		}

		if world.Has(ent, affID) {
			aff := (*components.Affiliation)(world.Get(ent, affID))
			if _, err := stmtAff.Exec(uid, aff.FamilyID, aff.ClanID, aff.CityID, aff.CountryID); err != nil {
				query.Close()
				return err
			}
		}

		// Tags
		isVillage := world.Has(ent, villageID)
		isNPC := world.Has(ent, npcID)
		isPossessed := world.Has(ent, possessedID)
		if isVillage || isNPC || isPossessed {
			if _, err := stmtTags.Exec(uid, isVillage, isNPC, isPossessed); err != nil {
				query.Close()
				return err
			}
		}

		// Storage
		if world.Has(ent, storageID) {
			store := (*components.StorageComponent)(world.Get(ent, storageID))
			if _, err := stmtStorage.Exec(uid, store.Wood, store.Stone, store.Iron, store.Food); err != nil {
				query.Close()
				return err
			}
		}

		// Velocity
		if world.Has(ent, velID) {
			vel := (*components.Velocity)(world.Get(ent, velID))
			if _, err := stmtVel.Exec(uid, vel.X, vel.Y); err != nil {
				query.Close()
				return err
			}
		}

		// Job
		if world.Has(ent, jobID) {
			job := (*components.JobComponent)(world.Get(ent, jobID))
			if _, err := stmtJob.Exec(uid, job.JobID, job.EmployerID); err != nil {
				query.Close()
				return err
			}
		}

		// Memory
		if world.Has(ent, memID) {
			mem := (*components.Memory)(world.Get(ent, memID))
			eventsJson, err := json.Marshal(mem.Events)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtMem.Exec(uid, string(eventsJson), mem.Head); err != nil {
				query.Close()
				return err
			}
		}

		// Beliefs
		if world.Has(ent, beliefID) {
			b := (*components.BeliefComponent)(world.Get(ent, beliefID))
			bJson, err := json.Marshal(b.Beliefs)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtBeliefs.Exec(uid, string(bJson)); err != nil {
				query.Close()
				return err
			}
		}

		// Genome
		if world.Has(ent, genID) {
			g := (*components.GenomeComponent)(world.Get(ent, genID))
			if _, err := stmtGen.Exec(uid, g.Strength, g.Beauty, g.Health, g.Intellect, g.Dominant, g.Recessive); err != nil {
				query.Close()
				return err
			}
		}

		// Vitals
		if world.Has(ent, vitID) {
			v := (*components.VitalsComponent)(world.Get(ent, vitID))
			if _, err := stmtVitals.Exec(uid, v.Stamina, v.Blood, v.Pain, v.Consciousness); err != nil {
				query.Close()
				return err
			}
		}

		// Population
		if world.Has(ent, popID) {
			p := (*components.PopulationComponent)(world.Get(ent, popID))
			citJson, err := json.Marshal(p.Citizens)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtPop.Exec(uid, p.Count, string(citJson)); err != nil {
				query.Close()
				return err
			}
		}

		// Desperation
		if world.Has(ent, despID) {
			d := (*components.DesperationComponent)(world.Get(ent, despID))
			if _, err := stmtDesp.Exec(uid, d.Level); err != nil {
				query.Close()
				return err
			}
		}

		// Secrets
		if world.Has(ent, secID) {
			s := (*components.SecretComponent)(world.Get(ent, secID))
			sJson, err := json.Marshal(s.Secrets)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtSec.Exec(uid, string(sJson)); err != nil {
				query.Close()
				return err
			}
		}

		// Equipment
		if world.Has(ent, equipID) {
			eq := (*components.EquipmentComponent)(world.Get(ent, equipID))
			histJson, err := json.Marshal(eq.Weapon.History)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtEquip.Exec(uid, eq.Weapon.NameID, eq.Weapon.Prestige, string(histJson), eq.Equipped); err != nil {
				query.Close()
				return err
			}
		}

		// Sanity
		if world.Has(ent, sanityID) {
			s := (*components.SanityComponent)(world.Get(ent, sanityID))
			if _, err := stmtSanity.Exec(uid, s.Stress, s.MaxStress, s.BreakState, s.BreakCooldown); err != nil {
				query.Close()
				return err
			}
		}

		// Treasury
		if world.Has(ent, treasuryID) {
			t := (*components.TreasuryComponent)(world.Get(ent, treasuryID))
			if _, err := stmtTreasury.Exec(uid, t.Wealth); err != nil {
				query.Close()
				return err
			}
		}

		// Market
		if world.Has(ent, marketID) {
			m := (*components.MarketComponent)(world.Get(ent, marketID))
			if _, err := stmtMarket.Exec(uid, m.WoodPrice, m.StonePrice, m.IronPrice, m.FoodPrice, m.WageRate); err != nil {
				query.Close()
				return err
			}
		}

		// Legitimacy
		if world.Has(ent, legitID) {
			l := (*components.LegitimacyComponent)(world.Get(ent, legitID))
			if _, err := stmtLegit.Exec(uid, l.Score); err != nil {
				query.Close()
				return err
			}
		}

		// Loyalty
		if world.Has(ent, loyaltyID) {
			l := (*components.LoyaltyComponent)(world.Get(ent, loyaltyID))
			if _, err := stmtLoyalty.Exec(uid, l.Value); err != nil {
				query.Close()
				return err
			}
		}

		// Jurisdiction
		if world.Has(ent, jurisID) {
			j := (*components.JurisdictionComponent)(world.Get(ent, jurisID))
			if _, err := stmtJuris.Exec(uid, j.RadiusSquared, j.IllegalActionIDs, j.Corruption, j.BannedSecretID, j.Trauma); err != nil {
				query.Close()
				return err
			}
		}

		// Structure
		if world.Has(ent, structID) {
			s := (*components.StructureComponent)(world.Get(ent, structID))
			if _, err := stmtStruct.Exec(uid, s.StructureType, s.Integrity, s.DataA, s.OwnerID); err != nil {
				query.Close()
				return err
			}
		}

		// ConstructionSite
		if world.Has(ent, constrID) {
			c := (*components.ConstructionSiteComponent)(world.Get(ent, constrID))
			if _, err := stmtConstr.Exec(uid, c.WoodRequired, c.WoodGathered, c.StoneRequired, c.StoneGathered, c.Progress, c.MaxProgress, c.BuilderID, c.SiteType); err != nil {
				query.Close()
				return err
			}
		}

		// Demographics
		if world.Has(ent, demoID) {
			d := (*components.DemographicsComponent)(world.Get(ent, demoID))
			if _, err := stmtDemo.Exec(uid, d.PeakPopulation, d.LaborCrisisActive); err != nil {
				query.Close()
				return err
			}
		}

		// Workbench
		if world.Has(ent, workID) {
			w := (*components.WorkbenchComponent)(world.Get(ent, workID))
			if _, err := stmtWork.Exec(uid, w.EmployerID, w.X, w.Y); err != nil {
				query.Close()
				return err
			}
		}

		// Culture
		if world.Has(ent, cultureID) {
			c := (*components.CultureComponent)(world.Get(ent, cultureID))
			if _, err := stmtCulture.Exec(uid, c.DialectTickStamp, c.ForeignInteractionTicks, c.LanguageID, c.ForeignLanguageID); err != nil {
				query.Close()
				return err
			}
		}

		// Ambitions (player goal progression)
		if world.Has(ent, ambitionsID) {
			a := (*components.AmbitionsComponent)(world.Get(ent, ambitionsID))
			ambJson, err := json.Marshal(a.Ambitions)
			if err != nil {
				query.Close()
				return err
			}
			offJson, err := json.Marshal(a.Offers)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtAmb.Exec(uid, string(ambJson), string(offJson), a.BuiltCount, a.FamilyBase); err != nil {
				query.Close()
				return err
			}
		}

		// Dynasty (Grand Strategy P2.3)
		if world.Has(ent, dynastyCompID) {
			d := (*components.DynastyComponent)(world.Get(ent, dynastyCompID))
			if _, err := stmtDynasty.Exec(uid, d.SpouseID, d.Children, d.Married); err != nil {
				query.Close()
				return err
			}
		}

		// Plot (Grand Strategy P2.4)
		if world.Has(ent, plotCompID) {
			p := (*components.PlotComponent)(world.Get(ent, plotCompID))
			if _, err := stmtPlot.Exec(uid, p.TargetID, p.StartTick, p.Progress, p.Power, p.Kind, p.Exposed); err != nil {
				query.Close()
				return err
			}
		}

		// Council (Grand Strategy P2.5)
		if world.Has(ent, councilCompID) {
			c := (*components.CouncilComponent)(world.Get(ent, councilCompID))
			if _, err := stmtCouncil.Exec(uid, c.Steward, c.Marshal, c.Diplomat, c.Spymaster); err != nil {
				query.Close()
				return err
			}
		}

		// Diplomacy relations ledger (Grand Strategy P2.2)
		if world.Has(ent, diplomacyCompID) {
			d := (*components.DiplomacyComponent)(world.Get(ent, diplomacyCompID))
			relJson, err := json.Marshal(d.Relations)
			if err != nil {
				query.Close()
				return err
			}
			if _, err := stmtDiplomacy.Exec(uid, string(relJson)); err != nil {
				query.Close()
				return err
			}
		}

		// Tax policy (Grand Strategy P2.6)
		if world.Has(ent, taxPolicyCompID) {
			p := (*components.TaxPolicyComponent)(world.Get(ent, taxPolicyCompID))
			if _, err := stmtTaxPolicy.Exec(uid, p.Rate); err != nil {
				query.Close()
				return err
			}
		}

		// Extra tags (Capital, Country, Administration)
		isCapital := world.Has(ent, capitalID)
		isCountry := world.Has(ent, countryID)
		isAdmin := world.Has(ent, adminID)
		if isCapital || isCountry || isAdmin {
			var currencyID uint32
			var debasement float32
			if isCountry {
				ctry := (*components.CountryComponent)(world.Get(ent, countryID))
				currencyID = ctry.StandardCurrencyID
				debasement = ctry.Debasement
			}
			if _, err := stmtExtra.Exec(uid, isCapital, isCountry, isAdmin, currencyID, debasement); err != nil {
				query.Close()
				return err
			}
		}
	}

	// Trade routes (Grand Strategy P2.6): route entities carry no Identity,
	// so they are collected after the Identity loop and stored keyless.
	routeCompID := ecs.ComponentID[components.TradeRouteComponent](world)
	type routeRow struct {
		from, to uint32
		vol      uint16
	}
	var routeRows []routeRow
	rq := world.Query(ecs.All(routeCompID))
	for rq.Next() {
		r := (*components.TradeRouteComponent)(rq.Get(routeCompID))
		routeRows = append(routeRows, routeRow{r.FromCity, r.ToCity, r.Volume})
	}
	if _, err := tx.Exec("DELETE FROM trade_routes"); err != nil {
		return err
	}
	for _, r := range routeRows {
		if _, err := tx.Exec("INSERT INTO trade_routes (from_city, to_city, volume) VALUES (?, ?, ?)", r.from, r.to, r.vol); err != nil {
			return err
		}
	}

	// Commit
	return tx.Commit()
}

// LoadGameState retrieves the global configuration parameters before loading entities.
func LoadGameState(db *sql.DB) (uint64, int, int, byte, error) {
	var ticks uint64
	var w, h int
	var seed int

	row := db.QueryRow("SELECT ticks, grid_width, grid_height, seed_val FROM game_state WHERE id = 1")
	err := row.Scan(&ticks, &w, &h, &seed)
	if err != nil {
		return 0, 256, 256, 1, err // Fallbacks
	}
	return ticks, w, h, byte(seed), nil
}

// LoadWorld reconstructs the ECS state from SQLite via memory maps to bypass N+1 DB row query bottleneck.
func LoadWorld(tm *TickManager, db *sql.DB) error {
	world := tm.World

	// Load Ticks
	ticks, _, _, _, err := LoadGameState(db)
	if err != nil {
		return err
	}
	tm.Ticks = ticks

	// Before loading, remove all existing entities to prevent duplication
	filter := ecs.All() // Select all entities
	query := world.Query(filter)
	var toRemove []ecs.Entity
	for query.Next() {
		toRemove = append(toRemove, query.Entity())
	}
	for _, e := range toRemove {
		world.RemoveEntity(e)
	}

	// 1. Fetch UIDs
	var uids []uint64
	rowsEnt, err := db.Query("SELECT uid FROM entities")
	if err != nil {
		return err
	}
	defer rowsEnt.Close()
	for rowsEnt.Next() {
		var u uint64
		if err := rowsEnt.Scan(&u); err != nil {
			return err
		}
		uids = append(uids, u)
	}
	if err := rowsEnt.Err(); err != nil {
		return err
	}

	// 2. Fetch Identity
	type idData struct {
		name   string
		traits uint32
		age    uint16
	}
	identities := make(map[uint64]idData)
	rowsId, err := db.Query("SELECT uid, name, basetraits, age FROM identity")
	if err != nil {
		return err
	}
	defer rowsId.Close()
	for rowsId.Next() {
		var u uint64
		var d idData
		if err := rowsId.Scan(&u, &d.name, &d.traits, &d.age); err != nil {
			return err
		}
		identities[u] = d
	}
	if err := rowsId.Err(); err != nil {
		return err
	}

	// 3. Fetch Position
	type posData struct{ x, y float32 }
	positions := make(map[uint64]posData)
	rowsPos, err := db.Query("SELECT uid, x, y FROM position")
	if err != nil {
		return err
	}
	defer rowsPos.Close()
	for rowsPos.Next() {
		var u uint64
		var p posData
		if err := rowsPos.Scan(&u, &p.x, &p.y); err != nil {
			return err
		}
		positions[u] = p
	}
	if err := rowsPos.Err(); err != nil {
		return err
	}

	// 4. Fetch Needs
	type needsData struct{ f, r, s, w float32 }
	needsMap := make(map[uint64]needsData)
	rowsNeeds, err := db.Query("SELECT uid, food, rest, safety, wealth FROM needs")
	if err != nil {
		return err
	}
	defer rowsNeeds.Close()
	for rowsNeeds.Next() {
		var u uint64
		var n needsData
		if err := rowsNeeds.Scan(&u, &n.f, &n.r, &n.s, &n.w); err != nil {
			return err
		}
		needsMap[u] = n
	}
	if err := rowsNeeds.Err(); err != nil {
		return err
	}

	// 5. Fetch Affiliation
	type affData struct{ fid, cid, cityid, ctryid uint32 }
	affMap := make(map[uint64]affData)
	rowsAff, err := db.Query("SELECT uid, family_id, clan_id, city_id, country_id FROM affiliation")
	if err != nil {
		return err
	}
	defer rowsAff.Close()
	for rowsAff.Next() {
		var u uint64
		var a affData
		if err := rowsAff.Scan(&u, &a.fid, &a.cid, &a.cityid, &a.ctryid); err != nil {
			return err
		}
		affMap[u] = a
	}
	if err := rowsAff.Err(); err != nil {
		return err
	}

	// 6. Fetch Tags
	type tagsData struct{ v, n, p bool }
	tagsMap := make(map[uint64]tagsData)
	rowsTags, err := db.Query("SELECT uid, is_village, is_npc, is_possessed FROM tags")
	if err != nil {
		return err
	}
	defer rowsTags.Close()
	for rowsTags.Next() {
		var u uint64
		var t tagsData
		if err := rowsTags.Scan(&u, &t.v, &t.n, &t.p); err != nil {
			return err
		}
		tagsMap[u] = t
	}
	if err := rowsTags.Err(); err != nil {
		return err
	}

	// 7. Fetch Storage
	type storeData struct{ w, s, i, f uint32 }
	storeMap := make(map[uint64]storeData)
	rowsStore, err := db.Query("SELECT uid, wood, stone, iron, food FROM storage")
	if err != nil {
		return err
	}
	defer rowsStore.Close()
	for rowsStore.Next() {
		var u uint64
		var s storeData
		if err := rowsStore.Scan(&u, &s.w, &s.s, &s.i, &s.f); err != nil {
			return err
		}
		storeMap[u] = s
	}
	if err := rowsStore.Err(); err != nil {
		return err
	}

	// 8. Fetch Velocity
	type velData struct{ vx, vy float32 }
	velMap := make(map[uint64]velData)
	rowsVel, err := db.Query("SELECT uid, x, y FROM velocity")
	if err != nil {
		return err
	}
	defer rowsVel.Close()
	for rowsVel.Next() {
		var u uint64
		var v velData
		if err := rowsVel.Scan(&u, &v.vx, &v.vy); err != nil {
			return err
		}
		velMap[u] = v
	}
	if err := rowsVel.Err(); err != nil {
		return err
	}

	// 9. Fetch Job
	type jobData struct {
		jid uint8
		eid uint64
	}
	jobMap := make(map[uint64]jobData)
	rowsJob, err := db.Query("SELECT uid, job_id, employer_id FROM job")
	if err != nil {
		return err
	}
	defer rowsJob.Close()
	for rowsJob.Next() {
		var u uint64
		var j jobData
		if err := rowsJob.Scan(&u, &j.jid, &j.eid); err != nil {
			return err
		}
		jobMap[u] = j
	}
	if err := rowsJob.Err(); err != nil {
		return err
	}

	// 10. Fetch Memory
	type memData struct {
		json string
		head uint8
	}
	memMap := make(map[uint64]memData)
	rowsMem, err := db.Query("SELECT uid, events_json, head FROM memory")
	if err != nil {
		return err
	}
	defer rowsMem.Close()
	for rowsMem.Next() {
		var u uint64
		var m memData
		if err := rowsMem.Scan(&u, &m.json, &m.head); err != nil {
			return err
		}
		memMap[u] = m
	}
	if err := rowsMem.Err(); err != nil {
		return err
	}

	// 11. Fetch Beliefs
	beliefsMap := make(map[uint64]string)
	rowsB, err := db.Query("SELECT uid, beliefs_json FROM beliefs")
	if err != nil {
		return err
	}
	defer rowsB.Close()
	for rowsB.Next() {
		var u uint64
		var j string
		if err := rowsB.Scan(&u, &j); err != nil {
			return err
		}
		beliefsMap[u] = j
	}
	if err := rowsB.Err(); err != nil {
		return err
	}

	// 12. Fetch Genome
	type genData struct {
		str, bea, hlt, itl uint8
		dom, rec           uint32
	}
	genMap := make(map[uint64]genData)
	rowsG, err := db.Query("SELECT uid, str, bea, hlt, itl, dom, rec FROM genome")
	if err != nil {
		return err
	}
	defer rowsG.Close()
	for rowsG.Next() {
		var u uint64
		var g genData
		if err := rowsG.Scan(&u, &g.str, &g.bea, &g.hlt, &g.itl, &g.dom, &g.rec); err != nil {
			return err
		}
		genMap[u] = g
	}
	if err := rowsG.Err(); err != nil {
		return err
	}

	// 13. Fetch Vitals
	type vitData struct{ s, b, p, c float32 }
	vitMap := make(map[uint64]vitData)
	rowsV, err := db.Query("SELECT uid, stamina, blood, pain, consciousness FROM vitals")
	if err != nil {
		return err
	}
	defer rowsV.Close()
	for rowsV.Next() {
		var u uint64
		var v vitData
		if err := rowsV.Scan(&u, &v.s, &v.b, &v.p, &v.c); err != nil {
			return err
		}
		vitMap[u] = v
	}
	if err := rowsV.Err(); err != nil {
		return err
	}

	// 14. Fetch Population
	type popData struct {
		count uint32
		json  string
	}
	popMap := make(map[uint64]popData)
	rowsP, err := db.Query("SELECT uid, count, citizens_json FROM population")
	if err != nil {
		return err
	}
	defer rowsP.Close()
	for rowsP.Next() {
		var u uint64
		var p popData
		if err := rowsP.Scan(&u, &p.count, &p.json); err != nil {
			return err
		}
		popMap[u] = p
	}
	if err := rowsP.Err(); err != nil {
		return err
	}

	// 15. Fetch Desperation
	despMap := make(map[uint64]uint8)
	rowsD, err := db.Query("SELECT uid, level FROM desperation")
	if err != nil {
		return err
	}
	defer rowsD.Close()
	for rowsD.Next() {
		var u uint64
		var l uint8
		if err := rowsD.Scan(&u, &l); err != nil {
			return err
		}
		despMap[u] = l
	}
	if err := rowsD.Err(); err != nil {
		return err
	}

	// 16. Fetch Secrets
	secMap := make(map[uint64]string)
	rowsS, err := db.Query("SELECT uid, secrets_json FROM secrets")
	if err != nil {
		return err
	}
	defer rowsS.Close()
	for rowsS.Next() {
		var u uint64
		var j string
		if err := rowsS.Scan(&u, &j); err != nil {
			return err
		}
		secMap[u] = j
	}
	if err := rowsS.Err(); err != nil {
		return err
	}

	// 17. Fetch Equipment
	type equipData struct {
		nameID, prestige uint32
		histJson         string
		equipped         bool
	}
	equipMap := make(map[uint64]equipData)
	rowsEquip, err := db.Query("SELECT uid, weapon_nameid, weapon_prestige, weapon_history_json, equipped FROM equipment")
	if err != nil {
		return err
	}
	defer rowsEquip.Close()
	for rowsEquip.Next() {
		var u uint64
		var e equipData
		if err := rowsEquip.Scan(&u, &e.nameID, &e.prestige, &e.histJson, &e.equipped); err != nil {
			return err
		}
		equipMap[u] = e
	}
	if err := rowsEquip.Err(); err != nil {
		return err
	}

	// 18. Fetch Sanity
	type sanityData struct {
		stress, maxStress         float32
		breakState, breakCooldown uint32
	}
	sanityMap := make(map[uint64]sanityData)
	rowsSanity, err := db.Query("SELECT uid, stress, max_stress, break_state, break_cooldown FROM sanity")
	if err != nil {
		return err
	}
	defer rowsSanity.Close()
	for rowsSanity.Next() {
		var u uint64
		var s sanityData
		if err := rowsSanity.Scan(&u, &s.stress, &s.maxStress, &s.breakState, &s.breakCooldown); err != nil {
			return err
		}
		sanityMap[u] = s
	}
	if err := rowsSanity.Err(); err != nil {
		return err
	}

	// 19. Fetch Treasury
	treasuryMap := make(map[uint64]float32)
	rowsTreasury, err := db.Query("SELECT uid, wealth FROM treasury")
	if err != nil {
		return err
	}
	defer rowsTreasury.Close()
	for rowsTreasury.Next() {
		var u uint64
		var w float32
		if err := rowsTreasury.Scan(&u, &w); err != nil {
			return err
		}
		treasuryMap[u] = w
	}
	if err := rowsTreasury.Err(); err != nil {
		return err
	}

	// 20. Fetch Market
	type marketData struct{ wood, stone, iron, food, wage float32 }
	marketMap := make(map[uint64]marketData)
	rowsMarket, err := db.Query("SELECT uid, wood_price, stone_price, iron_price, food_price, wage_rate FROM market")
	if err != nil {
		return err
	}
	defer rowsMarket.Close()
	for rowsMarket.Next() {
		var u uint64
		var m marketData
		if err := rowsMarket.Scan(&u, &m.wood, &m.stone, &m.iron, &m.food, &m.wage); err != nil {
			return err
		}
		marketMap[u] = m
	}
	if err := rowsMarket.Err(); err != nil {
		return err
	}

	// 21. Fetch Legitimacy
	legitMap := make(map[uint64]uint32)
	rowsLegit, err := db.Query("SELECT uid, score FROM legitimacy")
	if err != nil {
		return err
	}
	defer rowsLegit.Close()
	for rowsLegit.Next() {
		var u uint64
		var s uint32
		if err := rowsLegit.Scan(&u, &s); err != nil {
			return err
		}
		legitMap[u] = s
	}
	if err := rowsLegit.Err(); err != nil {
		return err
	}

	// 22. Fetch Loyalty
	loyaltyMap := make(map[uint64]uint32)
	rowsLoyalty, err := db.Query("SELECT uid, value FROM loyalty")
	if err != nil {
		return err
	}
	defer rowsLoyalty.Close()
	for rowsLoyalty.Next() {
		var u uint64
		var v uint32
		if err := rowsLoyalty.Scan(&u, &v); err != nil {
			return err
		}
		loyaltyMap[u] = v
	}
	if err := rowsLoyalty.Err(); err != nil {
		return err
	}

	// 23. Fetch Jurisdiction
	type jurisData struct {
		radiusSq                    float32
		illegal, corruption, banned uint32
		trauma                      uint16
	}
	jurisMap := make(map[uint64]jurisData)
	rowsJuris, err := db.Query("SELECT uid, radius_sq, illegal_action_ids, corruption, banned_secret_id, trauma FROM jurisdiction")
	if err != nil {
		return err
	}
	defer rowsJuris.Close()
	for rowsJuris.Next() {
		var u uint64
		var j jurisData
		if err := rowsJuris.Scan(&u, &j.radiusSq, &j.illegal, &j.corruption, &j.banned, &j.trauma); err != nil {
			return err
		}
		jurisMap[u] = j
	}
	if err := rowsJuris.Err(); err != nil {
		return err
	}

	// 24. Fetch Structure
	type structData struct {
		sType     uint32
		integrity float32
		dataA     uint32
		ownerID   uint32
	}
	structMap := make(map[uint64]structData)
	rowsStruct, err := db.Query("SELECT uid, structure_type, integrity, data_a, owner_id FROM structure")
	if err != nil {
		return err
	}
	defer rowsStruct.Close()
	for rowsStruct.Next() {
		var u uint64
		var s structData
		if err := rowsStruct.Scan(&u, &s.sType, &s.integrity, &s.dataA, &s.ownerID); err != nil {
			return err
		}
		structMap[u] = s
	}
	if err := rowsStruct.Err(); err != nil {
		return err
	}

	// 25. Fetch ConstructionSite
	type constrData struct {
		woodReq, woodGot, stoneReq, stoneGot, progress, maxProgress uint32
		builderID                                                   uint64
		siteType                                                    uint32
	}
	constrMap := make(map[uint64]constrData)
	rowsConstr, err := db.Query("SELECT uid, wood_req, wood_gathered, stone_req, stone_gathered, progress, max_progress, builder_id, site_type FROM construction_site")
	if err != nil {
		return err
	}
	defer rowsConstr.Close()
	for rowsConstr.Next() {
		var u uint64
		var c constrData
		if err := rowsConstr.Scan(&u, &c.woodReq, &c.woodGot, &c.stoneReq, &c.stoneGot, &c.progress, &c.maxProgress, &c.builderID, &c.siteType); err != nil {
			return err
		}
		constrMap[u] = c
	}
	if err := rowsConstr.Err(); err != nil {
		return err
	}

	// 26. Fetch Demographics
	type demoData struct {
		peak   uint32
		crisis bool
	}
	demoMap := make(map[uint64]demoData)
	rowsDemo, err := db.Query("SELECT uid, peak_population, labor_crisis FROM demographics")
	if err != nil {
		return err
	}
	defer rowsDemo.Close()
	for rowsDemo.Next() {
		var u uint64
		var d demoData
		if err := rowsDemo.Scan(&u, &d.peak, &d.crisis); err != nil {
			return err
		}
		demoMap[u] = d
	}
	if err := rowsDemo.Err(); err != nil {
		return err
	}

	// 27. Fetch Workbench
	type workData struct {
		employerID uint64
		x, y       float32
	}
	workMap := make(map[uint64]workData)
	rowsWork, err := db.Query("SELECT uid, employer_id, x, y FROM workbench")
	if err != nil {
		return err
	}
	defer rowsWork.Close()
	for rowsWork.Next() {
		var u uint64
		var w workData
		if err := rowsWork.Scan(&u, &w.employerID, &w.x, &w.y); err != nil {
			return err
		}
		workMap[u] = w
	}
	if err := rowsWork.Err(); err != nil {
		return err
	}

	// 28. Fetch Culture
	type cultureData struct {
		dialectTick           uint64
		foreignTicks          uint32
		langID, foreignLangID uint16
	}
	cultureMap := make(map[uint64]cultureData)
	rowsCulture, err := db.Query("SELECT uid, dialect_tick, foreign_ticks, language_id, foreign_language_id FROM culture")
	if err != nil {
		return err
	}
	defer rowsCulture.Close()
	for rowsCulture.Next() {
		var u uint64
		var c cultureData
		if err := rowsCulture.Scan(&u, &c.dialectTick, &c.foreignTicks, &c.langID, &c.foreignLangID); err != nil {
			return err
		}
		cultureMap[u] = c
	}
	if err := rowsCulture.Err(); err != nil {
		return err
	}

	// 29. Fetch ExtraTags
	type extraData struct {
		capital, country, admin bool
		currencyID              uint32
		debasement              float32
	}
	extraMap := make(map[uint64]extraData)
	rowsExtra, err := db.Query("SELECT uid, is_capital, is_country, is_admin, country_currency_id, country_debasement FROM extra_tags")
	if err != nil {
		return err
	}
	defer rowsExtra.Close()
	for rowsExtra.Next() {
		var u uint64
		var e extraData
		if err := rowsExtra.Scan(&u, &e.capital, &e.country, &e.admin, &e.currencyID, &e.debasement); err != nil {
			return err
		}
		extraMap[u] = e
	}
	if err := rowsExtra.Err(); err != nil {
		return err
	}

	// 30. Fetch Ambitions
	type ambData struct {
		ambJson, offJson  string
		built, familyBase uint32
	}
	ambMap := make(map[uint64]ambData)
	rowsAmb, err := db.Query("SELECT uid, ambitions_json, offers_json, built_count, family_base FROM ambitions")
	if err != nil {
		return err
	}
	defer rowsAmb.Close()
	for rowsAmb.Next() {
		var u uint64
		var a ambData
		if err := rowsAmb.Scan(&u, &a.ambJson, &a.offJson, &a.built, &a.familyBase); err != nil {
			return err
		}
		ambMap[u] = a
	}
	if err := rowsAmb.Err(); err != nil {
		return err
	}

	// 31. Fetch Dynasty (Grand Strategy P2.3)
	type dynastyData struct {
		spouseID uint64
		children uint16
		married  bool
	}
	dynastyMap := make(map[uint64]dynastyData)
	rowsDyn, err := db.Query("SELECT uid, spouse_id, children, married FROM dynasty")
	if err != nil {
		return err
	}
	defer rowsDyn.Close()
	for rowsDyn.Next() {
		var u uint64
		var d dynastyData
		if err := rowsDyn.Scan(&u, &d.spouseID, &d.children, &d.married); err != nil {
			return err
		}
		dynastyMap[u] = d
	}
	if err := rowsDyn.Err(); err != nil {
		return err
	}

	// 32. Fetch Plot (Grand Strategy P2.4)
	type plotData struct {
		targetID  uint64
		startTick uint64
		progress  uint16
		power     uint16
		kind      uint8
		exposed   bool
	}
	plotMap := make(map[uint64]plotData)
	rowsPlot, err := db.Query("SELECT uid, target_id, start_tick, progress, power, kind, exposed FROM plot")
	if err != nil {
		return err
	}
	defer rowsPlot.Close()
	for rowsPlot.Next() {
		var u uint64
		var p plotData
		if err := rowsPlot.Scan(&u, &p.targetID, &p.startTick, &p.progress, &p.power, &p.kind, &p.exposed); err != nil {
			return err
		}
		plotMap[u] = p
	}
	if err := rowsPlot.Err(); err != nil {
		return err
	}

	// 33. Fetch Council (Grand Strategy P2.5)
	type councilData struct {
		steward, marshal, diplomat, spymaster uint64
	}
	councilMap := make(map[uint64]councilData)
	rowsCouncil, err := db.Query("SELECT uid, steward, marshal, diplomat, spymaster FROM council")
	if err != nil {
		return err
	}
	defer rowsCouncil.Close()
	for rowsCouncil.Next() {
		var u uint64
		var c councilData
		if err := rowsCouncil.Scan(&u, &c.steward, &c.marshal, &c.diplomat, &c.spymaster); err != nil {
			return err
		}
		councilMap[u] = c
	}
	if err := rowsCouncil.Err(); err != nil {
		return err
	}

	// 34. Fetch Diplomacy relations (Grand Strategy P2.2)
	diploMap := make(map[uint64]string)
	rowsDiplo, err := db.Query("SELECT uid, relations_json FROM diplomacy")
	if err != nil {
		return err
	}
	defer rowsDiplo.Close()
	for rowsDiplo.Next() {
		var u uint64
		var relJson string
		if err := rowsDiplo.Scan(&u, &relJson); err != nil {
			return err
		}
		diploMap[u] = relJson
	}
	if err := rowsDiplo.Err(); err != nil {
		return err
	}

	// 35. Fetch Tax Policy (Grand Strategy P2.6)
	taxMap := make(map[uint64]uint8)
	rowsTax, err := db.Query("SELECT uid, rate FROM tax_policy")
	if err != nil {
		return err
	}
	defer rowsTax.Close()
	for rowsTax.Next() {
		var u uint64
		var rate uint8
		if err := rowsTax.Scan(&u, &rate); err != nil {
			return err
		}
		taxMap[u] = rate
	}
	if err := rowsTax.Err(); err != nil {
		return err
	}

	// Component IDs
	idID := ecs.ComponentID[components.Identity](world)
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	velID := ecs.ComponentID[components.Velocity](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	npcID := ecs.ComponentID[components.NPC](world)
	possessedID := ecs.ComponentID[components.Possessed](world)
	memID := ecs.ComponentID[components.Memory](world)
	beliefID := ecs.ComponentID[components.BeliefComponent](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	vitID := ecs.ComponentID[components.VitalsComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	secID := ecs.ComponentID[components.SecretComponent](world)
	equipID := ecs.ComponentID[components.EquipmentComponent](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	jurisID := ecs.ComponentID[components.JurisdictionComponent](world)
	structID := ecs.ComponentID[components.StructureComponent](world)
	constrID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	demoID := ecs.ComponentID[components.DemographicsComponent](world)
	workID := ecs.ComponentID[components.WorkbenchComponent](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	countryID := ecs.ComponentID[components.CountryComponent](world)
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	ambitionsID := ecs.ComponentID[components.AmbitionsComponent](world)
	dynastyCompID := ecs.ComponentID[components.DynastyComponent](world)
	plotCompID := ecs.ComponentID[components.PlotComponent](world)
	councilCompID := ecs.ComponentID[components.CouncilComponent](world)
	diplomacyCompID := ecs.ComponentID[components.DiplomacyComponent](world)
	taxPolicyCompID := ecs.ComponentID[components.TaxPolicyComponent](world)

	for _, uid := range uids {
		ent := world.NewEntity()

		if d, ok := identities[uid]; ok {
			world.Add(ent, idID)
			ident := (*components.Identity)(world.Get(ent, idID))
			ident.ID = uid
			ident.Name = d.name
			ident.BaseTraits = d.traits
			ident.Age = d.age
		}

		if p, ok := positions[uid]; ok {
			world.Add(ent, posID)
			pos := (*components.Position)(world.Get(ent, posID))
			pos.X = p.x
			pos.Y = p.y
		}

		if n, ok := needsMap[uid]; ok {
			world.Add(ent, needsID)
			needs := (*components.Needs)(world.Get(ent, needsID))
			needs.Food = n.f
			needs.Rest = n.r
			needs.Safety = n.s
			needs.Wealth = n.w
		}

		if a, ok := affMap[uid]; ok {
			world.Add(ent, affID)
			aff := (*components.Affiliation)(world.Get(ent, affID))
			aff.FamilyID = a.fid
			aff.ClanID = a.cid
			aff.CityID = a.cityid
			aff.CountryID = a.ctryid
		}

		if t, ok := tagsMap[uid]; ok {
			if t.v {
				world.Add(ent, villageID)
			}
			if t.n {
				world.Add(ent, npcID)
			}
			if t.p {
				world.Add(ent, possessedID)
			}
		}

		if s, ok := storeMap[uid]; ok {
			world.Add(ent, storageID)
			store := (*components.StorageComponent)(world.Get(ent, storageID))
			store.Wood = s.w
			store.Stone = s.s
			store.Iron = s.i
			store.Food = s.f
		}

		if v, ok := velMap[uid]; ok {
			world.Add(ent, velID)
			vel := (*components.Velocity)(world.Get(ent, velID))
			vel.X = v.vx
			vel.Y = v.vy
		}

		if j, ok := jobMap[uid]; ok {
			world.Add(ent, jobID)
			job := (*components.JobComponent)(world.Get(ent, jobID))
			job.JobID = j.jid
			job.EmployerID = j.eid
		}

		if m, ok := memMap[uid]; ok {
			world.Add(ent, memID)
			mem := (*components.Memory)(world.Get(ent, memID))
			mem.Head = m.head
			var events [50]components.MemoryEvent
			if err := json.Unmarshal([]byte(m.json), &events); err != nil {
				return err
			}
			mem.Events = events
		}

		if bstr, ok := beliefsMap[uid]; ok {
			world.Add(ent, beliefID)
			b := (*components.BeliefComponent)(world.Get(ent, beliefID))
			var beliefs []components.Belief
			if err := json.Unmarshal([]byte(bstr), &beliefs); err != nil {
				return err
			}
			b.Beliefs = beliefs
		}

		if g, ok := genMap[uid]; ok {
			world.Add(ent, genID)
			gen := (*components.GenomeComponent)(world.Get(ent, genID))
			gen.Strength = g.str
			gen.Beauty = g.bea
			gen.Health = g.hlt
			gen.Intellect = g.itl
			gen.Dominant = g.dom
			gen.Recessive = g.rec
		}

		if v, ok := vitMap[uid]; ok {
			world.Add(ent, vitID)
			vit := (*components.VitalsComponent)(world.Get(ent, vitID))
			vit.Stamina = v.s
			vit.Blood = v.b
			vit.Pain = v.p
			vit.Consciousness = v.c
		}

		if p, ok := popMap[uid]; ok {
			world.Add(ent, popID)
			pop := (*components.PopulationComponent)(world.Get(ent, popID))
			pop.Count = p.count
			var cits []components.CitizenData
			if err := json.Unmarshal([]byte(p.json), &cits); err != nil {
				return err
			}
			pop.Citizens = cits
		}

		if d, ok := despMap[uid]; ok {
			world.Add(ent, despID)
			desp := (*components.DesperationComponent)(world.Get(ent, despID))
			desp.Level = d
		}

		if s, ok := secMap[uid]; ok {
			world.Add(ent, secID)
			sec := (*components.SecretComponent)(world.Get(ent, secID))
			var secrets []components.Secret
			if err := json.Unmarshal([]byte(s), &secrets); err != nil {
				return err
			}
			sec.Secrets = secrets
		}

		if e, ok := equipMap[uid]; ok {
			world.Add(ent, equipID)
			eq := (*components.EquipmentComponent)(world.Get(ent, equipID))
			eq.Weapon.NameID = e.nameID
			eq.Weapon.Prestige = e.prestige
			eq.Equipped = e.equipped
			var hist []uint32
			if err := json.Unmarshal([]byte(e.histJson), &hist); err != nil {
				return err
			}
			eq.Weapon.History = hist
		}

		if s, ok := sanityMap[uid]; ok {
			world.Add(ent, sanityID)
			san := (*components.SanityComponent)(world.Get(ent, sanityID))
			san.Stress = s.stress
			san.MaxStress = s.maxStress
			san.BreakState = s.breakState
			san.BreakCooldown = s.breakCooldown
		}

		if w, ok := treasuryMap[uid]; ok {
			world.Add(ent, treasuryID)
			t := (*components.TreasuryComponent)(world.Get(ent, treasuryID))
			t.Wealth = w
		}

		if m, ok := marketMap[uid]; ok {
			world.Add(ent, marketID)
			mk := (*components.MarketComponent)(world.Get(ent, marketID))
			mk.WoodPrice = m.wood
			mk.StonePrice = m.stone
			mk.IronPrice = m.iron
			mk.FoodPrice = m.food
			mk.WageRate = m.wage
		}

		if s, ok := legitMap[uid]; ok {
			world.Add(ent, legitID)
			l := (*components.LegitimacyComponent)(world.Get(ent, legitID))
			l.Score = s
		}

		if v, ok := loyaltyMap[uid]; ok {
			world.Add(ent, loyaltyID)
			l := (*components.LoyaltyComponent)(world.Get(ent, loyaltyID))
			l.Value = v
		}

		if j, ok := jurisMap[uid]; ok {
			world.Add(ent, jurisID)
			juris := (*components.JurisdictionComponent)(world.Get(ent, jurisID))
			juris.RadiusSquared = j.radiusSq
			juris.IllegalActionIDs = j.illegal
			juris.Corruption = j.corruption
			juris.BannedSecretID = j.banned
			juris.Trauma = j.trauma
		}

		if s, ok := structMap[uid]; ok {
			world.Add(ent, structID)
			st := (*components.StructureComponent)(world.Get(ent, structID))
			st.StructureType = s.sType
			st.Integrity = s.integrity
			st.DataA = s.dataA
			st.OwnerID = s.ownerID
		}

		if c, ok := constrMap[uid]; ok {
			world.Add(ent, constrID)
			cs := (*components.ConstructionSiteComponent)(world.Get(ent, constrID))
			cs.WoodRequired = c.woodReq
			cs.WoodGathered = c.woodGot
			cs.StoneRequired = c.stoneReq
			cs.StoneGathered = c.stoneGot
			cs.Progress = c.progress
			cs.MaxProgress = c.maxProgress
			cs.BuilderID = c.builderID
			cs.SiteType = c.siteType
		}

		if d, ok := demoMap[uid]; ok {
			world.Add(ent, demoID)
			dm := (*components.DemographicsComponent)(world.Get(ent, demoID))
			dm.PeakPopulation = d.peak
			dm.LaborCrisisActive = d.crisis
		}

		if w, ok := workMap[uid]; ok {
			world.Add(ent, workID)
			wb := (*components.WorkbenchComponent)(world.Get(ent, workID))
			wb.EmployerID = w.employerID
			wb.X = w.x
			wb.Y = w.y
		}

		if c, ok := cultureMap[uid]; ok {
			world.Add(ent, cultureID)
			cul := (*components.CultureComponent)(world.Get(ent, cultureID))
			cul.DialectTickStamp = c.dialectTick
			cul.ForeignInteractionTicks = c.foreignTicks
			cul.LanguageID = c.langID
			cul.ForeignLanguageID = c.foreignLangID
		}

		if e, ok := extraMap[uid]; ok {
			if e.capital {
				world.Add(ent, capitalID)
			}
			if e.admin {
				world.Add(ent, adminID)
			}
			if e.country {
				world.Add(ent, countryID)
				ctry := (*components.CountryComponent)(world.Get(ent, countryID))
				ctry.StandardCurrencyID = e.currencyID
				ctry.Debasement = e.debasement
			}
		}

		if a, ok := ambMap[uid]; ok {
			world.Add(ent, ambitionsID)
			amb := (*components.AmbitionsComponent)(world.Get(ent, ambitionsID))
			_ = json.Unmarshal([]byte(a.ambJson), &amb.Ambitions)
			_ = json.Unmarshal([]byte(a.offJson), &amb.Offers)
			amb.BuiltCount = a.built
			amb.FamilyBase = a.familyBase
		}

		if d, ok := dynastyMap[uid]; ok {
			world.Add(ent, dynastyCompID)
			dyn := (*components.DynastyComponent)(world.Get(ent, dynastyCompID))
			dyn.SpouseID = d.spouseID
			dyn.Children = d.children
			dyn.Married = d.married
		}

		if p, ok := plotMap[uid]; ok {
			world.Add(ent, plotCompID)
			plot := (*components.PlotComponent)(world.Get(ent, plotCompID))
			plot.TargetID = p.targetID
			plot.StartTick = p.startTick
			plot.Progress = p.progress
			plot.Power = p.power
			plot.Kind = p.kind
			plot.Exposed = p.exposed
		}

		if c, ok := councilMap[uid]; ok {
			world.Add(ent, councilCompID)
			council := (*components.CouncilComponent)(world.Get(ent, councilCompID))
			council.Steward = c.steward
			council.Marshal = c.marshal
			council.Diplomat = c.diplomat
			council.Spymaster = c.spymaster
		}

		if relJson, ok := diploMap[uid]; ok {
			world.Add(ent, diplomacyCompID)
			d := (*components.DiplomacyComponent)(world.Get(ent, diplomacyCompID))
			_ = json.Unmarshal([]byte(relJson), &d.Relations)
		}

		if rate, ok := taxMap[uid]; ok {
			world.Add(ent, taxPolicyCompID)
			(*components.TaxPolicyComponent)(world.Get(ent, taxPolicyCompID)).Rate = rate
		}
	}

	// Trade routes are standalone entities recreated from their keyless table.
	routeCompID := ecs.ComponentID[components.TradeRouteComponent](world)
	type routeRow struct {
		from, to uint32
		vol      uint16
	}
	var routeRows []routeRow
	trRows, err := db.Query("SELECT from_city, to_city, volume FROM trade_routes ORDER BY rowid")
	if err != nil {
		return err
	}
	defer trRows.Close()
	for trRows.Next() {
		var r routeRow
		if err := trRows.Scan(&r.from, &r.to, &r.vol); err != nil {
			return err
		}
		routeRows = append(routeRows, r)
	}
	if err := trRows.Err(); err != nil {
		return err
	}
	for _, r := range routeRows {
		ent := world.NewEntity(routeCompID)
		route := (*components.TradeRouteComponent)(world.Get(ent, routeCompID))
		route.FromCity, route.ToCity, route.Volume = r.from, r.to, r.vol
	}

	return nil
}
