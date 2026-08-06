package systems

import (
	"errors"
	"sort"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.3): marriage and dynasty queries.
// Pure world-mutating helpers invoked by the UI between simulation ticks.
// Marriage links two adult NPCs through DynastyComponent.SpouseID (keyed by
// Identity.ID, the hook-graph convention) and bonds them with affection hooks.

// MarriageAdultAge is the minimum age to marry (matches the FindStartCandidate
// adulthood floor in heir.go).
const MarriageAdultAge uint16 = 16

// marriageAffectionHook is the mutual hook bond sealed by a wedding.
const marriageAffectionHook = 10

var (
	ErrMarryNotAlive   = errors.New("marriage requires both parties alive with an Identity")
	ErrMarryUnderage   = errors.New("marriage requires both parties to be adults")
	ErrMarryAlreadyWed = errors.New("marriage requires both parties to be unmarried")
	ErrMarrySelf       = errors.New("cannot marry oneself")
)

// Marry weds two NPCs: both must be alive, adult, and unmarried. Sets
// SpouseID both ways, marks both Married, and seals the union with a mutual
// +10 affection hook. Family merge rule: when spouse B has no family
// (FamilyID == 0) it joins A's family, and vice versa; two rooted families
// stay distinct (the wife keeps her house). Must be called outside any open
// query loop (structural mutations).
func Marry(world *ecs.World, hooks *engine.SparseHookGraph, a, b ecs.Entity) error {
	identID := ecs.ComponentID[components.Identity](world)
	dynastyID := ecs.ComponentID[components.DynastyComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	if a == b {
		return ErrMarrySelf
	}
	if !world.Alive(a) || !world.Alive(b) || !world.Has(a, identID) || !world.Has(b, identID) {
		return ErrMarryNotAlive
	}

	identA := (*components.Identity)(world.Get(a, identID))
	identB := (*components.Identity)(world.Get(b, identID))
	if identA.Age < MarriageAdultAge || identB.Age < MarriageAdultAge {
		return ErrMarryUnderage
	}

	// Reject when either party already carries a sealed marriage.
	if world.Has(a, dynastyID) && (*components.DynastyComponent)(world.Get(a, dynastyID)).Married {
		return ErrMarryAlreadyWed
	}
	if world.Has(b, dynastyID) && (*components.DynastyComponent)(world.Get(b, dynastyID)).Married {
		return ErrMarryAlreadyWed
	}

	// Structural additions before fetching pointers: world.Add can move
	// archetype rows and invalidate previously fetched component pointers.
	if !world.Has(a, dynastyID) {
		world.Add(a, dynastyID)
	}
	if !world.Has(b, dynastyID) {
		world.Add(b, dynastyID)
	}

	// Re-fetch identities after the structural changes above.
	identA = (*components.Identity)(world.Get(a, identID))
	identB = (*components.Identity)(world.Get(b, identID))

	dynA := (*components.DynastyComponent)(world.Get(a, dynastyID))
	dynB := (*components.DynastyComponent)(world.Get(b, dynastyID))
	dynA.SpouseID = identB.ID
	dynA.Married = true
	dynB.SpouseID = identA.ID
	dynB.Married = true

	// Family merge: a rootless spouse joins the rooted family.
	if world.Has(a, affID) && world.Has(b, affID) {
		affA := (*components.Affiliation)(world.Get(a, affID))
		affB := (*components.Affiliation)(world.Get(b, affID))
		if affB.FamilyID == 0 && affA.FamilyID != 0 {
			affB.FamilyID = affA.FamilyID
		} else if affA.FamilyID == 0 && affB.FamilyID != 0 {
			affA.FamilyID = affB.FamilyID
		}
	}

	// Affection bond: +10 hooks in both directions.
	if hooks != nil {
		hooks.AddHook(identA.ID, identB.ID, marriageAffectionHook)
		hooks.AddHook(identB.ID, identA.ID, marriageAffectionHook)
	}

	return nil
}

// DynastyMember summarizes one living family member for the dynasty panel.
type DynastyMember struct {
	Entity   ecs.Entity
	ID       uint64
	SpouseID uint64
	Name     string
	Age      uint16
	JobID    uint8
	Married  bool
}

// ListDynasty lists all living NPCs of the given family with ages, jobs, and
// marital links, eldest first (Identity.ID ascending on age ties, the heir.go
// convention). Returns nil for FamilyID 0 (no family).
func ListDynasty(world *ecs.World, familyID uint32) []DynastyMember {
	if familyID == 0 {
		return nil
	}

	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	dynastyID := ecs.ComponentID[components.DynastyComponent](world)

	var members []DynastyMember
	q := world.Query(ecs.All(npcID, identID, affID))
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(affID))
		if aff.FamilyID != familyID {
			continue
		}
		ident := (*components.Identity)(q.Get(identID))
		m := DynastyMember{Entity: q.Entity(), ID: ident.ID, Name: ident.Name, Age: ident.Age}
		if q.Has(jobID) {
			m.JobID = (*components.JobComponent)(q.Get(jobID)).JobID
		}
		if q.Has(dynastyID) {
			dyn := (*components.DynastyComponent)(q.Get(dynastyID))
			m.SpouseID = dyn.SpouseID
			m.Married = dyn.Married
		}
		members = append(members, m)
	}
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].Age != members[j].Age {
			return members[i].Age > members[j].Age
		}
		return members[i].ID < members[j].ID
	})
	return members
}
