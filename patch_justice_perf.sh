sed -i '/pathID := ecs.ComponentID\[components.Path\](world)/d' internal/systems/justice.go
sed -i '/idID := ecs.ComponentID\[components.Identity\](world)/a \	pathID := ecs.ComponentID[components.Path](world)' internal/systems/justice.go
