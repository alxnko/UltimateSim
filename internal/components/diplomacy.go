package components

// Grand Strategy Phase P2.2: Diplomacy
// The macro country model is a capital village entity (CapitalComponent +
// CountryComponent + Affiliation.CountryID, see systems/law_actions.go).
// DiplomacyComponent attaches to that same capital entity so the country's
// diplomatic ledger lives exactly where its treasury and war tracker already do.

// CountryRelation is one row of a country's diplomatic ledger against a single
// foreign country. Rows are kept symmetric by systems/diplomacy.go: the mirror
// row on the target's capital always carries the same Opinion/Alliance/AtWar
// state and a negated WarScore.
type CountryRelation struct {
	TruceUntil     uint64 // Absolute tick until which war declarations are blocked (0 = no truce)
	LastActionTick uint64 // Tick of the last ImproveRelations action (cooldown bookkeeping)
	TargetCountry  uint32 // CountryID of the foreign country this row describes
	Opinion        int16  // -100 (hatred) .. +100 (devotion), drifts toward 0
	WarScore       int16  // Positive = this side is winning the active war; settled at +/-100
	Alliance       bool
	AtWar          bool
	_              uint16 // Padding to maintain 8-byte alignment (32-byte row)
	_              uint32
}

// DiplomacyComponent holds every diplomatic relation of the owning country.
// The slice header is flat (24 bytes); rows are fixed-size CountryRelation
// structs appended lazily as foreign countries are discovered.
type DiplomacyComponent struct {
	Relations []CountryRelation
}
