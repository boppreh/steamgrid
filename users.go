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
	SteamID32 string
	SteamID64 string
	Dir       string
}

// Used to convert between SteamId32 and SteamId64.
const idConversionConstant = 0x110000100000000

// GetUsers given the Steam installation dir (NOT the library!), returns all users in
// this computer.
func GetUsers(installationDir string) ([]User, error) {
	userdataDir := filepath.Join(installationDir, "userdata")
	files, err := ioutil.ReadDir(userdataDir)
	if err != nil {
		return nil, err
	}

	var users []User

	for _, userDir := range files {
		userID := userDir.Name()
		userDir := filepath.Join(userdataDir, userID)

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

		steamID32, err := strconv.ParseInt(userID, 10, 64)
		steamID64 := steamID32 + idConversionConstant
		strSteamID64 := strconv.FormatInt(steamID64, 10)
		users = append(users, User{username, userID, strSteamID64, userDir})
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

// GetProfile returns the HTML profile from a user from their SteamId32.
func GetProfile(user User) (string, error) {
	response, err := http.Get(fmt.Sprintf(profilePermalinkFormat, user.SteamID64))
	if err != nil {
		return "", err
	}

	if response.StatusCode >= 400 {
		return "", errors.New("Profile not found. Make sure you have a public Steam profile")
	}

	contentBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}

	profile := string(contentBytes)
	if strings.Contains(profile, steamProfileErrorMessage) {
		return "", errors.New("Profile not found")
	}

	return profile, nil
}

// GetSteamInstallation Returns the Steam installation directory in Windows. Should work for
// internationalized systems, 32 and 64 bits and users that moved their
// ProgramFiles folder. If a folder is given by program parameter, uses that.
func GetSteamInstallation(steamDir string) (path string, err error) {
	if steamDir != "" {
		_, err := os.Stat(steamDir)
		if err == nil {
			return steamDir, nil
		}
		return "", errors.New("Argument must be a valid Steam directory, or empty for auto detection. Got: " + steamDir)
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

		macSteamDir := filepath.Join(currentUser.HomeDir, "Library", "Application Support", "Steam")
		if _, err = os.Stat(macSteamDir); err == nil {
			return macSteamDir, nil
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

	return "", errors.New("Could not find Steam installation folder. You can drag and drop the Steam folder into `steamgrid.exe` or call `steamgrid STEAMPATH` for a manual override")
}
