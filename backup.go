package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"unicode"
)

// BackupGame if a game has a custom image, backs it up by appending "(original)" to the
// file name.
func backupGame(gridDir string, game *Game, artStyleExtensions []string) error {
	if game.CleanImageBytes != nil {
		return ioutil.WriteFile(getBackupPath(gridDir, game, artStyleExtensions), game.CleanImageBytes, 0666)
	}
	return nil
}

func getBackupPath(gridDir string, game *Game, artStyleExtensions []string) string {
	hash := sha256.Sum256(game.OverlayImageBytes)
	// [:] is required to convert a fixed length byte array to a byte slice.
	hexHash := hex.EncodeToString(hash[:])
	return filepath.Join(gridDir, "originals", game.ID+artStyleExtensions[0]+" "+hexHash+game.ImageExt)
}

func removeExisting(gridDir string, gameID string, artStyleExtensions []string) error {
	images, err := filepath.Glob(filepath.Join(gridDir, gameID+artStyleExtensions[0]+".*"))
	if err != nil {
		return err
	}
	images = filterForImages(images)

	backups, err := filepath.Glob(filepath.Join(gridDir, "originals", gameID+artStyleExtensions[0]+" *.*"))
	if err != nil {
		return err
	}
	backups = filterForImages(backups)

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

// https://wenzr.wordpress.com/2018/04/09/go-glob-case-insensitive/
func insensitiveFilepath(path string) string {
	if runtime.GOOS == "windows" {
		return path
	}

	p := ""
	for _, r := range path {
		if unicode.IsLetter(r) {
			p += fmt.Sprintf("[%c%c]", unicode.ToLower(r), unicode.ToUpper(r))
		} else {
			p += string(r)
		}
	}
	return p
}

func filterForImages(paths []string) []string {
	var matchedPaths []string
	for _, path := range paths {
		ext := filepath.Ext(path)
		switch ext {
		case ".png":
			matchedPaths = append(matchedPaths, path)
		case ".jpg":
			matchedPaths = append(matchedPaths, path)
		case ".jpeg":
			matchedPaths = append(matchedPaths, path)
		}
	}
	return matchedPaths
}

func loadExisting(overridePath string, gridDir string, game *Game, artStyleExtensions []string) {
	overridenIDs, _ := filepath.Glob(filepath.Join(overridePath, game.ID+artStyleExtensions[0]+".*"))
	if overridenIDs != nil && len(overridenIDs) > 0 {
		loadImage(game, "local file in directory 'games'", overridenIDs[0])
		return
	}
	overridenIDs = filterForImages(overridenIDs)

	if game.Name != "" {
		re := regexp.MustCompile(`\W+`)
		globName := re.ReplaceAllString(game.Name, "*")
		overridenNames, _ := filepath.Glob(filepath.Join(overridePath, insensitiveFilepath(globName)+artStyleExtensions[1]+".*"))
		if overridenNames != nil && len(overridenNames) > 0 {
			loadImage(game, "local file in directory games/", overridenNames[0])
			return
		}
	}

	// If there are any old-style backups (without hash), load them over the existing (with overlay) images.
	oldBackups, err := filepath.Glob(filepath.Join(gridDir, game.ID+artStyleExtensions[0]+" (original)*"))
	if err == nil && len(oldBackups) > 0 {
		err = loadImage(game, "legacy backup (now converted)", oldBackups[0])
		if err == nil {
			os.Remove(oldBackups[0])
			return
		}
	}

	files, err := filepath.Glob(filepath.Join(gridDir, game.ID+artStyleExtensions[0]+".*"))
	files = filterForImages(files)
	if err == nil && len(files) > 0 {
		err = loadImage(game, "manual customization", files[0])
		if err == nil {
			// set as overlay to check for hash in getBackupPath()
			game.OverlayImageBytes = game.CleanImageBytes

			// See if there exists a backup image with no overlays or modifications.
			loadImage(game, "backup", getBackupPath(gridDir, game, artStyleExtensions))

			// remove overlay
			game.OverlayImageBytes = nil
		}
	}

}
