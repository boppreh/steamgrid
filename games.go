package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Game in a steam library. May or may not be installed.
type Game struct {
	// Official appID or custom shortcut ID
	ID string
	// Warning, may contain Unicode characters.
	Name string
	// Tags, including user-created category and Steam's "Favorite" tag.
	Tags []string
	// Image format (.jpg, .jpeg, or .png).
	ImageExt string
	// Raw bytes of the encoded image (jpg or png) without overlays.
	CleanImageBytes []byte
	// Raw bytes of the encoded image (jpg or png) with overlays.
	OverlayImageBytes []byte
	// Description of where the image was found (backup, official, search).
	ImageSource string
	// Is custom shortcut?
	Custom bool
	// LegacyID used in BigPicture
	LegacyID uint64
}

// Pattern of game declarations in the public profile. It's actually JSON
// inside Javascript, but this way is easier to extract.
const profileGamePattern = `\{"appid":\s*(\d+),\s*"name":\s*"(.+?)"`

// Fetches the list of games from the public user profile. This is better than
// looking locally because the profiles give the full game name, which can be
// used for image searches later on.
func addGamesFromProfile(user User, games map[string]*Game) (err error) {
	profile, err := GetProfile(user)
	if err != nil {
		return
	}

	// Fetch game list from public profile.
	pattern := regexp.MustCompile(profileGamePattern)
	for _, groups := range pattern.FindAllStringSubmatch(profile, -1) {
		gameID := groups[1]
		gameName := groups[2]
		tags := []string{""}
		games[gameID] = &Game{gameID, gameName, tags, "", nil, nil, "", false, 0}
	}

	return
}

// Loads the categories list. This finds the categories for the games loaded
// from the profile and sometimes find new games, although without names.
func addUnknownGames(user User, games map[string]*Game, skipCategory string) {
	// Fetch game categories from local file.
	sharedConfFile := filepath.Join(user.Dir, "7", "remote", "sharedconfig.vdf")
	if _, err := os.Stat(sharedConfFile); err != nil {
		// No categories file found, skipping this part.
		return
	}
	sharedConfBytes, err := ioutil.ReadFile(sharedConfFile)
	if err != nil {
		return
	}

	sharedConf := string(sharedConfBytes)
	// VDF pattern: "steamid" { "tags { "0" "category" } }
	gamePattern := regexp.MustCompile(`"([0-9]+)"\s*{[^}]+?"tags"\s*{([^}]+?)}`)
	tagsPattern := regexp.MustCompile(`"[0-9]+"\s*"(.+?)"`)
	for _, gameGroups := range gamePattern.FindAllStringSubmatch(sharedConf, -1) {
		gameID := gameGroups[1]
		tagsText := gameGroups[2]

		for _, tagGroups := range tagsPattern.FindAllStringSubmatch(tagsText, -1) {
			tag := tagGroups[1]

			game, ok := games[gameID]
			if ok {
				game.Tags = append(game.Tags, tag)
			} else {
				// If for some reason it wasn't included in the profile, create a new
				// entry for it now. Unfortunately we don't have a name.
				gameName := ""
				games[gameID] = &Game{gameID, gameName, []string{tag}, "", nil, nil, "", false, 0}
			}

			if len(skipCategory) > 0 && strings.Contains(strings.ToLower(tag), strings.ToLower(skipCategory)) {
				delete(games, gameID)
				break
			}
		}
	}
}

// Adds non-Steam games that have been registered locally.
// This information is in the file config/shortcuts.vdf, in binary format.
// It contains the non-Steam games with names, target (exe location) and
// tags/categories. To create a grid image we must compute the Steam ID, which
// is just crc32(target + label) + "02000000", using IEEE standard polynomials.
func addNonSteamGames(user User, games map[string]*Game, skipCategory string) {
	shortcutsVdf := filepath.Join(user.Dir, "config", "shortcuts.vdf")
	if _, err := os.Stat(shortcutsVdf); err != nil {
		return
	}
	shortcutBytes, err := ioutil.ReadFile(shortcutsVdf)
	if err != nil {
		return
	}

	// The actual binary format is known, but using regexes is way easier than
	// parsing the entire file. If I run into any problems I'll replace this.
	gamePattern := regexp.MustCompile("(?i)\x00\x02appid\x00(.{1,4})\x01appname\x00([^\x08]+?)\x00\x01exe\x00([^\x08]+?)\x00\x01.+?\x00tags\x00(?:\x01([^\x08]+?)|)\x08\x08")
	tagsPattern := regexp.MustCompile("\\d\x00([^\x00\x01\x08]+?)\x00")
	for _, gameGroups := range gamePattern.FindAllSubmatch(shortcutBytes, -1) {
		gameID := fmt.Sprint(binary.LittleEndian.Uint32(gameGroups[1]))
		gameName := gameGroups[2]

		// BigPicture is still using these
		target := gameGroups[3]
		uniqueName := bytes.Join([][]byte{target, gameName}, []byte(""))
		LegacyID := uint64(crc32.ChecksumIEEE(uniqueName)) | 0x80000000

		game := Game{gameID, string(gameName), []string{}, "", nil, nil, "", true, LegacyID}
		games[gameID] = &game

		tagsText := gameGroups[4]
		for _, tagGroups := range tagsPattern.FindAllSubmatch(tagsText, -1) {
			tag := tagGroups[1]
			game.Tags = append(game.Tags, string(tag))

			if len(skipCategory) > 0 && strings.Contains(strings.ToLower(string(tag)), strings.ToLower(skipCategory)) {
				delete(games, gameID)
				break
			}
		}
	}
}

// GetGames returns all games from a given user, using both the public profile and local
// files to gather the data. Returns a map of game by ID.
func GetGames(user User, nonSteamOnly bool, appIDs string, skipCategory string) map[string]*Game {
	games := make(map[string]*Game, 0)

	if appIDs != "" {
		for _, appID := range strings.Split(appIDs, ",") {
			games[appID] = &Game{appID, "", []string{}, "", nil, nil, "", false, 0}
		}
		return games
	}

	if !nonSteamOnly {
		addGamesFromProfile(user, games)
		addUnknownGames(user, games, skipCategory)
	}
	addNonSteamGames(user, games, skipCategory)

	return games
}
