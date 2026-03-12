# Phase 15: Economic Agency, Businesses, & Currencies

_Objective: Move beyond simple resource needs to a full-fledged agentic economy where NPCs can start businesses, hire employees, and trade using localized or global currencies._

## 15.1 Individual Economic Agency

- **`BusinessComponent`**: NPCs with high `Wealth` needs and specific `Traits` (Ambitious, RiskTaker) can instantiate a `BusinessEntity`.
- **Business Types**: Initial businesses include Farms, Mines, Workshops, and Trading Houses.
- **Ownership**: The founding NPC becomes the "Owner" and manages the `StorageComponent` and `TreasuryComponent` of the business.

## 15.2 Employment & Wages

- **`JobMarketSystem`**: Businesses post "Job Openings" as data tags in the localized `Village` hub.
- **Hiring**: NPCs lacking resources or seeking specific career paths can "Apply" and receive a `JobComponent` linked to that business.
- **Wages**: Every X ticks, the business transfers a portion of its `TreasuryComponent` (Wealth) to the employee's personal `Needs.Wealth` tracker. Failure to pay leads to strikes or NPCs leaving for better opportunities.

## 15.3 Currency & Debt

- **`CurrencyComponent`**: Different `CityID` or `ClanID` organizations can mint their own physical `CoinEntities`.
- **Exchange Rates**: NPCs evaluate different currencies based on the backing organization's `Prestige` and resource stocks.
- **Lending**: Wealthy NPCs or Trading Houses can issue loans to other NPCs, creating physical `LoanContractComponents` that can be traded as assets.

## 15.4 Physical Locations & Workplaces

- NPCs must physically travel to their `Workplace` coordinates (Workshops, Fields) once per day/cycle.
- **Productivity**: Business output is a direct function of the `Genetics.Strength/Intellect` of the physical NPCs present at the location.
