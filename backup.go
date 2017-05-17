package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// BackupGame if a game has a custom image, backs it up by appending "(original)" to the
// file name.
func BackupGame(gridDir string, game *Game) error {
	if game.CleanImageBytes != nil {
		return ioutil.WriteFile(getBackupPath(gridDir, game), game.CleanImageBytes, 0666)
	}
	return nil
}

func getBackupPath(gridDir string, game *Game) string {
	hash := sha256.Sum256(game.OverlayImageBytes)
	// [:] is required to convert a fixed length byte array to a byte slice.
	hexHash := hex.EncodeToString(hash[:])
	return filepath.Join(gridDir, "originals", game.ID+" "+hexHash+game.ImageExt)
}

func RemoveExisting(gridDir string, gameId string) error {
	images, err := filepath.Glob(filepath.Join(gridDir, gameId+".*"))
	if err != nil {
		return err
	}

	backups, err := filepath.Glob(filepath.Join(gridDir, "originals", gameId+" *.*"))
	if err != nil {
		return err
	}

	all := append(images, backups...)
	for _, path := range all {
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadImage(game *Game, sourceName string, imagePath string) error {
	imageBytes, err := ioutil.ReadFile(imagePath)
	if err == nil {
		game.ImageExt = filepath.Ext(imagePath)
		game.CleanImageBytes = imageBytes
		game.ImageSource = sourceName
	}
	return err
}

func LoadExisting(overridePath string, gridDir string, game *Game) {
	overridenIDs, _ := filepath.Glob(filepath.Join(overridePath, game.ID+".*"))
	if overridenIDs != nil && len(overridenIDs) > 0 {
		loadImage(game, "local file in directory 'games'", overridenIDs[0])
		return
	}

	if game.Name != "" {
		re := regexp.MustCompile(`\W+`)
		globName := re.ReplaceAllString(game.Name, "*")
		overridenNames, _ := filepath.Glob(filepath.Join(overridePath, globName+".*"))
		if overridenNames != nil && len(overridenNames) > 0 {
			loadImage(game, "local file in directory games/", overridenNames[0])
			return
		}
	}

	// If there are any old-style backups (without hash), load them over the existing (with overlay) images.
	oldBackups, err := filepath.Glob(filepath.Join(gridDir, game.ID+" (original)*"))
	if err == nil && len(oldBackups) > 0 {
		err = loadImage(game, "legacy backup (now converted)", oldBackups[0])
		if err == nil {
			os.Remove(oldBackups[0])
			return
		}
	}

	files, err := filepath.Glob(filepath.Join(gridDir, game.ID+".*"))
	if err == nil && len(files) > 0 {
		err = loadImage(game, "manual customization", files[0])
		if err == nil {
			game.OverlayImageBytes = game.CleanImageBytes

			// See if there exists a backup image with no overlays or modifications.
			loadImage(game, "backup", getBackupPath(gridDir, game))
		}
	}

}
