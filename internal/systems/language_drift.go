package systems

import (
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

// Phase 07.3: Linguistic Drift
// Propagates language mutations and drift natively via ticks and interactions

// LanguageDriftSystem handles dialect formation and pidgin creation
type LanguageDriftSystem struct {
	tickCounter uint64
	GlobalLanguageCounter uint16

	cultureID ecs.ID
	memoryID  ecs.ID
	ruinID    ecs.ID
}

// Update runs the system every tick, but processes logic based on tick thresholds.
func (s *LanguageDriftSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run check every 100 ticks to save CPU
	if s.tickCounter%100 != 0 {
		return
	}

	if s.GlobalLanguageCounter == 0 {
		s.GlobalLanguageCounter = 1000 // Base languages under 1000, new languages above
	}

	s.cultureID = ecs.ComponentID[components.CultureComponent](world)
	s.memoryID = ecs.ComponentID[components.Memory](world)
	s.ruinID = ecs.ComponentID[components.RuinComponent](world)

	filter := ecs.All(s.cultureID, s.memoryID).Without(s.ruinID)
	query := world.Query(&filter)

	for query.Next() {
		culture := (*components.CultureComponent)(query.Get(s.cultureID))
		memory := (*components.Memory)(query.Get(s.memoryID))

		var lastParentInteraction uint64 = culture.DialectTickStamp

		// Track interactions for this cycle
		var localForeignLanguageID uint16 = 0
		var maxForeignInteractions uint32 = 0

		foreignInteractionCounts := make(map[uint16]uint32)

		for _, event := range memory.Events {
			if event.TickStamp == 0 {
				continue
			}

			// We check for events that are recent.
			if event.LanguageID == culture.LanguageID {
				if event.TickStamp > lastParentInteraction {
					lastParentInteraction = event.TickStamp
				}
			} else if event.LanguageID != 0 {
				// Foreign Language interaction
				if event.TickStamp > s.tickCounter-100 && event.TickStamp <= s.tickCounter {
					foreignInteractionCounts[event.LanguageID] += 1
				}
			}
		}

		// Dialect Formation Check
		// If 10,000 ticks have passed without parent language interaction
		if s.tickCounter > lastParentInteraction+10000 {
			s.GlobalLanguageCounter++
			culture.LanguageID = s.GlobalLanguageCounter
			culture.DialectTickStamp = s.tickCounter
			// Reset foreign ticks
			culture.ForeignLanguageID = 0
			culture.ForeignInteractionTicks = 0
			continue // Entity dialect shifted, skip pidgin check this cycle
		}

		// Update DialectTickStamp to prevent premature drift if we just interacted
		culture.DialectTickStamp = lastParentInteraction

		// Pidgin Check preparation
		for langID, count := range foreignInteractionCounts {
			// use >= to ensure that if there's only 1 interaction, it gets picked up
			if count >= maxForeignInteractions {
				maxForeignInteractions = count
				localForeignLanguageID = langID
			}
		}

		if localForeignLanguageID != 0 {
			if culture.ForeignLanguageID == localForeignLanguageID {
				culture.ForeignInteractionTicks += 100
			} else {
				culture.ForeignLanguageID = localForeignLanguageID
				culture.ForeignInteractionTicks = 100
			}

			// Pidgin Creation Check
			// If 50,000 ticks of dominant foreign interaction
			if culture.ForeignInteractionTicks >= 50000 {
				// Mathematically assign new shared PidginLanguageID
				// Using deterministic XOR assignment based on both languages
				var minLang, maxLang uint16
				if culture.LanguageID < culture.ForeignLanguageID {
					minLang = culture.LanguageID
					maxLang = culture.ForeignLanguageID
				} else {
					minLang = culture.ForeignLanguageID
					maxLang = culture.LanguageID
				}

				pidginID := uint16(50000) + minLang + (maxLang * 3)

				culture.LanguageID = pidginID
				culture.DialectTickStamp = s.tickCounter
				culture.ForeignLanguageID = 0
				culture.ForeignInteractionTicks = 0
			}
		} else {
			// Decay foreign interaction ticks if no foreign interactions are present
			if culture.ForeignInteractionTicks > 0 {
				if culture.ForeignInteractionTicks >= 100 {
					culture.ForeignInteractionTicks -= 100
				} else {
					culture.ForeignInteractionTicks = 0
					culture.ForeignLanguageID = 0
				}
			}
		}
	}
}
