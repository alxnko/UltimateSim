package ui

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Shell Phase: decoders turning packed simulation IDs/bitmasks into readable
// labels for the inspector, HUD and menus.

// TraitNames decodes a BaseTraits bitmask into names.
func TraitNames(mask uint32) []string {
	var out []string
	bits := []struct {
		bit  uint32
		name string
	}{
		{components.TraitRiskTaker, "Risk-Taker"},
		{components.TraitCautious, "Cautious"},
		{components.TraitGossip, "Gossip"},
		{components.TraitJealous, "Jealous"},
		{components.TraitAbolitionist, "Abolitionist"},
		{components.TraitAmbitious, "Ambitious"},
		{components.TraitEsoteric, "Esoteric"},
	}
	for _, b := range bits {
		if mask&b.bit != 0 {
			out = append(out, b.name)
		}
	}
	return out
}

// InteractionName names an interaction type constant.
func InteractionName(t uint8) string {
	switch t {
	case components.InteractionGossip:
		return "Talked"
	case components.InteractionLanguage:
		return "Conversed"
	case components.InteractionAssault:
		return "Assaulted"
	case components.InteractionTheft:
		return "Stole from"
	case components.InteractionMurder:
		return "Murdered"
	case components.InteractionEsoteric:
		return "Cursed"
	case components.InteractionParasite:
		return "Infested"
	}
	return "Met"
}

// JobName names a job ID.
func JobName(id uint8) string {
	switch id {
	case components.JobFarmer:
		return "Farmer"
	case components.JobLumberjack:
		return "Lumberjack"
	case components.JobArtisan:
		return "Artisan"
	case components.JobGuard:
		return "Guard"
	case components.JobPreacher:
		return "Preacher"
	case components.JobCaster:
		return "Caster"
	case components.JobBandit:
		return "Bandit"
	case components.JobSailor:
		return "Sailor"
	case components.JobCaptain:
		return "Captain"
	case components.JobPenalLabor:
		return "Penal Laborer"
	case components.JobMercenary:
		return "Mercenary"
	case components.JobBuilder:
		return "Builder"
	case components.JobGravedigger:
		return "Gravedigger"
	case components.JobDoctor:
		return "Doctor"
	case components.JobMiner:
		return "Miner"
	}
	return "Unemployed"
}

// StructureName names a structure type.
func StructureName(t uint32) string {
	switch uint8(t) {
	case components.StructureHouse:
		return "House"
	case components.StructureWorkshop:
		return "Workshop"
	case components.StructureStorehouse:
		return "Storehouse"
	case components.StructureShrine:
		return "Shrine"
	case components.StructureFarm:
		return "Farm"
	case components.StructureTavern:
		return "Tavern"
	}
	return "Structure"
}

// ItemName names a resource item ID.
func ItemName(id uint8) string {
	switch id {
	case components.ItemWood:
		return "Wood"
	case components.ItemStone:
		return "Stone"
	case components.ItemIron:
		return "Iron"
	case components.ItemFood:
		return "Food"
	}
	return "Goods"
}

// BiomeName names a biome ID.
func BiomeName(id uint8) string {
	switch id {
	case engine.BiomeOcean:
		return "Ocean"
	case engine.BiomeBeach:
		return "Beach"
	case engine.BiomeScorched:
		return "Scorched"
	case engine.BiomeBare:
		return "Bare"
	case engine.BiomeTundra:
		return "Tundra"
	case engine.BiomeSnow:
		return "Snow"
	case engine.BiomeTemperateDesert:
		return "Temperate Desert"
	case engine.BiomeShrubland:
		return "Shrubland"
	case engine.BiomeGrassland:
		return "Grassland"
	case engine.BiomeTemperateDeciduousForest:
		return "Deciduous Forest"
	case engine.BiomeTemperateRainForest:
		return "Temperate Rainforest"
	case engine.BiomeSubtropicalDesert:
		return "Subtropical Desert"
	case engine.BiomeTropicalSeasonalForest:
		return "Tropical Forest"
	case engine.BiomeTropicalRainForest:
		return "Tropical Rainforest"
	case engine.BiomeMountain:
		return "Mountain"
	}
	return "Unknown"
}
