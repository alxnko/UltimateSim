package ui

import (
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
)

// Shell Phase: thin UI→engine save/load bridge. Persists to a fixed slot.

const savePath = "savegame.db"

// engineSave writes the current world to the save slot.
func engineSave(status *render.LoadingStatus) error {
	db, err := engine.InitDB(savePath)
	if err != nil {
		return err
	}
	defer db.Close()
	return engine.SaveWorld(status.TM, status.Grid, status.Seed, db)
}

// engineLoad restores the world from the save slot into the live TickManager.
func engineLoad(status *render.LoadingStatus) error {
	db, err := engine.InitDB(savePath)
	if err != nil {
		return err
	}
	defer db.Close()
	return engine.LoadWorld(status.TM, db)
}
