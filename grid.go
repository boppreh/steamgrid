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

const imageUrlFormat = `https://steamcdn-a.akamaihd.net/steam/apps/%v/header.jpg`
const alternativeUrlFormat = `http://cdn.steampowered.com/v/gfx/apps/%v/header.jpg`
func tryDownload(gameId string, format string) (*http.Response, error) {

	url := fmt.Sprintf(imageUrlFormat, gameId)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == 404 {
		// Some apps don't have an image and there's nothing we can do.
		return nil, nil
	} else if response.StatusCode > 400 {
		// Other errors should be reported, though.
		return nil, errors.New("Failed to download image " + url + ": " + response.Status)
	}

	return response, nil
}

func DownloadImage(gameId string, gridDir string) (found bool, err error) {
	filename := filepath.Join(gridDir, gameId + ".jpg")
	if _, err := os.Stat(filename); err == nil {
		// File already exists, skip it.
		return true, nil
	}

	var response *http.Response
	response, err = tryDownload(gameId, imageUrlFormat)
	if err != nil || response == nil {
		response, err = tryDownload(gameId, alternativeUrlFormat)
		if err != nil {
			return false, err
		} else if response == nil {
			return false, nil
		}

		fmt.Printf("\n\nDownloaded %v from alternative url.\n\n")
	}
	
	imageBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return true, ioutil.WriteFile(filename, imageBytes, 0666)
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

func PrintProgress(current int, total int) {
	fmt.Print("\r[")
	printedHead := false
	for i := 0; i < 40; i++ {
		part := int(float64(i) * (float64(total) / 40.0))
		if part < current {
			fmt.Print("=")
		} else if !printedHead {
			printedHead = true
			fmt.Print(">")
		} else {
			fmt.Print(" ")
		}
	}
	fmt.Printf("] (%v/%v)", current, total)
}

func errorAndExit(err error) {
	fmt.Println("An unexpected error occurred:")
	fmt.Println(err)
	os.Exit(1)
}

func main() {
	installationDir, err := GetSteamInstallation()
	if err != nil {
		errorAndExit(err)
	}

	users, err := GetUsers(installationDir) 
	if err != nil {
		errorAndExit(err)
	}

	for _, user := range users {
		fmt.Printf("Found user %v. Fetching game list from profile...\n\n\n", user.Name)

		games, err := GetGames(user.Name) 
		if err != nil {
			errorAndExit(err)
		}

		notFounds := make([]Game, 0)
		fmt.Printf("Found %v games. Downloading images...\n\n", len(games))
		for i, game := range games {
			PrintProgress(i+1, len(games))
			gridDir := filepath.Join(user.Dir, "grid")
			found, err := DownloadImage(game.Id, gridDir)
			if err != nil {
				errorAndExit(err)
			}
			if !found {
				notFounds = append(notFounds, game)
			}
		}
		fmt.Print("\n\n\n")

		if len(notFounds) == 0 {
			fmt.Println("All grid images downloaded!")
		} else {
			fmt.Printf("%v images could not be found:\n", len(notFounds))
			for _, game := range notFounds {
				fmt.Printf("* %v (steam id %v)\n", game.Name, game.Id)
			}
		}
	}

	fmt.Print("\n\n")
	fmt.Println("You can press enter to close this window.")
	os.Stdin.Read(make([]byte, 1))
}
