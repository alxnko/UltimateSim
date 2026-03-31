1. **Goal:** Execute the "Systemic Emergence" objective by implementing a missing mechanic from the Vision ("Artifacts (e.g., 'Sword of Bektur') that carry historical memory and grant 'Auras of Legitimacy.'").
2. **DOD Strategy:** Modify `LegitimacySystem` to check if the ruler possesses an equipped `EquipmentComponent` with a `LegendComponent` weapon. If they do, extract its `Prestige` and dynamically add it to their `Legitimacy.Score`, granting the "Aura of Legitimacy".
3. **The "Butterfly Effect" Test:** Create `TestLegitimacySystem_ArtifactAura` to verify that giving a ruling capital an equipped artifact mathematically boosts its `LegitimacyComponent.Score`, preventing the `MilitaryRevoltSystem` (Phase 27.1) from triggering due to low legitimacy.
4. **Implementation:**
   - Update `internal/systems/legitimacy.go` to inject `ecs.ComponentID[components.EquipmentComponent](world)` into the `NewLegitimacySystem` and `Update` loops.
   - Inject logic into `LegitimacySystem`'s loop to check `world.Has(entity, equipID)`, pull `EquipmentComponent`, and if `Equipped` is true, add `Weapon.Prestige / 10` (or similar scaling) to `newScore`.
   - Update `internal/systems/legitimacy_test.go` to add the new integration test.
   - Run verification tests.
   - Update `docs/internal/EVOLUTION_LOG.md`.
5. **Pre-commit Checks:** Call `pre_commit_instructions`.
6. **Submit:** Commit the changes on branch `evolution/artifact-legitimacy-aura` with appropriate message.
