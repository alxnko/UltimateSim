# Phase 16: Geopolitical Sovereignty & Unions

_Objective: Move beyond independent settlements to simulate high-level political structures—Countries and Unions—driven by security, profit, and economic efficiency._

## 16.1 The Country Entity (Macro-State)

- **`CountryComponent`**: A higher-level tag attached to a "Capital" entity that manages sub-affiliations.
- **State Currency**: A Country enforces a `StandardCurrencyID` across all its `CityID` members. This eliminates exchange fees between internal cities, boosting trade efficiency.
- **Taxation Loops**: Sub-cities transfer a portion of their `MarketComponent` revenue to the Country's `TreasuryComponent`.

## 16.2 Strategic Unions & Pacts

- **`UnionEntity`**: A specialized non-physical entity representing a treaty or agreement between independent Countries or Cities.
- **Union Types**:
    - **Defense Pact (War Union)**: If one member's `MilitaryForce` is attacked, all members trigger an aggressive pathing state towards the aggressor.
    - **Currency Union (The Common Market)**: Multiple independent states agree to use a shared `CurrencyID`. `PriceNormalizationSystem` ensures goods flow freely without tariff penalties.
    - **Economic Bloc**: Shared access to `StorageComponent` stockpiles during famines.

## 16.3 Profit-Driven Unification

- **Sovereignty Calculus**: Every X ticks, a City's `AdministrationSystem` performs a cost-benefit analysis:
    - **Profit Gain**: Joining a Union/Country reduces trade friction and exchange rate risks.
    - **Loss**: Joining requires paying taxes and following regional `IllegalActionIDs` (Phase 17).
- **Emergent Mergers**: If a city identifies that a shared currency with a neighbor will increase its `MarketComponent` volume by >15%, it initiates a "Diplomatic Hook" to propose a Union.

## 16.4 Administrative Reach & Friction

- **Information Latency**: Just like administrative decay, Union/Country decisions are limited by the physical distance a messenger NPC must travel.
- **Fracture Logic**: If a city is too far from the Capital to benefit from the Defense Pact, it may unilaterally withdraw from the Union to save on tax costs.
