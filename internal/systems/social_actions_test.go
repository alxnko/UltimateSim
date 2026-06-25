package systems

import (
	"strings"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: Player Social Actions E2E tests.
// Deterministic and headless: go test ./internal/systems/ -run TestSocialActions -count=2
//
// Every test seeds the global RNG with a fixed seed so that any RNG-gated
// branch (ShareRumor's translation penalty) resolves identically on every run.

// fixedSeed is an arbitrary, fixed 32-byte seed for deterministic RNG.
var fixedSeed = [32]byte{
	1, 2, 3, 4, 5, 6, 7, 8,
	9, 10, 11, 12, 13, 14, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24,
	25, 26, 27, 28, 29, 30, 31, 32,
}

// spawnSocialActor builds an entity with Identity+Memory+Needs+CultureComponent.
// languageID drives the CultureComponent; needs sets the four Needs fields.
func spawnSocialActor(world *ecs.World, identityID uint64, name string, languageID uint16, food, rest, safety, wealth float32) ecs.Entity {
	identID := ecs.ComponentID[components.Identity](world)
	memoryID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)

	e := world.NewEntity(identID, memoryID, needsID, cultureID)

	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = identityID
	ident.Name = name

	needs := (*components.Needs)(world.Get(e, needsID))
	needs.Food = food
	needs.Rest = rest
	needs.Safety = safety
	needs.Wealth = wealth

	culture := (*components.CultureComponent)(world.Get(e, cultureID))
	culture.LanguageID = languageID

	return e
}

// TestSocialActionsChat verifies the mutual small-talk bond: +1 hook both
// directions, both Memory ring heads advance, and the reply starts with the
// target's name.
func TestSocialActionsChat(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	memoryID := ecs.ComponentID[components.Memory](&world)

	// Same language for both. Target's lowest need keeps the reply deterministic
	// (all needs are high here, so the content line is used; only the name prefix
	// is asserted).
	player := spawnSocialActor(&world, 100, "Player", 1, 80, 80, 80, 80)
	target := spawnSocialActor(&world, 200, "Brannoc", 1, 90, 90, 90, 90)

	playerHeadBefore := (*components.Memory)(world.Get(player, memoryID)).Head
	targetHeadBefore := (*components.Memory)(world.Get(target, memoryID)).Head

	reply := Chat(&world, hooks, player, target, 42)

	// Mutual +1 hook in both directions.
	if got := hooks.GetHook(100, 200); got != 1 {
		t.Errorf("player->target hook = %d, want 1", got)
	}
	if got := hooks.GetHook(200, 100); got != 1 {
		t.Errorf("target->player hook = %d, want 1", got)
	}

	// Both Memory ring heads advanced by exactly one slot.
	playerHeadAfter := (*components.Memory)(world.Get(player, memoryID)).Head
	targetHeadAfter := (*components.Memory)(world.Get(target, memoryID)).Head
	if playerHeadAfter != (playerHeadBefore+1)%50 {
		t.Errorf("player Memory head = %d, want %d", playerHeadAfter, (playerHeadBefore+1)%50)
	}
	if targetHeadAfter != (targetHeadBefore+1)%50 {
		t.Errorf("target Memory head = %d, want %d", targetHeadAfter, (targetHeadBefore+1)%50)
	}

	// Reply starts with the target's name.
	if !strings.HasPrefix(reply, "Brannoc: ") {
		t.Errorf("reply = %q, want prefix %q", reply, "Brannoc: ")
	}
}

// TestSocialActionsChatNeedLine verifies the reply line tracks the target's
// lowest Need (white-box check of chatLineForNeeds wiring through Chat).
func TestSocialActionsChatNeedLine(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	player := spawnSocialActor(&world, 100, "Player", 1, 80, 80, 80, 80)
	// Food is the lowest and below the satisfied threshold -> starving line.
	target := spawnSocialActor(&world, 200, "Maeve", 1, 5, 90, 90, 90)

	reply := Chat(&world, hooks, player, target, 1)
	want := "Maeve: " + chatLineStarving
	if reply != want {
		t.Errorf("reply = %q, want %q", reply, want)
	}
}

// TestSocialActionsGiveGift verifies a successful transfer, the +3 recipient
// hook, and that insufficient funds error out with no state change.
func TestSocialActionsGiveGift(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	needsID := ecs.ComponentID[components.Needs](&world)

	player := spawnSocialActor(&world, 100, "Player", 1, 50, 50, 50, 50)
	target := spawnSocialActor(&world, 200, "Cael", 1, 50, 50, 50, 0)

	// Successful gift of 10: player 50 -> 40, target 0 -> 10.
	if err := GiveGift(&world, hooks, player, target, 10); err != nil {
		t.Fatalf("GiveGift returned error: %v", err)
	}
	playerNeeds := (*components.Needs)(world.Get(player, needsID))
	targetNeeds := (*components.Needs)(world.Get(target, needsID))
	if playerNeeds.Wealth != 40 {
		t.Errorf("player wealth = %f, want 40", playerNeeds.Wealth)
	}
	if targetNeeds.Wealth != 10 {
		t.Errorf("target wealth = %f, want 10", targetNeeds.Wealth)
	}
	// Recipient owes the player: +3 hook target -> player.
	if got := hooks.GetHook(200, 100); got != 3 {
		t.Errorf("target->player hook = %d, want 3", got)
	}
	// The giver does not gain a hook toward the recipient from a gift.
	if got := hooks.GetHook(100, 200); got != 0 {
		t.Errorf("player->target hook = %d, want 0", got)
	}

	// Insufficient funds: player only has 40 now; gifting 100 must error and
	// leave both balances and the hook untouched.
	if err := GiveGift(&world, hooks, player, target, 100); err == nil {
		t.Errorf("GiveGift with insufficient funds = nil error, want error")
	}
	playerNeeds = (*components.Needs)(world.Get(player, needsID))
	targetNeeds = (*components.Needs)(world.Get(target, needsID))
	if playerNeeds.Wealth != 40 {
		t.Errorf("player wealth after failed gift = %f, want 40 (no change)", playerNeeds.Wealth)
	}
	if targetNeeds.Wealth != 10 {
		t.Errorf("target wealth after failed gift = %f, want 10 (no change)", targetNeeds.Wealth)
	}
	if got := hooks.GetHook(200, 100); got != 3 {
		t.Errorf("target->player hook after failed gift = %d, want 3 (no change)", got)
	}
}

// TestSocialActionsThreatenNoJurisdiction verifies that a threat outside any
// jurisdiction returns false, drops target Safety by 10, and applies a -2 hook.
func TestSocialActionsThreatenNoJurisdiction(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	needsID := ecs.ComponentID[components.Needs](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	player := spawnSocialActor(&world, 100, "Player", 1, 50, 50, 50, 50)
	target := spawnSocialActor(&world, 200, "Doran", 1, 50, 50, 50, 50)

	crime := Threaten(&world, hooks, player, target, 7)
	if crime {
		t.Errorf("Threaten outside any jurisdiction = true, want false")
	}

	// Safety dropped by 10 (50 -> 40).
	targetNeeds := (*components.Needs)(world.Get(target, needsID))
	if targetNeeds.Safety != 40 {
		t.Errorf("target Safety = %f, want 40", targetNeeds.Safety)
	}
	// -2 hook target -> player.
	if got := hooks.GetHook(200, 100); got != -2 {
		t.Errorf("target->player hook = %d, want -2", got)
	}
	// No crime, so no marker.
	if world.Has(player, crimeID) {
		t.Errorf("player gained a CrimeMarker outside any jurisdiction")
	}
}

// TestSocialActionsThreatenSafetyFloor verifies Safety is floored at 0.
func TestSocialActionsThreatenSafetyFloor(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	needsID := ecs.ComponentID[components.Needs](&world)

	player := spawnSocialActor(&world, 100, "Player", 1, 50, 50, 50, 50)
	target := spawnSocialActor(&world, 200, "Edda", 1, 50, 50, 5, 50)

	Threaten(&world, hooks, player, target, 1)
	targetNeeds := (*components.Needs)(world.Get(target, needsID))
	if targetNeeds.Safety != 0 {
		t.Errorf("target Safety = %f, want 0 (floored)", targetNeeds.Safety)
	}
}

// TestSocialActionsThreatenInsideJurisdiction verifies that a threat inside a
// jurisdiction that outlaws assault returns true and attaches a CrimeMarker.
func TestSocialActionsThreatenInsideJurisdiction(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	identID := ecs.ComponentID[components.Identity](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	jurisdictionID := ecs.ComponentID[components.JurisdictionComponent](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	// Player with Identity+Memory+Needs+Culture AND a Position inside the jurisdiction.
	player := world.NewEntity(identID, memoryID, needsID, cultureID, posID)
	pIdent := (*components.Identity)(world.Get(player, identID))
	pIdent.ID = 100
	pIdent.Name = "Player"
	pPos := (*components.Position)(world.Get(player, posID))
	pPos.X = 3
	pPos.Y = 4 // distance^2 from origin = 25 <= 100

	target := spawnSocialActor(&world, 200, "Faolan", 1, 50, 50, 50, 50)

	// Jurisdiction at the origin: radius^2 = 100, assault is illegal.
	jur := world.NewEntity(jurisdictionID, posID)
	jc := (*components.JurisdictionComponent)(world.Get(jur, jurisdictionID))
	jc.RadiusSquared = 100
	jc.IllegalActionIDs = 1 << components.InteractionAssault
	jPos := (*components.Position)(world.Get(jur, posID))
	jPos.X = 0
	jPos.Y = 0

	crime := Threaten(&world, hooks, player, target, 9)
	if !crime {
		t.Fatalf("Threaten inside an assault-outlawing jurisdiction = false, want true")
	}
	if !world.Has(player, crimeID) {
		t.Fatalf("player should have gained a CrimeMarker")
	}
	cm := (*components.CrimeMarker)(world.Get(player, crimeID))
	if cm.CrimeLevel != 1 || cm.Bounty != 10 {
		t.Errorf("CrimeMarker = {CrimeLevel:%d Bounty:%d}, want {1 10}", cm.CrimeLevel, cm.Bounty)
	}

	// The -2 hook and Safety damage still apply alongside the crime.
	targetNeeds := (*components.Needs)(world.Get(target, needsID))
	if targetNeeds.Safety != 40 {
		t.Errorf("target Safety = %f, want 40", targetNeeds.Safety)
	}
	if got := hooks.GetHook(200, 100); got != -2 {
		t.Errorf("target->player hook = %d, want -2", got)
	}
}

// TestSocialActionsShareRumorSameLanguage verifies that with matching languages
// the first unknown secret always transfers to the target.
func TestSocialActionsShareRumorSameLanguage(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()

	secretID := ecs.ComponentID[components.SecretComponent](&world)

	player := spawnSocialActor(&world, 100, "Player", 1, 50, 50, 50, 50)
	target := spawnSocialActor(&world, 200, "Gwen", 1, 50, 50, 50, 50)

	// Player knows one secret; target knows none.
	world.Add(player, secretID)
	pSecrets := (*components.SecretComponent)(world.Get(player, secretID))
	pSecrets.Secrets = []components.Secret{{OriginID: 100, SecretID: 7, Virality: 3, BeliefID: 0}}

	if !ShareRumor(&world, player, target) {
		t.Fatalf("same-language ShareRumor = false, want true")
	}
	if !world.Has(target, secretID) {
		t.Fatalf("target should have gained a SecretComponent")
	}
	tSecrets := (*components.SecretComponent)(world.Get(target, secretID))
	if len(tSecrets.Secrets) != 1 || tSecrets.Secrets[0].SecretID != 7 {
		t.Fatalf("target secrets = %+v, want one secret with SecretID 7", tSecrets.Secrets)
	}

	// Sharing again transfers nothing new (the target already knows secret 7).
	if ShareRumor(&world, player, target) {
		t.Errorf("re-sharing an already-known secret = true, want false")
	}
	tSecrets = (*components.SecretComponent)(world.Get(target, secretID))
	if len(tSecrets.Secrets) != 1 {
		t.Errorf("target secrets count = %d, want 1 (no duplicate)", len(tSecrets.Secrets))
	}
}

// TestSocialActionsShareRumorNoSecret verifies that a player holding no secret
// transfers nothing.
func TestSocialActionsShareRumorNoSecret(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()

	player := spawnSocialActor(&world, 100, "Player", 1, 50, 50, 50, 50)
	target := spawnSocialActor(&world, 200, "Hale", 1, 50, 50, 50, 50)

	if ShareRumor(&world, player, target) {
		t.Errorf("ShareRumor with no player secret = true, want false")
	}
}

// TestSocialActionsShareRumorCrossLanguage verifies that the RNG-gated
// cross-language path does not panic and returns a bool deterministically.
func TestSocialActionsShareRumorCrossLanguage(t *testing.T) {
	engine.InitializeRNG(fixedSeed)

	world := ecs.NewWorld()

	secretID := ecs.ComponentID[components.SecretComponent](&world)

	// Distinct languages -> the 10% translation gate applies.
	player := spawnSocialActor(&world, 100, "Player", 1, 50, 50, 50, 50)
	target := spawnSocialActor(&world, 200, "Isolde", 2, 50, 50, 50, 50)

	world.Add(player, secretID)
	pSecrets := (*components.SecretComponent)(world.Get(player, secretID))
	pSecrets.Secrets = []components.Secret{{OriginID: 100, SecretID: 11, Virality: 1, BeliefID: 0}}

	// We only assert it does not panic and returns a bool; the gate is RNG-driven.
	transferred := ShareRumor(&world, player, target)

	// If it claims success, the secret must actually be present; if it claims
	// failure, the target must not have learned it. This keeps the contract
	// consistent regardless of which branch the seeded RNG took.
	if transferred {
		if !world.Has(target, secretID) {
			t.Errorf("ShareRumor returned true but target has no SecretComponent")
		} else {
			tSecrets := (*components.SecretComponent)(world.Get(target, secretID))
			if len(tSecrets.Secrets) != 1 || tSecrets.Secrets[0].SecretID != 11 {
				t.Errorf("ShareRumor returned true but secret 11 not present: %+v", tSecrets.Secrets)
			}
		}
	} else if world.Has(target, secretID) {
		tSecrets := (*components.SecretComponent)(world.Get(target, secretID))
		if len(tSecrets.Secrets) != 0 {
			t.Errorf("ShareRumor returned false but target gained secrets: %+v", tSecrets.Secrets)
		}
	}
}
