package systems

import (
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

// Phase 07.3: Linguistic Drift
// LanguageDriftSystem propagates language mutations and dialects over time.

type LanguageDriftSystem struct {
	tickCounter        uint64
	pidginTracker      map[uint32]uint32
	establishedPidgins map[uint32]uint16
	lastMutationMap    map[uint64]uint64
	nextLanguageID     uint16

	cultureID ecs.ID
	memoryID  ecs.ID
	identID   ecs.ID
}

// NewLanguageDriftSystem initializes a new LanguageDriftSystem.
func NewLanguageDriftSystem() *LanguageDriftSystem {
	return &LanguageDriftSystem{
		pidginTracker:      make(map[uint32]uint32),
		establishedPidgins: make(map[uint32]uint16),
		lastMutationMap:    make(map[uint64]uint64),
		nextLanguageID:     1000, // Start generating new IDs from 1000
	}
}

// Update runs the system logic.
func (s *LanguageDriftSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Performance optimization: Run only every 100 ticks
	if s.tickCounter%100 != 0 {
		return
	}

	s.cultureID = ecs.ComponentID[components.CultureComponent](world)
	s.memoryID = ecs.ComponentID[components.Memory](world)
	s.identID = ecs.ComponentID[components.Identity](world)

	filter := ecs.All(s.cultureID, s.memoryID, s.identID)
	query := world.Query(filter)

	// Since we need to look up target entities during iteration,
	// and Query prevents concurrent get, we first extract valid nodes.
	type NodeData struct {
		Entity  ecs.Entity
		Culture *components.CultureComponent
		Memory  *components.Memory
		Ident   *components.Identity
	}

	var nodes []NodeData

	for query.Next() {
		nodes = append(nodes, NodeData{
			Entity:  query.Entity(),
			Culture: (*components.CultureComponent)(query.Get(s.cultureID)),
			Memory:  (*components.Memory)(query.Get(s.memoryID)),
			Ident:   (*components.Identity)(query.Get(s.identID)),
		})
	}

	// We also need to map all entities with a CultureComponent to support target lookups,
	// even if they don't have Memory (like entityB in the test).
	filterAllCulture := ecs.All(s.cultureID, s.identID)
	queryAllCulture := world.Query(filterAllCulture)

	// Map of entity ID to LanguageID to prevent frequent world.Get lookups
	languageMap := make(map[uint64]uint16)
	for queryAllCulture.Next() {
		cult := (*components.CultureComponent)(queryAllCulture.Get(s.cultureID))
		id := (*components.Identity)(queryAllCulture.Get(s.identID))
		languageMap[id.ID] = cult.LanguageID
	}

	for _, node := range nodes {
		lastSameLanguageTick := uint64(0)
		hasInteractedSameLang := false

		// Check memory events
		for _, event := range node.Memory.Events {
			if event.InteractionType == components.InteractionGossip {
				targetLang, exists := languageMap[event.TargetID]
				if !exists {
					continue
				}

				if targetLang == node.Culture.LanguageID {
					if event.TickStamp > lastSameLanguageTick {
						lastSameLanguageTick = event.TickStamp
					}
					hasInteractedSameLang = true
				} else {
					// Different language. Track interaction for Pidgin Creation.
					// We only count it once if it happened in the last 100 ticks (since last execution)
					// to avoid over-counting the same event in the ring buffer.
					if event.TickStamp > s.tickCounter-100 {
						var minLang, maxLang uint16
						if node.Culture.LanguageID < targetLang {
							minLang = node.Culture.LanguageID
							maxLang = targetLang
						} else {
							minLang = targetLang
							maxLang = node.Culture.LanguageID
						}

						pairKey := (uint32(minLang) << 16) | uint32(maxLang)
						s.pidginTracker[pairKey]++

						// Assign established pidgin or check for creation
						if pidginID, exists := s.establishedPidgins[pairKey]; exists {
							node.Culture.LanguageID = pidginID
							s.lastMutationMap[node.Ident.ID] = s.tickCounter
						} else {
							// Pidgin Creation Check
							if s.pidginTracker[pairKey] >= 50000 {
								// Generate Pidgin
								s.pidginTracker[pairKey] = 0 // Reset
								pidginID := s.nextLanguageID
								s.nextLanguageID++
								s.establishedPidgins[pairKey] = pidginID

								// Assign to interacting node directly. It will spread organically later.
								node.Culture.LanguageID = pidginID
								s.lastMutationMap[node.Ident.ID] = s.tickCounter
							}
						}
					}
				}
			}
		}

		// Dialect Formation Check
		// Prevent infinite mutations by checking when the entity last mutated
		lastMutationTick := s.lastMutationMap[node.Ident.ID]
		if s.tickCounter > 10000 && (s.tickCounter-lastMutationTick > 10000) {
			if !hasInteractedSameLang || (s.tickCounter-lastSameLanguageTick > 10000) {
				// Dialect Formation
				node.Culture.LanguageID = s.nextLanguageID
				s.nextLanguageID++
				s.lastMutationMap[node.Ident.ID] = s.tickCounter
			}
		}
	}
}
