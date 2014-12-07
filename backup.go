package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Restores all images that were saved as backup. This avoid double applying
// overlays when running the program multiple times.
func RestoreBackup(user User) {
	baseDir := filepath.Join(user.Dir, "config", "grid")
	entries, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return
	}

	for _, file := range entries {
		if strings.Contains(file.Name(), " (original)") {
			backupPath := filepath.Join(baseDir, file.Name())
			mainPath := strings.Replace(backupPath, " (original)", "", 1)
			bytes, _ := ioutil.ReadFile(backupPath)
			_ = ioutil.WriteFile(mainPath, bytes, 0666)
		}
	}
}

// If a game has a custom image, backs it up by appending "(original)" to the
// file name.
func BackupGame(game *Game) error {
	if game.ImagePath != "" && game.ImageBytes != nil {
		backupPath := strings.Replace(game.ImagePath, ".", " (original).", 1)
		return ioutil.WriteFile(backupPath, game.ImageBytes, 0666)
	} else {
		return nil
	}
}
