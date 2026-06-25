package ui_test

import (
	"github.com/ALXNKO/UltimateSim/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"testing"
)

type mockState struct {
	updateCalled bool
	drawCalled   bool
}

func (m *mockState) Update(sm *ui.StateManager) error {
	m.updateCalled = true
	return nil
}

func (m *mockState) Draw(screen *ebiten.Image) {
	m.drawCalled = true
}

func TestStateManager(t *testing.T) {
	sm := ui.NewStateManager()
	ms := &mockState{}
	sm.Push(ms)

	sm.Update()
	if !ms.updateCalled {
		t.Error("expected state update to be called")
	}

	sm.Draw(ebiten.NewImage(10, 10))
	if !ms.drawCalled {
		t.Error("expected state draw to be called")
	}
}
