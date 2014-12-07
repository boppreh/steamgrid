package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

// A Steam game in a library. May or may not be installed.
type Game struct {
	// Official Steam id.
	Id string
	// Warning, may contain Unicode characters.
	Name string
	// Tags, including user-created category and Steam's "Favorite" tag.
	Tags []string
	// Path for the grid image.
	ImagePath string
	// Raw bytes of the encoded image (usually jpg).
	ImageBytes []byte
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
		gameId := groups[1]
		gameName := groups[2]
		tags := []string{""}
		imagePath := ""
		games[gameId] = &Game{gameId, gameName, tags, imagePath, nil}
	}

	return
}

func addNonSteamGaems(user User, game map[string]*Game) {
}

// Loads the categories list. This finds the categories for the games loaded
// from the profile and sometimes find new games, although without names.
func addUnknownGames(user User, games map[string]*Game) {
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
		gameId := gameGroups[1]
		tagsText := gameGroups[2]

		for _, tagGroups := range tagsPattern.FindAllStringSubmatch(tagsText, -1) {
			tag := tagGroups[1]

			game, ok := games[gameId]
			if ok {
				game.Tags = append(game.Tags, tag)
			} else {
				// If for some reason it wasn't included in the profile, create a new
				// entry for it now. Unfortunately we don't have a name.
				gameName := ""
				games[gameId] = &Game{gameId, gameName, []string{tag}, "", nil}
			}
		}
	}
}

// Adds non-Steam games that have been registered locally.
func addNonSteamGames(user User, games map[string]*Game) {
	// File that holds list of all non-Steam games.
	file := filepath.Join(user.Dir, "760", "screenshots.vdf")
	if _, err := os.Stat(file); err != nil {
		// No custom games, skip this part.
		return
	}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	content := string(bytes)
	// VDF pattern: "customid"		"name"
	gamePattern := regexp.MustCompile(`"([0-9]+)"\s*"(.+?)"`)
	for _, gameGroups := range gamePattern.FindAllStringSubmatch(content, -1) {
		gameId := gameGroups[1]
		name := gameGroups[2]
		games[gameId] = &Game{gameId, name, []string{}, "", nil}
	}
}

// Returns all games from a given user, using both the public profile and local
// files to gather the data. Returns a map of game by ID.
func GetGames(user User) (games map[string]*Game, err error) {
	games = make(map[string]*Game, 0)

	addGamesFromProfile(user, games)
	addUnknownGames(user, games)
	addNonSteamGames(user, games)

	return
}
