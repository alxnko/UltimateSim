package systems

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: Player Social Actions
// Pure world-mutating functions invoked by the UI between simulation ticks.
// These reuse the existing systemic primitives (SparseHookGraph, Memory ring
// buffers, Jurisdiction law bitmasks, translation penalty convention) so that
// player-driven actions emerge through the exact same channels as NPC actions.

// Chat reply lines, selected deterministically by the target's lowest Need.
const (
	chatLineStarving = "I'm starving..."
	chatLineTired    = "I can barely keep my eyes open..."
	chatLineScared   = "I don't feel safe around here..."
	chatLinePoor     = "I haven't a coin to my name..."
	chatLineContent  = "Life treats me well these days."
)

// chatNeedSatisfied is the threshold above which a Need is considered "fine".
const chatNeedSatisfied float32 = 50.0

// threatenSafetyDamage is how much Safety a threat strips from the target.
const threatenSafetyDamage float32 = 10.0

// socialEntityID resolves the social-graph ID used by the SparseHookGraph.
// Hooks are keyed by Identity.ID (gossip/justice convention); entities
// without an Identity fall back to their raw ECS entity ID.
func socialEntityID(world *ecs.World, e ecs.Entity, identID ecs.ID) uint64 {
	if world.Has(e, identID) {
		return (*components.Identity)(world.Get(e, identID)).ID
	}
	return uint64(e.ID())
}

// appendSocialMemory writes an event into a Memory ring buffer and advances
// the head with wrap-around at 50 (GossipDistributionSystem convention).
func appendSocialMemory(mem *components.Memory, event components.MemoryEvent) {
	mem.Events[mem.Head] = event
	mem.Head = (mem.Head + 1) % 50
}

// chatLineForNeeds deterministically picks a reply line from the lowest Need.
// Ties break in fixed priority order: Food > Rest > Safety > Wealth.
func chatLineForNeeds(needs *components.Needs) string {
	lowest := needs.Food
	line := chatLineStarving
	if needs.Rest < lowest {
		lowest = needs.Rest
		line = chatLineTired
	}
	if needs.Safety < lowest {
		lowest = needs.Safety
		line = chatLineScared
	}
	if needs.Wealth < lowest {
		lowest = needs.Wealth
		line = chatLinePoor
	}
	if lowest >= chatNeedSatisfied {
		return chatLineContent
	}
	return line
}

// Chat performs friendly small talk between the player and a target NPC.
// Both parties gain a +1 hook on each other and both Memory rings (if
// present) record an InteractionGossip event. The returned reply string is
// "<Name>: " plus a line chosen by the target's lowest Need. Deterministic,
// no RNG.
func Chat(world *ecs.World, hooks *engine.SparseHookGraph, player, target ecs.Entity, tick uint64) string {
	identID := ecs.ComponentID[components.Identity](world)
	memoryID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)

	playerID := socialEntityID(world, player, identID)
	targetID := socialEntityID(world, target, identID)

	// Mutual small-talk bond: +1 hook in both directions.
	if hooks != nil {
		hooks.AddHook(playerID, targetID, 1)
		hooks.AddHook(targetID, playerID, 1)
	}

	// Memory metadata stores the speaker's tongue (gossip convention).
	var playerLang, targetLang uint16
	if world.Has(player, cultureID) {
		playerLang = (*components.CultureComponent)(world.Get(player, cultureID)).LanguageID
	}
	if world.Has(target, cultureID) {
		targetLang = (*components.CultureComponent)(world.Get(target, cultureID)).LanguageID
	}

	// Record the conversation in BOTH Memory rings, if present.
	if world.Has(player, memoryID) {
		mem := (*components.Memory)(world.Get(player, memoryID))
		appendSocialMemory(mem, components.MemoryEvent{
			TargetID:        targetID,
			TickStamp:       tick,
			InteractionType: components.InteractionGossip,
			LanguageID:      targetLang,
		})
	}
	if world.Has(target, memoryID) {
		mem := (*components.Memory)(world.Get(target, memoryID))
		appendSocialMemory(mem, components.MemoryEvent{
			TargetID:        playerID,
			TickStamp:       tick,
			InteractionType: components.InteractionGossip,
			LanguageID:      playerLang,
		})
	}

	// Build the reply from the target's identity and lowest Need.
	name := "Stranger"
	if world.Has(target, identID) {
		name = (*components.Identity)(world.Get(target, identID)).Name
	}
	line := chatLineContent
	if world.Has(target, needsID) {
		line = chatLineForNeeds((*components.Needs)(world.Get(target, needsID)))
	}
	return name + ": " + line
}

// GiveGift transfers wealth from the player to the target. The recipient
// remembers the gift and places a +3 hook on the player (they owe you).
// Errors if the player cannot afford the amount; no state changes on error.
func GiveGift(world *ecs.World, hooks *engine.SparseHookGraph, player, target ecs.Entity, amount float32) error {
	identID := ecs.ComponentID[components.Identity](world)
	memoryID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)

	if amount < 0 {
		return fmt.Errorf("gift amount must be non-negative, got %.2f", amount)
	}
	if !world.Has(player, needsID) {
		return fmt.Errorf("giver has no Needs component to draw wealth from")
	}
	if !world.Has(target, needsID) {
		return fmt.Errorf("recipient has no Needs component to receive wealth")
	}

	playerNeeds := (*components.Needs)(world.Get(player, needsID))
	if playerNeeds.Wealth < amount {
		return fmt.Errorf("insufficient wealth: have %.2f, need %.2f", playerNeeds.Wealth, amount)
	}

	// Execute the transfer.
	targetNeeds := (*components.Needs)(world.Get(target, needsID))
	playerNeeds.Wealth -= amount
	targetNeeds.Wealth += amount

	playerID := socialEntityID(world, player, identID)
	targetID := socialEntityID(world, target, identID)

	// The recipient owes the player: +3 hook target -> player.
	if hooks != nil {
		hooks.AddHook(targetID, playerID, 3)
	}

	// The recipient remembers the generosity.
	if world.Has(target, memoryID) {
		mem := (*components.Memory)(world.Get(target, memoryID))
		appendSocialMemory(mem, components.MemoryEvent{
			TargetID:        playerID,
			InteractionType: components.InteractionGossip,
			Value:           int32(amount),
		})
	}

	return nil
}

// Threaten intimidates the target: -2 hook target -> player, target Safety
// drops by 10 (floored at 0), and the target remembers an assault. If the
// player stands inside a Jurisdiction whose laws flag InteractionAssault as
// illegal, a CrimeMarker{CrimeLevel: 1, Bounty: 10} is attached to the player
// (structural mutation applied strictly outside the query loop). Returns
// whether a crime was committed.
func Threaten(world *ecs.World, hooks *engine.SparseHookGraph, player, target ecs.Entity, tick uint64) bool {
	identID := ecs.ComponentID[components.Identity](world)
	memoryID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)
	posID := ecs.ComponentID[components.Position](world)
	jurisdictionID := ecs.ComponentID[components.JurisdictionComponent](world)
	crimeID := ecs.ComponentID[components.CrimeMarker](world)

	playerID := socialEntityID(world, player, identID)
	targetID := socialEntityID(world, target, identID)

	// The target resents the player: -2 hook target -> player.
	if hooks != nil {
		hooks.AddHook(targetID, playerID, -2)
	}

	// Fear response: Safety drops, floored at 0.
	if world.Has(target, needsID) {
		targetNeeds := (*components.Needs)(world.Get(target, needsID))
		targetNeeds.Safety -= threatenSafetyDamage
		if targetNeeds.Safety < 0 {
			targetNeeds.Safety = 0
		}
	}

	// The target remembers the assault (the JusticeSystem reads this too).
	if world.Has(target, memoryID) {
		mem := (*components.Memory)(world.Get(target, memoryID))
		appendSocialMemory(mem, components.MemoryEvent{
			TargetID:        playerID,
			TickStamp:       tick,
			InteractionType: components.InteractionAssault,
		})
	}

	// Witness check: was the threat made inside a Jurisdiction outlawing assault?
	crimeCommitted := false
	if world.Has(player, posID) {
		playerPos := (*components.Position)(world.Get(player, posID))
		px, py := playerPos.X, playerPos.Y

		jurMask := ecs.All(jurisdictionID, posID)
		jurQuery := world.Query(&jurMask)
		for jurQuery.Next() {
			jur := (*components.JurisdictionComponent)(jurQuery.Get(jurisdictionID))
			jurPos := (*components.Position)(jurQuery.Get(posID))

			dx := px - jurPos.X
			dy := py - jurPos.Y
			distSq := (dx * dx) + (dy * dy)

			if distSq <= jur.RadiusSquared && (jur.IllegalActionIDs&(1<<components.InteractionAssault)) != 0 {
				crimeCommitted = true
				jurQuery.Close() // Early exit requires closing the query to unlock the world
				break
			}
		}
	}

	// Structural mutation strictly OUTSIDE the query loop.
	if crimeCommitted {
		if !world.Has(player, crimeID) {
			world.Add(player, crimeID)
			cm := (*components.CrimeMarker)(world.Get(player, crimeID))
			cm.CrimeLevel = 1
			cm.Bounty = 10
		} else {
			// Repeat offense: stack the bounty on the existing marker.
			cm := (*components.CrimeMarker)(world.Get(player, crimeID))
			if cm.CrimeLevel < 1 {
				cm.CrimeLevel = 1
			}
			cm.Bounty += 10
		}
	}

	return crimeCommitted
}

// ShareRumor passes the first secret the player holds that the target lacks.
// If both parties carry a CultureComponent and their LanguageIDs mismatch,
// transfer succeeds only on a 10% deterministic RNG gate (the Phase 07.4
// translation penalty convention). On success the secret is appended to the
// target's SecretComponent, creating the component if missing (structural
// mutation outside any query). Returns whether the rumor was transferred.
func ShareRumor(world *ecs.World, player, target ecs.Entity) bool {
	secretID := ecs.ComponentID[components.SecretComponent](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)

	if !world.Has(player, secretID) {
		return false
	}
	playerSecrets := (*components.SecretComponent)(world.Get(player, secretID))
	if len(playerSecrets.Secrets) == 0 {
		return false
	}

	var targetSecrets *components.SecretComponent
	if world.Has(target, secretID) {
		targetSecrets = (*components.SecretComponent)(world.Get(target, secretID))
	}

	// Pick the first secret the target does not already know.
	chosenIdx := -1
	for i := range playerSecrets.Secrets {
		alreadyKnown := false
		if targetSecrets != nil {
			for j := range targetSecrets.Secrets {
				if targetSecrets.Secrets[j].SecretID == playerSecrets.Secrets[i].SecretID {
					alreadyKnown = true
					break
				}
			}
		}
		if !alreadyKnown {
			chosenIdx = i
			break
		}
	}
	if chosenIdx == -1 {
		return false // Nothing new to share
	}

	// Translation penalty gate: mismatched languages leave a 10% window.
	if world.Has(player, cultureID) && world.Has(target, cultureID) {
		playerCulture := (*components.CultureComponent)(world.Get(player, cultureID))
		targetCulture := (*components.CultureComponent)(world.Get(target, cultureID))
		if playerCulture.LanguageID != targetCulture.LanguageID {
			if engine.GetRandomInt()%10 != 0 {
				return false
			}
		}
	}

	// Copy the secret by value BEFORE any structural change: world.Add can
	// move archetype rows and invalidate previously fetched pointers.
	chosen := playerSecrets.Secrets[chosenIdx]

	if !world.Has(target, secretID) {
		world.Add(target, secretID) // Outside any query: safe structural mutation
	}
	freshTargetSecrets := (*components.SecretComponent)(world.Get(target, secretID))
	freshTargetSecrets.Secrets = append(freshTargetSecrets.Secrets, components.Secret{
		OriginID: chosen.OriginID,
		SecretID: chosen.SecretID,
		Virality: chosen.Virality,
		BeliefID: chosen.BeliefID,
	})

	return true
}
