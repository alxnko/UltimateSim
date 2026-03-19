package engine

import (
	"database/sql"
	"encoding/json"

	"github.com/ALXNKO/UltimateSim/internal/components"
	_ "modernc.org/sqlite"
	"github.com/mlange-42/arche/ecs"
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
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return db, nil
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
	tables := []string{"entities", "identity", "position", "needs", "affiliation", "tags", "storage", "velocity", "job", "memory", "beliefs", "genome", "vitals", "population", "desperation", "secrets"}
	for _, table := range tables {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return err
		}
	}

	// Prepare statements
	stmtEnt, err := tx.Prepare("INSERT OR REPLACE INTO entities (uid) VALUES (?)")
	if err != nil { return err }
	defer stmtEnt.Close()
	stmtId, err := tx.Prepare("INSERT OR REPLACE INTO identity (uid, name, basetraits, age) VALUES (?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtId.Close()
	stmtPos, err := tx.Prepare("INSERT OR REPLACE INTO position (uid, x, y) VALUES (?, ?, ?)")
	if err != nil { return err }
	defer stmtPos.Close()
	stmtNeeds, err := tx.Prepare("INSERT OR REPLACE INTO needs (uid, food, rest, safety, wealth) VALUES (?, ?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtNeeds.Close()
	stmtAff, err := tx.Prepare("INSERT OR REPLACE INTO affiliation (uid, family_id, clan_id, city_id, country_id) VALUES (?, ?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtAff.Close()
	stmtTags, err := tx.Prepare("INSERT OR REPLACE INTO tags (uid, is_village, is_npc, is_possessed) VALUES (?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtTags.Close()
	stmtStorage, err := tx.Prepare("INSERT OR REPLACE INTO storage (uid, wood, stone, iron, food) VALUES (?, ?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtStorage.Close()
	stmtVel, err := tx.Prepare("INSERT OR REPLACE INTO velocity (uid, x, y) VALUES (?, ?, ?)")
	if err != nil { return err }
	defer stmtVel.Close()
	stmtJob, err := tx.Prepare("INSERT OR REPLACE INTO job (uid, job_id, employer_id) VALUES (?, ?, ?)")
	if err != nil { return err }
	defer stmtJob.Close()
	stmtMem, err := tx.Prepare("INSERT OR REPLACE INTO memory (uid, events_json, head) VALUES (?, ?, ?)")
	if err != nil { return err }
	defer stmtMem.Close()
	stmtBeliefs, err := tx.Prepare("INSERT OR REPLACE INTO beliefs (uid, beliefs_json) VALUES (?, ?)")
	if err != nil { return err }
	defer stmtBeliefs.Close()
	stmtGen, err := tx.Prepare("INSERT OR REPLACE INTO genome (uid, str, bea, hlt, itl, dom, rec) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtGen.Close()
	stmtVitals, err := tx.Prepare("INSERT OR REPLACE INTO vitals (uid, stamina, blood, pain, consciousness) VALUES (?, ?, ?, ?, ?)")
	if err != nil { return err }
	defer stmtVitals.Close()
	stmtPop, err := tx.Prepare("INSERT OR REPLACE INTO population (uid, count, citizens_json) VALUES (?, ?, ?)")
	if err != nil { return err }
	defer stmtPop.Close()
	stmtDesp, err := tx.Prepare("INSERT OR REPLACE INTO desperation (uid, level) VALUES (?, ?)")
	if err != nil { return err }
	defer stmtDesp.Close()
	stmtSec, err := tx.Prepare("INSERT OR REPLACE INTO secrets (uid, secrets_json) VALUES (?, ?)")
	if err != nil { return err }
	defer stmtSec.Close()

	// Extract components
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

	// We query entities with Identity
	query := world.Query(ecs.All(idID))
	for query.Next() {
		ent := query.Entity()
		ident := (*components.Identity)(query.Get(idID))
		uid := ident.ID

		if _, err := stmtEnt.Exec(uid); err != nil { query.Close(); return err }
		if _, err := stmtId.Exec(uid, ident.Name, ident.BaseTraits, ident.Age); err != nil { query.Close(); return err }

		if world.Has(ent, posID) {
			pos := (*components.Position)(world.Get(ent, posID))
			if _, err := stmtPos.Exec(uid, pos.X, pos.Y); err != nil { query.Close(); return err }
		}

		if world.Has(ent, needsID) {
			needs := (*components.Needs)(world.Get(ent, needsID))
			if _, err := stmtNeeds.Exec(uid, needs.Food, needs.Rest, needs.Safety, needs.Wealth); err != nil { query.Close(); return err }
		}

		if world.Has(ent, affID) {
			aff := (*components.Affiliation)(world.Get(ent, affID))
			if _, err := stmtAff.Exec(uid, aff.FamilyID, aff.ClanID, aff.CityID, aff.CountryID); err != nil { query.Close(); return err }
		}

		// Tags
		isVillage := world.Has(ent, villageID)
		isNPC := world.Has(ent, npcID)
		isPossessed := world.Has(ent, possessedID)
		if isVillage || isNPC || isPossessed {
			if _, err := stmtTags.Exec(uid, isVillage, isNPC, isPossessed); err != nil { query.Close(); return err }
		}

		// Storage
		if world.Has(ent, storageID) {
			store := (*components.StorageComponent)(world.Get(ent, storageID))
			if _, err := stmtStorage.Exec(uid, store.Wood, store.Stone, store.Iron, store.Food); err != nil { query.Close(); return err }
		}

		// Velocity
		if world.Has(ent, velID) {
			vel := (*components.Velocity)(world.Get(ent, velID))
			if _, err := stmtVel.Exec(uid, vel.X, vel.Y); err != nil { query.Close(); return err }
		}

		// Job
		if world.Has(ent, jobID) {
			job := (*components.JobComponent)(world.Get(ent, jobID))
			if _, err := stmtJob.Exec(uid, job.JobID, job.EmployerID); err != nil { query.Close(); return err }
		}

		// Memory
		if world.Has(ent, memID) {
			mem := (*components.Memory)(world.Get(ent, memID))
			eventsJson, err := json.Marshal(mem.Events)
			if err != nil { query.Close(); return err }
			if _, err := stmtMem.Exec(uid, string(eventsJson), mem.Head); err != nil { query.Close(); return err }
		}

		// Beliefs
		if world.Has(ent, beliefID) {
			b := (*components.BeliefComponent)(world.Get(ent, beliefID))
			bJson, err := json.Marshal(b.Beliefs)
			if err != nil { query.Close(); return err }
			if _, err := stmtBeliefs.Exec(uid, string(bJson)); err != nil { query.Close(); return err }
		}

		// Genome
		if world.Has(ent, genID) {
			g := (*components.GenomeComponent)(world.Get(ent, genID))
			if _, err := stmtGen.Exec(uid, g.Strength, g.Beauty, g.Health, g.Intellect, g.Dominant, g.Recessive); err != nil { query.Close(); return err }
		}

		// Vitals
		if world.Has(ent, vitID) {
			v := (*components.VitalsComponent)(world.Get(ent, vitID))
			if _, err := stmtVitals.Exec(uid, v.Stamina, v.Blood, v.Pain, v.Consciousness); err != nil { query.Close(); return err }
		}

		// Population
		if world.Has(ent, popID) {
			p := (*components.PopulationComponent)(world.Get(ent, popID))
			citJson, err := json.Marshal(p.Citizens)
			if err != nil { query.Close(); return err }
			if _, err := stmtPop.Exec(uid, p.Count, string(citJson)); err != nil { query.Close(); return err }
		}

		// Desperation
		if world.Has(ent, despID) {
			d := (*components.DesperationComponent)(world.Get(ent, despID))
			if _, err := stmtDesp.Exec(uid, d.Level); err != nil { query.Close(); return err }
		}

		// Secrets
		if world.Has(ent, secID) {
			s := (*components.SecretComponent)(world.Get(ent, secID))
			sJson, err := json.Marshal(s.Secrets)
			if err != nil { query.Close(); return err }
			if _, err := stmtSec.Exec(uid, string(sJson)); err != nil { query.Close(); return err }
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
	if err := rowsEnt.Err(); err != nil { return err }

	// 2. Fetch Identity
	type idData struct { name string; traits uint32; age uint16 }
	identities := make(map[uint64]idData)
	rowsId, err := db.Query("SELECT uid, name, basetraits, age FROM identity")
	if err != nil { return err }
	defer rowsId.Close()
	for rowsId.Next() {
		var u uint64
		var d idData
		if err := rowsId.Scan(&u, &d.name, &d.traits, &d.age); err != nil {
			return err
		}
		identities[u] = d
	}
	if err := rowsId.Err(); err != nil { return err }

	// 3. Fetch Position
	type posData struct { x, y float32 }
	positions := make(map[uint64]posData)
	rowsPos, err := db.Query("SELECT uid, x, y FROM position")
	if err != nil { return err }
	defer rowsPos.Close()
	for rowsPos.Next() {
		var u uint64
		var p posData
		if err := rowsPos.Scan(&u, &p.x, &p.y); err != nil {
			return err
		}
		positions[u] = p
	}
	if err := rowsPos.Err(); err != nil { return err }

	// 4. Fetch Needs
	type needsData struct { f, r, s, w float32 }
	needsMap := make(map[uint64]needsData)
	rowsNeeds, err := db.Query("SELECT uid, food, rest, safety, wealth FROM needs")
	if err != nil { return err }
	defer rowsNeeds.Close()
	for rowsNeeds.Next() {
		var u uint64
		var n needsData
		if err := rowsNeeds.Scan(&u, &n.f, &n.r, &n.s, &n.w); err != nil {
			return err
		}
		needsMap[u] = n
	}
	if err := rowsNeeds.Err(); err != nil { return err }

	// 5. Fetch Affiliation
	type affData struct { fid, cid, cityid, ctryid uint32 }
	affMap := make(map[uint64]affData)
	rowsAff, err := db.Query("SELECT uid, family_id, clan_id, city_id, country_id FROM affiliation")
	if err != nil { return err }
	defer rowsAff.Close()
	for rowsAff.Next() {
		var u uint64
		var a affData
		if err := rowsAff.Scan(&u, &a.fid, &a.cid, &a.cityid, &a.ctryid); err != nil {
			return err
		}
		affMap[u] = a
	}
	if err := rowsAff.Err(); err != nil { return err }

	// 6. Fetch Tags
	type tagsData struct { v, n, p bool }
	tagsMap := make(map[uint64]tagsData)
	rowsTags, err := db.Query("SELECT uid, is_village, is_npc, is_possessed FROM tags")
	if err != nil { return err }
	defer rowsTags.Close()
	for rowsTags.Next() {
		var u uint64
		var t tagsData
		if err := rowsTags.Scan(&u, &t.v, &t.n, &t.p); err != nil {
			return err
		}
		tagsMap[u] = t
	}
	if err := rowsTags.Err(); err != nil { return err }

	// 7. Fetch Storage
	type storeData struct { w, s, i, f uint32 }
	storeMap := make(map[uint64]storeData)
	rowsStore, err := db.Query("SELECT uid, wood, stone, iron, food FROM storage")
	if err != nil { return err }
	defer rowsStore.Close()
	for rowsStore.Next() {
		var u uint64
		var s storeData
		if err := rowsStore.Scan(&u, &s.w, &s.s, &s.i, &s.f); err != nil {
			return err
		}
		storeMap[u] = s
	}
	if err := rowsStore.Err(); err != nil { return err }

	// 8. Fetch Velocity
	type velData struct { vx, vy float32 }
	velMap := make(map[uint64]velData)
	rowsVel, err := db.Query("SELECT uid, x, y FROM velocity")
	if err != nil { return err }
	defer rowsVel.Close()
	for rowsVel.Next() {
		var u uint64
		var v velData
		if err := rowsVel.Scan(&u, &v.vx, &v.vy); err != nil {
			return err
		}
		velMap[u] = v
	}
	if err := rowsVel.Err(); err != nil { return err }

	// 9. Fetch Job
	type jobData struct { jid uint8; eid uint64 }
	jobMap := make(map[uint64]jobData)
	rowsJob, err := db.Query("SELECT uid, job_id, employer_id FROM job")
	if err != nil { return err }
	defer rowsJob.Close()
	for rowsJob.Next() {
		var u uint64
		var j jobData
		if err := rowsJob.Scan(&u, &j.jid, &j.eid); err != nil {
			return err
		}
		jobMap[u] = j
	}
	if err := rowsJob.Err(); err != nil { return err }

	// 10. Fetch Memory
	type memData struct { json string; head uint8 }
	memMap := make(map[uint64]memData)
	rowsMem, err := db.Query("SELECT uid, events_json, head FROM memory")
	if err != nil { return err }
	defer rowsMem.Close()
	for rowsMem.Next() {
		var u uint64
		var m memData
		if err := rowsMem.Scan(&u, &m.json, &m.head); err != nil {
			return err
		}
		memMap[u] = m
	}
	if err := rowsMem.Err(); err != nil { return err }

	// 11. Fetch Beliefs
	beliefsMap := make(map[uint64]string)
	rowsB, err := db.Query("SELECT uid, beliefs_json FROM beliefs")
	if err != nil { return err }
	defer rowsB.Close()
	for rowsB.Next() {
		var u uint64
		var j string
		if err := rowsB.Scan(&u, &j); err != nil {
			return err
		}
		beliefsMap[u] = j
	}
	if err := rowsB.Err(); err != nil { return err }

	// 12. Fetch Genome
	type genData struct { str, bea, hlt, itl uint8; dom, rec uint32 }
	genMap := make(map[uint64]genData)
	rowsG, err := db.Query("SELECT uid, str, bea, hlt, itl, dom, rec FROM genome")
	if err != nil { return err }
	defer rowsG.Close()
	for rowsG.Next() {
		var u uint64
		var g genData
		if err := rowsG.Scan(&u, &g.str, &g.bea, &g.hlt, &g.itl, &g.dom, &g.rec); err != nil {
			return err
		}
		genMap[u] = g
	}
	if err := rowsG.Err(); err != nil { return err }

	// 13. Fetch Vitals
	type vitData struct { s, b, p, c float32 }
	vitMap := make(map[uint64]vitData)
	rowsV, err := db.Query("SELECT uid, stamina, blood, pain, consciousness FROM vitals")
	if err != nil { return err }
	defer rowsV.Close()
	for rowsV.Next() {
		var u uint64
		var v vitData
		if err := rowsV.Scan(&u, &v.s, &v.b, &v.p, &v.c); err != nil {
			return err
		}
		vitMap[u] = v
	}
	if err := rowsV.Err(); err != nil { return err }

	// 14. Fetch Population
	type popData struct { count uint32; json string }
	popMap := make(map[uint64]popData)
	rowsP, err := db.Query("SELECT uid, count, citizens_json FROM population")
	if err != nil { return err }
	defer rowsP.Close()
	for rowsP.Next() {
		var u uint64
		var p popData
		if err := rowsP.Scan(&u, &p.count, &p.json); err != nil {
			return err
		}
		popMap[u] = p
	}
	if err := rowsP.Err(); err != nil { return err }

	// 15. Fetch Desperation
	despMap := make(map[uint64]uint8)
	rowsD, err := db.Query("SELECT uid, level FROM desperation")
	if err != nil { return err }
	defer rowsD.Close()
	for rowsD.Next() {
		var u uint64
		var l uint8
		if err := rowsD.Scan(&u, &l); err != nil {
			return err
		}
		despMap[u] = l
	}
	if err := rowsD.Err(); err != nil { return err }

	// 16. Fetch Secrets
	secMap := make(map[uint64]string)
	rowsS, err := db.Query("SELECT uid, secrets_json FROM secrets")
	if err != nil { return err }
	defer rowsS.Close()
	for rowsS.Next() {
		var u uint64
		var j string
		if err := rowsS.Scan(&u, &j); err != nil {
			return err
		}
		secMap[u] = j
	}
	if err := rowsS.Err(); err != nil { return err }

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
			if t.v { world.Add(ent, villageID) }
			if t.n { world.Add(ent, npcID) }
			if t.p { world.Add(ent, possessedID) }
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
	}

	return nil
}
