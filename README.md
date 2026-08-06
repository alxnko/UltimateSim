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

A grand-strategy life-sim: EU-style realms, CK-style dynasties and intrigue,
Vic-style economy — but you live inside it as ONE character, and you can play
as ANY character. The world generates as a real planet (continents, oceans,
seas, islands), starts already political (cities, countries, capitals, rulers,
councils, married dynasties), and keeps simulating around you whether or not
you act. Interactive events (tax demands, bandit shakedowns, insults, job
offers, festivals, wars, successions) arrive as choice popups, CK-style.

### Start
Press `Enter` at the menu; the world warms up, then the **character select**
lists real inhabitants — pick anyone (arrows + `Enter`, `R` for random). You
can switch bodies at any time later: right-click anyone → **Play As**.

### Controls
| Input | Action |
|-------|--------|
| `WASD` | Move your character (camera re-follows) |
| `Arrows` / `MMB drag` / minimap click | Free camera pan (`F` returns to your character) |
| `Left Click` | Attack an NPC / select an entity (Shift+Click always selects) |
| `Right Click` | Context menu (Talk, Trade, Attack, Play As, Order, Build…) — cancels build/order mode |
| `Mouse Wheel` | Zoom (below 0.5x switches to the strategic lens map) |
| `1` `2` `3` `4` | Sim speed 1x / 2x / 4x / 8x (also HUD buttons) |
| `Space` | Pause / resume |
| `P` / `M` | Diplomacy panel / Market & trade panel (also top tab strip) |
| `B` | Build mode: ghost preview shows validity + cost (`R` cycles structure) |
| `E` | Hammer the nearest construction site (builders also auto-work) |
| `G` / `C` / `L` | Goals / Character sheet / Chronicle |
| `Tab` / `F1`–`F4` | Strategic lenses (Political, Wealth, Crime, Culture) |
| `Esc` | Close panels / pause menu (Save, Load, Quit) |

### Core loops
- **Live**: keep Food, Blood, Stamina up. Citizens visibly work their jobs —
  farmers at farms, lumberjacks in forests, guards on patrol — and so can you.
- **Events**: story beats arrive as popups with choices; the sim pauses while
  you decide. The chronicle records your saga.
- **Society & dynasty**: talk, gift, threaten, marry; family and hooks feed
  legitimacy, plots and succession.
- **Power**: win standing → **Claim Leadership** of your city → set laws and
  taxes, appoint a council (Steward/Marshal/Diplomat/Spymaster), issue orders.
  Rule the capital and you are Sovereign: diplomacy, alliances, war.
- **Realm**: the diplomacy panel runs opinions, alliances, truces, war score;
  the market panel runs prices, tax rates and trade routes between cities.
- **Intrigue**: plot to seize a throne or assassinate a rival — spymasters
  hunt plotters.
- **Legacy**: die and continue as your heir, inheriting prestige, debt and
  feuds. The hint strip above always shows your next step on the ladder.

---

**Crucial Note to Agents:** You must maintain `docs/implemented_functionality.md` at all times. If you implement a new feature, update the document immediately.
