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

func GetUsers(installationDir string) ([]User, error) {
	userdataDir := filepath.Join(installationDir, "userdata")
	files, err := ioutil.ReadDir(userdataDir)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)

	for _, userDir := range files {
		userId := userDir.Name()
		userDir := filepath.Join(userdataDir, userId, "config")
		configFile := filepath.Join(userDir, "localconfig.vdf")

		configBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		pattern := regexp.MustCompile(`"PersonaName"\s*"(.+?)"`)
		username := pattern.FindStringSubmatch(string(configBytes))[1]
		users = append(users, User{username, userDir})
	}

	return users, nil
}

const urlFormat = `http://steamcommunity.com/id/%v/games?tab=all`
func GetProfile(username string) (string, error) {
	response, err := http.Get(fmt.Sprintf(urlFormat, username))
	if err != nil {
		return "", err
	}

	contentBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}

	return string(contentBytes), nil
}

type Game struct {
	Id string
	Name string
}

const gamePattern = `\{"appid":\s*(\d+),\s*"name":\s*"(.+?)"`
func GetGames(username string) ([]Game, error) {
	pattern := regexp.MustCompile(gamePattern)

	games := make([]Game, 0)

	profile, err := GetProfile(username)
	if err != nil {
		return nil, err
	}

	for _, groups := range pattern.FindAllStringSubmatch(profile, -1) {
		games = append(games, Game{groups[1], groups[2]})
	}

	return games, nil
}

const imageUrlFormat = `http://cdn.steampowered.com/v/gfx/apps/STORE_APP_ID_HERE/header.jpg`
func DownloadImage(gameid string, filename string) error {
	response, err := http.Get(imageUrlFormat)	
	if err != nil {
		return err
	}
	
	imageBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return ioutil.WriteFile(filename, imageBytes, 0666)
}

func GetSteamInstallation() (path string, err error) {
	if len(os.Args) == 2 {
		argDir := os.Args[1]
		_, err := os.Stat(argDir)
		if err == nil {
			return argDir, nil
		} else {
			return "", errors.New("Argument must be a valid Steam directory, or empty for auto detection. Got: " + argDir)
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

	users, err := GetUsers(installationDir) 
	if err != nil {
		panic(err)
	}

	for _, user := range users {
		fmt.Printf("Found user %v. Fetching game list...\n", user.Name)
		continue

		games, err := GetGames(user.Name) 
		if err != nil {
			panic(err)
		}

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
