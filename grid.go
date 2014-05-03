package main

import (
	"fmt"
	"os"
	"regexp"
	"path/filepath"
	"io/ioutil"
	"net/http"
	"errors"
)

type User struct {
	Name string
	Dir string
}

func GetUsers(installationDir string) []User {
	userdataDir := filepath.Join(installationDir, "userdata")
	files, err := ioutil.ReadDir(userdataDir)
	if err != nil {
		panic(err)
	}

	users := make([]User, 0)

	for _, userDir := range files {
		userId := userDir.Name()
		userDir := filepath.Join(userdataDir, userId, "config")
		configFile := filepath.Join(userDir, "localconfig.vdf")

		configBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err)
		}
		pattern := regexp.MustCompile(`"PersonaName"\s*"(.+?)"`)
		username := pattern.FindStringSubmatch(string(configBytes))[1]
		users = append(users, User{username, userDir})
	}

	return users
}

const urlFormat = `http://steamcommunity.com/id/%v/games?tab=all`
func GetProfile(username string) string {
	response, err := http.Get(fmt.Sprintf(urlFormat, username))
	if err != nil {
		panic(err)
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

const imageUrlFormat = `http://cdn.steampowered.com/v/gfx/apps/STORE_APP_ID_HERE/header.jpg`
func DownloadImage(gameid string, filename string) error {
	response, err := http.Get(imageUrlFormat)	
	if err != nil {
		return err
	}
	
	var imageBytes []byte
	imageBytes, err = ioutil.ReadAll(response.Body)
	response.Body.Close()
	return ioutil.WriteFile(filename, imageBytes, 0666)
}

func containsSteam(path string) bool {
	_, err := os.Stat(filepath.Join(path, "Steam"))
	return err != nil
}
func GetSteamInstallation() (path string, err error) {
	if len(os.Args) == 2 {
		argDir := os.Args[1]
		if _, err = os.Stat(argDir); err == nil {
			return argDir, nil
		}
	}

	programFiles86Dir := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Steam")
	if _, err = os.Stat(programFiles86Dir); err == nil {
		return programFiles86Dir, nil
	}

	programFilesDir := filepath.Join(os.Getenv("ProgramFiles"), "Steam")
	if _, err = os.Stat(programFilesDir); err == nil {
		return programFilesDir, nil
	}

	return "", errors.New("Could not find Steam installation folder.")
}

func main() {
	installationDir, err := GetSteamInstallation()
	if err != nil {
		panic(err)
	}

	for _, user := range GetUsers(installationDir) {
		fmt.Printf("Found user %v. Fetching game list...\n", user.Name)
		continue

		games := GetGames(user.Name) 
		fmt.Printf("Found %v games. Download images...\n", len(games))
		for _, game := range games {
			gridDir := filepath.Join(user.Dir, "grid")
			err := DownloadImage(game.Id, gridDir)
			fmt.Println(game.Id, game.Name)
			if err != nil {
				panic(err)
			}
		}
	}
}
