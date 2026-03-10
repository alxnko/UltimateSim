# Phase 10: State Failure & Frictional Limits

_Objective: Implement the mechanics that force stable empires to splinter and collapse, mimicking the massive frictional realities of medieval geography and biology._

## 10.1 Debt Default Execution (The Hook Trap)

- **`LoanContractComponent`:** Attach to debtor nodes. Contains `{CreditorID uint64, AssetID uint32, DueTick uint64}`.
- **Execution Logic:** If `current_tick >= DueTick` and internal `StorageComponent` metrics fail repayment logic -> Transfer mapped `AssetID` (like `CityID` bounds mapping to Guild arrays).
- Debt explicitly functions as the most destructive engine of war in the math limits.

## 10.2 Bureaucratic Delay (Administrative Entropy)

- **The Problem:** Empires normally grow indefinitely. We require strict communication friction.
- **`OrderEntity` spawning:** A specific action declared at a `CityID` designated as "Capital" does NOT instantly evaluate globally.
- It physically spawns an `OrderEntity` traversing geographical distance via the HPA\* grid routing matrix.
- **`AdministrativeDecaySystem`:** Tests transit time of the `OrderEntity` vs the targeted `CityID.Loyalty` base integer.
- If Decay > Loyalty, the Order terminates prematurely (failed request, simulating rebellious vassals intercepting couriers).

## 10.3 Biological Entropy (Plagues & Immune Arrays)

- **`DiseaseEntity` generation:** Run randomly based on the Global Seed parameter focused on high-traffic trade hub arrays.
- **`DiseaseVectorSystem`:** Maps lethality logic against exposed entities calculating `GeneticsComponent.Health` arrays.
- **Survivors:** Attach an `ImmunityTag` component map to surviving ECS structures. Subsequent `DiseaseEntity` evaluations will mathematically ignore these identities, generating localized biological walls against specific plagues.
