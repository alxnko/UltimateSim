package ui

import (
	"image/color"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type StateMainMenu struct {
	OnStart func()
}

func NewStateMainMenu(onStart func()) *StateMainMenu {
	return &StateMainMenu{OnStart: onStart}
}

func (s *StateMainMenu) Update(sm *StateManager) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if s.OnStart != nil {
			s.OnStart()
		}
	}
	return nil
}

func (s *StateMainMenu) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 20, 255})
	ebitenutil.DebugPrintAt(screen, "BOUNDLESS SOVEREIGNS\n\nPress ENTER to Start", 300, 250)
}
