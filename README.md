# UltimateSim (Boundless Sovereigns)

A systemic Action-RPG and Total Simulation grand strategy game (like Streets of Rogue meets Kenshi) built on Go and an ECS (`arche-go`) engine.

## Documentation

The codebase uses a strictly maintained set of documentation to coordinate local AI Agents and ensure architectural compliance.

- [Vision](docs/vision.md)
- [Mechanics](docs/mechanics.md)
- [Architecture](docs/architecture.md)
- [Implemented Functionality](docs/implemented_functionality.md) - **The comprehensive index of all currently implemented packages, ECS Components, and ECS Systems.**
- [Roadmap](docs/roadmap.md)

## How to Build and Run

### Prerequisites
- [Go](https://go.dev/dl/) (version 1.21+)

### Build
To build the game as a single executable without external dependencies (pure Go):
```ps1
$env:CGO_ENABLED=0; go build -o game.exe ./cmd/game
```

### Run
```ps1
./game.exe
```

## How to Play

You possess a single character in a fully simulated medieval world. Survive, build,
climb the social hierarchy, rule a city, wage war, and — when you die — continue as
your heir. The world keeps simulating around you whether or not you act.

### Controls
| Input | Action |
|-------|--------|
| `WASD` | Move your character |
| `Left Click` | Attack an NPC / select an entity (Shift+Click always selects) |
| `Right Click` | Open a context menu (Talk, Trade, Attack, Order, Pick up, Build, Inspect) |
| `Mouse Wheel` | Zoom (below 0.5x switches to the strategic lens map) |
| `E` | Hammer the nearest construction site |
| `B` | Toggle build mode (`R` cycles structure, click to place) |
| `G` | Ambitions panel (optional goals + rewards) |
| `C` | Character sheet |
| `L` | Chronicle (event log) |
| `Tab` / `F1`–`F4` | Cycle / pick strategic lens (Political, Wealth, Crime, Culture) |
| `Space` | Pause / resume the simulation |
| `Esc` | Pause menu (Save, Load, Quit) / close panels |

### Core loops
- **Survive**: keep Food, Blood and Stamina up; combat depletes vitals, pain slows you.
- **Society**: Talk to build hooks (favors), Gift, Threaten, or Share rumors. Relationships
  feed gossip, legitimacy and feuds.
- **Power**: earn enough positive standing in your city to **Claim Leadership**. As a Ruler
  you issue Orders (move/work/attack/follow) to subordinates and set Laws; as a Sovereign
  (capital ruler) you control currency debasement and declare war.
- **Build**: buy materials at a market, enter build mode, place Houses, Workshops,
  Storehouses, Shrines, Farms or Taverns. NPC builders (or you, with `E`) complete them.
- **Legacy**: when you die, pick a family heir to continue your dynasty — inheriting prestige,
  debt, artifacts and feuds. No heir means a dynasty-end / reincarnate screen.

---

**Crucial Note to Agents:** You must maintain `docs/implemented_functionality.md` at all times. If you implement a new feature, update the document immediately.
