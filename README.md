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

## Controls
- **Global Strategy / Action-RPG Mode**:
    - `WASD`: Move possessed entity
    - `Mouse Wheel`: Zoom camera
    - `Left Click`: Attack / Primary Action
    - `Right-Click`: Interact
    - `SPACE`: Pause/Resume Simulation
    - `ESC`: Exit Game

---

**Crucial Note to Agents:** You must maintain `docs/implemented_functionality.md` at all times. If you implement a new feature, update the document immediately.
