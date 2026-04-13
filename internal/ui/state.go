package ui

import "github.com/hajimehoshi/ebiten/v2"

type GameState interface {
	Update(sm *StateManager) error
	Draw(screen *ebiten.Image)
}

type StateManager struct {
	states []GameState
}

func NewStateManager() *StateManager {
	return &StateManager{
		states: make([]GameState, 0),
	}
}

func (sm *StateManager) Push(state GameState) {
	sm.states = append(sm.states, state)
}

func (sm *StateManager) Pop() {
	if len(sm.states) > 0 {
		sm.states = sm.states[:len(sm.states)-1]
	}
}

func (sm *StateManager) Switch(state GameState) {
	if len(sm.states) > 0 {
		sm.states[len(sm.states)-1] = state
	} else {
		sm.Push(state)
	}
}

func (sm *StateManager) Update() error {
	if len(sm.states) > 0 {
		return sm.states[len(sm.states)-1].Update(sm)
	}
	return nil
}

func (sm *StateManager) Draw(screen *ebiten.Image) {
	if len(sm.states) > 0 {
		sm.states[len(sm.states)-1].Draw(screen)
	}
}
