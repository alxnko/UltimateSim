package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"log"
)

func main() {
	if ebiten.Termination == nil {
		log.Fatal("ebiten.Termination is nil?")
	}
}
