package main

import (
	"fmt"
	"regexp"
	"path/filepath"
	"io/ioutil"
	"net/http"
)

func GetUsernames(installationDir string) []string {
	files, err := ioutil.ReadDir(installationDir)
	if err != nil {
		panic(err)
	}

	names := make([]string, 0)

	for _, userDir := range files {
		userId := userDir.Name()
		configFile := filepath.Join(installationDir, userId, "config", "localconfig.vdf")

		configBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err)
		}
		pattern := regexp.MustCompile(`"PersonaName"\s*"(.+?)"`)
		username := pattern.FindStringSubmatch(string(configBytes))[1]
		names = append(names, username)
	}

	return names
}

const urlFormat = `http://steamcommunity.com/id/%v/games?tab=all`
func GetProfile(username string) string {
	fmt.Println("Processing user", username)
	response, err := http.Get(fmt.Sprintf(urlFormat, username))
	if err != nil {
		panic(err)
	}

	if response.StatusCode != 200 {
		panic("Profile not found, server returned " + response.Status)
	}
	contentBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		panic(err)
	}

	return string(contentBytes)
}

type Game struct {
	Id string
	Name string
}

const gamePattern = `\{"appid":\s*(\d+),\s*"name":\s*"(.+?)"`
func GetGames(username string) []Game {
	pattern := regexp.MustCompile(gamePattern)

	games := make([]Game, 0)

	profile := GetProfile(username)
	for _, groups := range pattern.FindAllStringSubmatch(profile, -1) {
		games = append(games, Game{groups[1], groups[2]})
	}

	return games
}

func main() {
	for _, username := range GetUsernames(`C:\Program Files (x86)\Steam\userdata`) {
		for _, game := range GetGames(username) {
			fmt.Println(game.Id, game.Name)
		}
	}
}
