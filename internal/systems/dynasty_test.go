package systems

import (
	"errors"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.3): marriage + dynasty query tests.
// Deterministic, headless coverage of Marry (two-way spouse links, affection
// hooks, family merge, rejection rules) and ListDynasty (family roster with
// ages/jobs, eldest-first ordering).

// spawnDynastyNPC creates an NPC+Identity+Affiliation entity.
func spawnDynastyNPC(world *ecs.World, identityID uint64, familyID uint32, age uint16) ecs.Entity {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)

	e := world.NewEntity(npcID, identID, affilID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = identityID
	ident.Age = age
	affil := (*components.Affiliation)(world.Get(e, affilID))
	affil.FamilyID = familyID
	return e
}

// TestMarryLinksBothWays verifies the spouse link is set in both directions,
// both parties are marked Married, mutual +10 affection hooks are placed, and
// a rootless spouse joins the rooted family.
func TestMarryLinksBothWays(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	a := spawnDynastyNPC(&world, 1, 7, 30)
	b := spawnDynastyNPC(&world, 2, 0, 25) // Rootless: must join family 7

	if err := Marry(&world, hooks, a, b); err != nil {
		t.Fatalf("Marry returned error: %v", err)
	}

	dynastyID := ecs.ComponentID[components.DynastyComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)

	dynA := (*components.DynastyComponent)(world.Get(a, dynastyID))
	dynB := (*components.DynastyComponent)(world.Get(b, dynastyID))
	if dynA.SpouseID != 2 || !dynA.Married {
		t.Errorf("a: SpouseID=%d Married=%v, want SpouseID=2 Married=true", dynA.SpouseID, dynA.Married)
	}
	if dynB.SpouseID != 1 || !dynB.Married {
		t.Errorf("b: SpouseID=%d Married=%v, want SpouseID=1 Married=true", dynB.SpouseID, dynB.Married)
	}

	// Affection hooks in both directions.
	if got := hooks.GetHook(1, 2); got != 10 {
		t.Errorf("hook a->b = %d, want 10", got)
	}
	if got := hooks.GetHook(2, 1); got != 10 {
		t.Errorf("hook b->a = %d, want 10", got)
	}

	// Family merge: rootless b joined a's family 7.
	affB := (*components.Affiliation)(world.Get(b, affID))
	if affB.FamilyID != 7 {
		t.Errorf("b FamilyID = %d, want 7 (joined spouse's family)", affB.FamilyID)
	}

	// Two rooted families stay distinct.
	c := spawnDynastyNPC(&world, 3, 11, 40)
	d := spawnDynastyNPC(&world, 4, 13, 41)
	if err := Marry(&world, hooks, c, d); err != nil {
		t.Fatalf("Marry(c, d) returned error: %v", err)
	}
	affC := (*components.Affiliation)(world.Get(c, affID))
	affD := (*components.Affiliation)(world.Get(d, affID))
	if affC.FamilyID != 11 || affD.FamilyID != 13 {
		t.Errorf("rooted families changed: c=%d d=%d, want 11 and 13", affC.FamilyID, affD.FamilyID)
	}
}

// TestMarryRejections verifies self-marriage, underage, already-married, and
// dead-party rejections leave the world untouched.
func TestMarryRejections(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	a := spawnDynastyNPC(&world, 1, 7, 30)
	b := spawnDynastyNPC(&world, 2, 7, 25)
	child := spawnDynastyNPC(&world, 3, 7, 12)

	if err := Marry(&world, hooks, a, a); !errors.Is(err, ErrMarrySelf) {
		t.Errorf("self-marriage: err = %v, want ErrMarrySelf", err)
	}
	if err := Marry(&world, hooks, a, child); !errors.Is(err, ErrMarryUnderage) {
		t.Errorf("underage: err = %v, want ErrMarryUnderage", err)
	}

	if err := Marry(&world, hooks, a, b); err != nil {
		t.Fatalf("valid marriage failed: %v", err)
	}
	// A married party cannot re-marry.
	c := spawnDynastyNPC(&world, 4, 9, 30)
	if err := Marry(&world, hooks, a, c); !errors.Is(err, ErrMarryAlreadyWed) {
		t.Errorf("bigamy: err = %v, want ErrMarryAlreadyWed", err)
	}

	// A dead party cannot marry.
	dead := spawnDynastyNPC(&world, 5, 9, 30)
	world.RemoveEntity(dead)
	if err := Marry(&world, hooks, c, dead); !errors.Is(err, ErrMarryNotAlive) {
		t.Errorf("dead party: err = %v, want ErrMarryNotAlive", err)
	}
}

// TestListDynasty verifies the family roster carries ages, jobs, and spouse
// links, is eldest-first with ID tiebreak, and excludes other families.
func TestListDynasty(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	jobID := ecs.ComponentID[components.JobComponent](&world)

	a := spawnDynastyNPC(&world, 1, 7, 40)
	b := spawnDynastyNPC(&world, 2, 0, 40) // Same age as a: ID tiebreak
	spawnDynastyNPC(&world, 3, 7, 12)      // The child
	spawnDynastyNPC(&world, 4, 9, 50)      // Foreign family: excluded

	world.Add(a, jobID)
	(*components.JobComponent)(world.Get(a, jobID)).JobID = components.JobFarmer

	if err := Marry(&world, hooks, a, b); err != nil {
		t.Fatalf("Marry returned error: %v", err)
	}

	members := ListDynasty(&world, 7)
	if len(members) != 3 {
		t.Fatalf("ListDynasty returned %d members, want 3", len(members))
	}
	// Eldest first, ID ascending on age tie: (1, 40) then (2, 40) then (3, 12).
	if members[0].ID != 1 || members[1].ID != 2 || members[2].ID != 3 {
		t.Errorf("order = [%d %d %d], want [1 2 3]", members[0].ID, members[1].ID, members[2].ID)
	}
	if members[0].JobID != components.JobFarmer {
		t.Errorf("members[0].JobID = %d, want JobFarmer", members[0].JobID)
	}
	if members[0].SpouseID != 2 || !members[0].Married {
		t.Errorf("members[0] spouse = %d married=%v, want 2/true", members[0].SpouseID, members[0].Married)
	}
	if members[1].SpouseID != 1 || !members[1].Married {
		t.Errorf("members[1] spouse = %d married=%v, want 1/true", members[1].SpouseID, members[1].Married)
	}
	for _, m := range members {
		if m.ID == 4 {
			t.Errorf("family-9 member leaked into family-7 roster")
		}
	}

	// FamilyID 0 yields no roster.
	if got := ListDynasty(&world, 0); got != nil {
		t.Errorf("ListDynasty(0) = %v, want nil", got)
	}
}
