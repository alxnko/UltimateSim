package systems

import (
	"fmt"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 44 - The Vassal Safety Valve Engine
// When a Clan accumulates massive wealth (Monopoly), rival clan members with TraitJealous
// automatically generate deep negative hooks against the wealthiest member and spread rumors.
// Bridging Economy (Wealth), Genetics/Traits (Jealousy), Information (Rumors), and Justice (Blood Feuds).

type npcValveData struct {
	Entity     ecs.Entity
	IdentityID uint64
	CityID     uint32
	ClanID     uint32
	Wealth     float32
	BaseTraits uint32
	SecretComp *components.SecretComponent
}

type clanWealthData struct {
	TotalWealth    float32
	WealthiestID   uint64
	MaxMemberWealth float32
}

type VassalSafetyValveSystem struct {
	tickStamp uint64
	hooks     *engine.SparseHookGraph

	// Component IDs
	npcID     ecs.ID
	identID   ecs.ID
	affilID   ecs.ID
	needsID   ecs.ID
	secretID  ecs.ID
}

func NewVassalSafetyValveSystem(world *ecs.World, hooks *engine.SparseHookGraph) *VassalSafetyValveSystem {
	return &VassalSafetyValveSystem{
		hooks:    hooks,
		npcID:    ecs.ComponentID[components.NPC](world),
		identID:  ecs.ComponentID[components.Identity](world),
		affilID:  ecs.ComponentID[components.Affiliation](world),
		needsID:  ecs.ComponentID[components.Needs](world),
		secretID: ecs.ComponentID[components.SecretComponent](world),
	}
}

func (s *VassalSafetyValveSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Process every 500 ticks to preserve performance
	if s.tickStamp%500 != 0 {
		return
	}

	if s.hooks == nil {
		return
	}

	// 1. Extract all active NPCs into a flat DOD slice
	query := world.Query(filter.All(s.npcID, s.identID, s.affilID, s.needsID))
	var npcs []npcValveData

	for query.Next() {
		ident := (*components.Identity)(query.Get(s.identID))
		affil := (*components.Affiliation)(query.Get(s.affilID))
		needs := (*components.Needs)(query.Get(s.needsID))

		var secret *components.SecretComponent
		if world.Has(query.Entity(), s.secretID) {
			secret = (*components.SecretComponent)(world.Get(query.Entity(), s.secretID))
		}

		// Only track citizens of a city belonging to a Clan
		if affil.CityID != 0 && affil.ClanID != 0 {
			npcs = append(npcs, npcValveData{
				Entity:     query.Entity(),
				IdentityID: ident.ID,
				CityID:     affil.CityID,
				ClanID:     affil.ClanID,
				Wealth:     needs.Wealth,
				BaseTraits: ident.BaseTraits,
				SecretComp: secret,
			})
		}
	}

	if len(npcs) == 0 {
		return
	}

	// 2. Map wealth per City and per Clan
	// cityID -> total wealth of city
	cityWealth := make(map[uint32]float32)
	// cityID -> clanID -> clanWealthData
	cityClanWealth := make(map[uint32]map[uint32]*clanWealthData)

	for _, npc := range npcs {
		cityWealth[npc.CityID] += npc.Wealth

		if _, exists := cityClanWealth[npc.CityID]; !exists {
			cityClanWealth[npc.CityID] = make(map[uint32]*clanWealthData)
		}

		if _, exists := cityClanWealth[npc.CityID][npc.ClanID]; !exists {
			cityClanWealth[npc.CityID][npc.ClanID] = &clanWealthData{}
		}

		cData := cityClanWealth[npc.CityID][npc.ClanID]
		cData.TotalWealth += npc.Wealth

		if npc.Wealth > cData.MaxMemberWealth {
			cData.MaxMemberWealth = npc.Wealth
			cData.WealthiestID = npc.IdentityID
		}
	}

	// 3. Identify Monopoly Clans per City
	// A monopoly clan holds > 50% of the city's wealth and > 1000 total wealth
	type monopolyData struct {
		ClanID       uint32
		WealthiestID uint64
	}
	cityMonopolies := make(map[uint32]monopolyData)

	for cityID, clans := range cityClanWealth {
		totalCityW := cityWealth[cityID]
		if totalCityW < 1000 {
			continue // City is too poor to care
		}

		for clanID, cData := range clans {
			if cData.TotalWealth > 1000 && cData.TotalWealth > (totalCityW * 0.5) {
				cityMonopolies[cityID] = monopolyData{
					ClanID:       clanID,
					WealthiestID: cData.WealthiestID,
				}
				break // Only one clan can hold > 50%
			}
		}
	}

	// 4. Trigger Jealousy Responses
	if len(cityMonopolies) == 0 {
		return
	}

	registry := engine.GetSecretRegistry()

	for _, npc := range npcs {
		monopoly, hasMonopoly := cityMonopolies[npc.CityID]
		if !hasMonopoly {
			continue
		}

		// The NPC is jealous and NOT in the Monopoly Clan
		if npc.ClanID != monopoly.ClanID && (npc.BaseTraits&components.TraitJealous != 0) {

			// Verify hook does not already exist
			existingHook := s.hooks.GetHook(npc.IdentityID, monopoly.WealthiestID)
			if existingHook > -50 {
				// Generate massive negative grudge to trigger Blood Feud (Phase 23)
				s.hooks.AddHook(npc.IdentityID, monopoly.WealthiestID, -50)

				// Generate mutated Secret to ruin reputation (Only generate once per target)
				if npc.SecretComp != nil && registry != nil {
					rumorText := fmt.Sprintf("monopoly_resentment_against_%d_tick_%d", monopoly.WealthiestID, s.tickStamp)
					secretID := registry.RegisterSecret(rumorText)

					npc.SecretComp.Secrets = append(npc.SecretComp.Secrets, components.Secret{
						OriginID: npc.IdentityID,
						SecretID: secretID,
						Virality: 255, // Highly contagious
					})
				}
			}
		}
	}
}
