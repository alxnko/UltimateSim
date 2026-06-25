package ui

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: right-click context menus and the order-targeting flow.

// menuContext records what the open context menu refers to.
type menuContext struct {
	entity ecs.Entity
	kind   uint8
	wx, wy float32
}

// openContextMenu builds a target-sensitive menu at the cursor.
func (s *StatePlaying) openContextMenu(mx, my int, wx, wy float32) {
	pc := s.PC
	world := s.Status.TM.World
	ent, kind, found := pc.EntityAt(wx, wy, 1.2)
	s.menuCtx = menuContext{entity: ent, kind: kind, wx: wx, wy: wy}

	var items []string
	if !found {
		items = []string{"Walk here", "Build here"}
		s.menuCtx.kind = TargetNone
	} else {
		switch kind {
		case TargetNPC:
			items = []string{"Inspect", "Talk", "Attack"}
			if player, ok := pc.PossessedEntity(); ok {
				rank, cityID, _ := systems.GetRank(world, player)
				affID := ecs.ComponentID[components.Affiliation](world)
				if rank >= systems.RankRuler && world.Has(ent, affID) &&
					(*components.Affiliation)(world.Get(ent, affID)).CityID == cityID {
					items = append(items, "Give Order")
				}
			}
		case TargetVillage:
			items = []string{"Inspect", "Trade"}
			if player, ok := pc.PossessedEntity(); ok {
				rank, cityID, _ := systems.GetRank(world, player)
				affID := ecs.ComponentID[components.Affiliation](world)
				if rank >= systems.RankRuler && world.Has(ent, affID) &&
					(*components.Affiliation)(world.Get(ent, affID)).CityID == cityID {
					items = append(items, "Edit Laws")
				}
			}
		case TargetItem, TargetCoin:
			items = []string{"Inspect", "Pick up"}
		case TargetSite:
			items = []string{"Inspect", "Work here"}
		default:
			items = []string{"Inspect"}
		}
	}
	pc.Menu.Open(mx, my, items)
}

// runContextAction executes the chosen top-level menu item.
func (s *StatePlaying) runContextAction(idx int) {
	pc := s.PC
	world := s.Status.TM.World
	ctx := s.menuCtx
	label := ""
	if idx < len(pc.Menu.Items) {
		label = pc.Menu.Items[idx]
	}
	// pc.Menu.Items was cleared by Update? No — Update keeps Items. Use stored.
	switch label {
	case "Inspect":
		pc.Select(ctx.entity, ctx.kind)
	case "Talk":
		pc.DialogOpen = true
		pc.DialogTarget = ctx.entity
		pc.DialogReply = ""
	case "Attack":
		if pc.Bridge != nil {
			pc.Bridge.AttackX, pc.Bridge.AttackY, pc.Bridge.AttackValid = ctx.wx, ctx.wy, true
		}
	case "Trade":
		pc.TradeOpen = true
		pc.TradeVillage = ctx.entity
	case "Edit Laws":
		pc.LawsOpen = true
		pc.LawsVillage = ctx.entity
	case "Give Order":
		s.openOrderMenu(ctx.entity)
	case "Pick up":
		s.doPickup(ctx.entity)
	case "Work here":
		if player, ok := pc.PossessedEntity(); ok {
			if err := systems.HammerSite(world, player, ctx.entity); err != nil {
				pc.PushNote(err.Error(), s.Status.TM.Ticks)
			}
		}
	case "Walk here":
		s.walkPlayerToward(ctx.wx, ctx.wy)
	case "Build here":
		pc.Mode = ModeBuild
		if pc.BuildType == 0 {
			pc.BuildType = components.StructureHouse
		}
	}
}

// doPickup picks up an item/coin and removes it from the world.
func (s *StatePlaying) doPickup(item ecs.Entity) {
	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		return
	}
	systems.EnsurePlayerStorage(world, player)
	desc, consumed, err := systems.PickUp(world, player, item)
	if err != nil {
		s.PC.PushNote("Nothing to take.", s.Status.TM.Ticks)
		return
	}
	s.PC.PushNote(desc, s.Status.TM.Ticks)
	if consumed && world.Alive(item) {
		world.RemoveEntity(item)
	}
}

// walkPlayerToward nudges the player's velocity toward a world point once.
func (s *StatePlaying) walkPlayerToward(wx, wy float32) {
	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		return
	}
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	if !world.Has(player, posID) || !world.Has(player, velID) {
		return
	}
	pos := (*components.Position)(world.Get(player, posID))
	vel := (*components.Velocity)(world.Get(player, velID))
	dx := wx - pos.X
	dy := wy - pos.Y
	dist := dx*dx + dy*dy
	if dist > 0 {
		inv := float32(2.0) / sqrt32(dist)
		vel.X, vel.Y = dx*inv, dy*inv
	}
}

// --- Order targeting flow ---

// openOrderMenu opens the verb submenu for ordering a subordinate.
func (s *StatePlaying) openOrderMenu(target ecs.Entity) {
	pc := s.PC
	pc.OrderTarget = target
	mx, my := orderMenuPos()
	pc.SubMenuKind = 1
	pc.SubMenu.Open(mx, my, []string{"Move to...", "Follow me", "Attack...", "Work as Builder", "Work as Farmer", "Work as Guard"})
}

func orderMenuPos() (int, int) { return 1280/2 - 60, 720/2 - 60 }

// runSubMenuAction handles order-verb selection.
func (s *StatePlaying) runSubMenuAction(idx int) {
	pc := s.PC
	if pc.SubMenuKind != 1 {
		return
	}
	world := s.Status.TM.World
	target := pc.OrderTarget
	if !world.Alive(target) {
		return
	}
	label := ""
	if idx < len(pc.SubMenu.Items) {
		label = pc.SubMenu.Items[idx]
	}
	switch label {
	case "Move to...":
		pc.OrderType = components.OrderMove
		pc.Mode = ModeOrder
		pc.PushNote("Click where they should go.", s.Status.TM.Ticks)
	case "Follow me":
		s.issueOrder(target, components.OrderFollow, 0, 0, 0, 0)
	case "Attack...":
		pc.OrderType = components.OrderAttack
		pc.Mode = ModeOrder
		pc.PushNote("Click the target to attack.", s.Status.TM.Ticks)
	case "Work as Builder":
		s.issueOrder(target, components.OrderAssignJob, 0, 0, components.JobBuilder, 0)
	case "Work as Farmer":
		s.issueOrder(target, components.OrderAssignJob, 0, 0, components.JobFarmer, 0)
	case "Work as Guard":
		s.issueOrder(target, components.OrderAssignJob, 0, 0, components.JobGuard, 0)
	}
}

// placeOrderTarget completes a Move/Attack order at the clicked point.
func (s *StatePlaying) placeOrderTarget(wx, wy float32) {
	pc := s.PC
	world := s.Status.TM.World
	target := pc.OrderTarget
	pc.Mode = ModeNormal
	if !world.Alive(target) {
		return
	}
	switch pc.OrderType {
	case components.OrderMove:
		s.issueOrder(target, components.OrderMove, wx, wy, 0, 0)
	case components.OrderAttack:
		if ent, kind, found := pc.EntityAt(wx, wy, 1.2); found && kind == TargetNPC {
			identID := ecs.ComponentID[components.Identity](world)
			if world.Has(ent, identID) {
				tid := (*components.Identity)(world.Get(ent, identID)).ID
				s.issueOrder(target, components.OrderAttack, 0, 0, 0, tid)
			}
		}
	}
}

// issueOrder attaches (or replaces) a PlayerOrderComponent on a subordinate.
func (s *StatePlaying) issueOrder(target ecs.Entity, otype uint8, x, y float32, jobID uint8, targetID uint64) {
	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		return
	}
	identID := ecs.ComponentID[components.Identity](world)
	var issuerID uint64
	if world.Has(player, identID) {
		issuerID = (*components.Identity)(world.Get(player, identID)).ID
	}
	orderID := ecs.ComponentID[components.PlayerOrderComponent](world)
	if !world.Has(target, orderID) {
		world.Add(target, orderID)
	}
	o := (*components.PlayerOrderComponent)(world.Get(target, orderID))
	o.OrderType = otype
	o.X, o.Y = x, y
	o.JobID = jobID
	o.TargetID = targetID
	o.IssuerID = issuerID
	s.PC.PushNote("Order issued.", s.Status.TM.Ticks)
}
