import sys

def fix_main():
    with open('cmd/game/main.go', 'r') as f:
        content = f.read()

    search = """	// Phase 18: Justice Engine
	tickManager.AddSystem(systems.NewJusticeSystem(world), engine.PhaseResolution)"""

    replace = """	// Phase 18: Justice Engine
	tickManager.AddSystem(systems.NewJusticeSystem(world), engine.PhaseResolution)

	// Phase 24.1: The Labor Union Engine
	tickManager.AddSystem(systems.NewLaborUnionSystem(world, hookGraph), engine.PhaseResolution)"""

    if search in content:
        content = content.replace(search, replace)
    else:
        print("Search string not found in main.go")

    with open('cmd/game/main.go', 'w') as f:
        f.write(content)

if __name__ == "__main__":
    fix_main()
    print("Fixed main.go")
