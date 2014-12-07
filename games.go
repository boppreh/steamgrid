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

// Returns all games from a given user, using both the public profile and local
// files to gather the data. Returns a map of game by ID.
func GetGames(user User) (games map[string]*Game, err error) {
	profile, err := GetProfile(user)
	if err != nil {
		return
	}

	// Fetch game list from public profile.
	pattern := regexp.MustCompile(profileGamePattern)
	games = make(map[string]*Game, 0)
	for _, groups := range pattern.FindAllStringSubmatch(profile, -1) {
		gameId := groups[1]
		gameName := groups[2]
		tags := []string{""}
		imagePath := ""
		games[gameId] = &Game{gameId, gameName, tags, imagePath, nil}
	}

	// Fetch game categories from local file.
	sharedConfFile := filepath.Join(user.Dir, "7", "remote", "sharedconfig.vdf")
	if _, err := os.Stat(sharedConfFile); err != nil {
		// No categories file found, skipping this part.
		return games, nil
	}
	sharedConfBytes, err := ioutil.ReadFile(sharedConfFile)

	sharedConf := string(sharedConfBytes)
	// VDF patterN: "steamid" { "tags { "0" "category" } }
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

	return
}
