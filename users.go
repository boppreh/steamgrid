package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// User in the local steam installation.
type User struct {
	Name      string
	SteamId32 string
	SteamId64 string
	Dir       string
}

// Used to convert between SteamId32 and SteamId64.
const idConversionConstant = 76561197960265728

// Given the Steam installation dir (NOT the library!), returns all users in
// this computer.
func GetUsers(installationDir string) ([]User, error) {
	userdataDir := filepath.Join(installationDir, "userdata")
	files, err := ioutil.ReadDir(userdataDir)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)

	for _, userDir := range files {
		userId := userDir.Name()
		userDir := filepath.Join(userdataDir, userId)

		configFile := filepath.Join(userDir, "config", "localconfig.vdf")
		// Malformed user directory. Without the localconfig file we can't get
		// the username and the game list, so we skip it.
		if _, err := os.Stat(configFile); err != nil {
			continue
		}

		configBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		// Makes sure the grid directory exists.
		gridDir := filepath.Join(userDir, "config", "grid")
		err = os.MkdirAll(gridDir, 0777)
		if err != nil {
			return nil, err
		}

		// The Linux version of Steam ships with the "grid" dir without executable bit.
		// This in turn denies permission to everything inside the folder. This line is
		// here to ensure we have the correct permission.
		fmt.Println("Setting permission...")
		os.Chmod(gridDir, 0777)

		pattern := regexp.MustCompile(`"PersonaName"\s*"(.+?)"`)
		username := pattern.FindStringSubmatch(string(configBytes))[1]

		steamId32, err := strconv.ParseInt(userId, 10, 64)
		steamId64 := steamId32 + idConversionConstant
		strSteamId64 := strconv.FormatInt(steamId64, 10)
		users = append(users, User{username, userId, strSteamId64, userDir})
	}

	return users, nil
}

// URL to get the game list from the SteamId64.
const profilePermalinkFormat = `http://steamcommunity.com/profiles/%v/games?tab=all`

// The Steam website has the terrible habit of returning 200 OK when requests
// fail, and signaling the error in HTML. So we have to parse the request to
// check if it has failed, and cross our fingers that they don't change the
// message.
const steamProfileErrorMessage = `The specified profile could not be found.`

// Returns the HTML profile from a user from their SteamId32.
func GetProfile(user User) (string, error) {
	response, err := http.Get(fmt.Sprintf(profilePermalinkFormat, user.SteamId64))
	if err != nil {
		return "", err
	}

	if response.StatusCode >= 400 {
		return "", errors.New("Profile not found. Make sure you have a public Steam profile.")
	}

	contentBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}

	profile := string(contentBytes)
	if strings.Contains(profile, steamProfileErrorMessage) {
		return "", errors.New("Profile not found.")
	}

	return profile, nil
}

// Returns the Steam installation directory in Windows. Should work for
// internationalized systems, 32 and 64 bits and users that moved their
// ProgramFiles folder. If a folder is given by program parameter, uses that.
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

	currentUser, err := user.Current()
	if err == nil {
		linuxSteamDir := filepath.Join(currentUser.HomeDir, ".local", "share", "Steam")
		if _, err = os.Stat(linuxSteamDir); err == nil {
			return linuxSteamDir, nil
		}

		linuxSteamDir = filepath.Join(currentUser.HomeDir, ".steam", "steam")
		if _, err = os.Stat(linuxSteamDir); err == nil {
			return linuxSteamDir, nil
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

	return "", errors.New("Could not find Steam installation folder. You can drag and drop the Steam folder into `steamgrid.exe` for a manual override.")
}
