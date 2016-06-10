package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// BackupGame if a game has a custom image, backs it up by appending "(original)" to the
// file name.
func BackupGame(game *Game) error {
	if game.ImagePath != "" && game.ImageBytes != nil {
		ext := filepath.Ext(game.ImagePath)
		base := filepath.Base(game.ImagePath)
		backupPath := filepath.Join(filepath.Dir(game.ImagePath), strings.TrimSuffix(base, ext)+" (original)"+ext)
		return ioutil.WriteFile(backupPath, game.ImageBytes, 0666)
        }
	return nil
}
