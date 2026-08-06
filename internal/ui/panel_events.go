package ui

// Grand Strategy Phase G1/G5: the interactive event popup and queue badge.
// DrawEventPopup renders the newest pending event as a ModalFrame with 1-3
// choice buttons; choosing funnels through systems.ResolveEvent and removes
// the event from the queue. While the popup is up the sim is frozen CK-style
// (state_playing gates its tick loop on EventPopupActive).
// Spec: docs/superpowers/specs/2026-08-06-grand-strategy-playable.md

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Popup state lives at package level because PlayerContext is owned by the
// orchestrator. Single Ebiten goroutine — no locking.
var (
	eventPopupOpen bool
	eventSeenCount int
)

// EventsPending returns how many interactive events await the player.
func EventsPending(pc *PlayerContext) int {
	player, ok := pc.PossessedEntity()
	if !ok {
		return 0
	}
	return len(systems.PendingEventList(pc.Status.TM.World, player))
}

// EventPopupActive reports whether the event popup is showing an event.
// state_playing freezes the sim while this is true (pause-on-event, spec G5).
func EventPopupActive(pc *PlayerContext) bool {
	return eventPopupOpen && EventsPending(pc) > 0
}

// DrawEventPopup renders the pending-events badge and, when open, the event
// modal. Call it late in the Draw chain (after the other panels, before the
// context menus and FlushTooltips).
func (s *StatePlaying) DrawEventPopup(screen *ebiten.Image) {
	pc := s.PC
	world := s.Status.TM.World
	player, ok := pc.PossessedEntity()
	if !ok {
		eventPopupOpen = false
		eventSeenCount = 0
		return
	}
	events := systems.PendingEventList(world, player)
	n := len(events)
	if n > eventSeenCount {
		eventPopupOpen = true // CK-style: a new event interrupts play
	}
	eventSeenCount = n
	if n == 0 {
		eventPopupOpen = false
		return
	}

	sh := screen.Bounds().Dy()
	if !eventPopupOpen {
		// Queue badge above the HUD: click to reopen the popup.
		label := fmt.Sprintf("! %d event", n)
		if n > 1 {
			label += "s"
		}
		if Button(screen, label, PadM, sh-HUDHeight-28, 110, 22) {
			eventPopupOpen = true
		}
		return
	}

	ev := events[n-1] // Newest pending event
	title := eventTitle(ev.Kind)
	body := s.eventBody(&ev)
	choices := eventChoices(&ev)

	w := 440
	bodyTop := len(body) * 18
	h := ModalTitleH + PadM + bodyTop + PadM + 26 + PadM + 16 + PadM
	cx, cy, closed := ModalFrame(screen, title, w, h)
	if closed {
		eventPopupOpen = false
		return
	}
	for i, ln := range body {
		DrawText(screen, ln, cx, cy+i*18, TextCol)
	}

	bx, by := cx, cy+bodyTop+PadM
	for i, label := range choices {
		bw := MeasureText(label) + 2*PadL
		if bw < 92 {
			bw = 92
		}
		if Button(screen, label, bx, by, bw, 26) {
			result := systems.ResolveEvent(world, s.Status.HookGraph, player, &ev, i)
			systems.RemovePendingEvent(world, player, ev.ID)
			if result != "" {
				pc.PushNote(result, s.Status.TM.Ticks)
			}
			eventSeenCount = n - 1
			if eventSeenCount == 0 {
				eventPopupOpen = false
			}
			return
		}
		bx += bw + PadM
	}
	if n > 1 {
		DrawSmall(screen, fmt.Sprintf("%d more waiting...", n-1), cx, by+26+PadS, TextDim)
	}
}

// eventTitle maps an event kind to its popup title.
func eventTitle(kind uint8) string {
	switch kind {
	case components.EventTaxDemand:
		return "A Demand from the Throne"
	case components.EventBanditShakedown:
		return "Roadside Ambush"
	case components.EventJobOffer:
		return "An Offer of Work"
	case components.EventInsult:
		return "A Public Insult"
	case components.EventFestival:
		return "Festival!"
	case components.EventWarNews:
		return "War!"
	case components.EventPeaceNews:
		return "Peace"
	case components.EventRulerDied:
		return "A Ruler Is Dead"
	default:
		return "Something Happens"
	}
}

// eventBody composes the popup body lines with real names.
func (s *StatePlaying) eventBody(ev *components.GameEvent) []string {
	world := s.Status.TM.World
	actor := npcNameByID(world, ev.ActorID)
	city := CityName(world, ev.CityID)

	switch ev.Kind {
	case components.EventTaxDemand:
		return []string{
			fmt.Sprintf("%s, ruler of %s, demands a special", actor, city),
			fmt.Sprintf("levy of %d gold from you. Today.", ev.Amount),
		}
	case components.EventBanditShakedown:
		return []string{
			fmt.Sprintf("%s blocks your path, blade drawn:", actor),
			fmt.Sprintf("\"Your gold or your blood — %d coins.\"", ev.Amount),
		}
	case components.EventJobOffer:
		lines := []string{
			fmt.Sprintf("%s needs a good %s.", city, JobName(uint8(ev.Amount))),
		}
		if ev.ActorID != 0 {
			lines = append(lines, fmt.Sprintf("%s vouches for the position.", actor))
		}
		lines = append(lines, "The work is yours if you want it.")
		return lines
	case components.EventInsult:
		return []string{
			fmt.Sprintf("%s mocks you before the crowd.", actor),
			"Their words sting more than they should.",
		}
	case components.EventFestival:
		return []string{
			fmt.Sprintf("The granaries of %s overflow!", city),
			"Music and firelight fill the streets.",
		}
	case components.EventWarNews:
		return []string{
			"Heralds cry the news in every square:",
			fmt.Sprintf("your realm is at war with Realm %d!", ev.Amount),
		}
	case components.EventPeaceNews:
		return []string{
			"The banners come down at last:",
			fmt.Sprintf("peace with Realm %d is sworn.", ev.Amount),
		}
	case components.EventRulerDied:
		who := actor
		if ev.ActorID == 0 {
			who = "The old ruler"
		}
		lines := []string{fmt.Sprintf("%s, ruler of %s, is dead.", who, city)}
		if ev.Flag == components.EventFlagThroneClaimable {
			lines = append(lines, "The throne sits empty. It could be yours.")
		}
		return lines
	default:
		return []string{"The world stirs."}
	}
}

// eventChoices maps an event to its choice labels. Indexes MUST match the
// choice contract documented on systems.ResolveEvent.
func eventChoices(ev *components.GameEvent) []string {
	switch ev.Kind {
	case components.EventTaxDemand:
		return []string{fmt.Sprintf("Pay %d gold", ev.Amount), "Refuse"}
	case components.EventBanditShakedown:
		return []string{fmt.Sprintf("Pay %d gold", ev.Amount), "Fight!", "Flee"}
	case components.EventJobOffer:
		return []string{"Accept the job", "Decline"}
	case components.EventInsult:
		return []string{"Laugh it off", "Demand apology", "Strike them!"}
	case components.EventFestival:
		return []string{"Join the festival", "Keep working"}
	default: // War/peace news, ruler died
		return []string{"Acknowledge"}
	}
}

// npcNameByID resolves a living NPC's display name (fallback for the departed).
func npcNameByID(world *ecs.World, id uint64) string {
	if id == 0 {
		return "someone"
	}
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)

	name := fmt.Sprintf("Citizen %d", id)
	f := ecs.All(npcID, identID)
	q := world.Query(&f)
	for q.Next() {
		ident := (*components.Identity)(q.Get(identID))
		if ident.ID == id && ident.Name != "" {
			name = ident.Name
		}
	}
	return name
}
