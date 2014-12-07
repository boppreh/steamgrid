// Automatically downloads and configures Steam grid images for all games in a
// given Steam installation.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/boppreh/go-ui"
)

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

// Prints an error and quits.
func errorAndExit(err error) {
	goui.Error("An unexpected error occurred:", err.Error())
	os.Exit(1)
}

func main() {
	goui.Start(func() {
		http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Second * 10

		descriptions := make(chan string)
		progress := make(chan int)

		go goui.Progress("SteamGrid", descriptions, progress, func() { os.Exit(1) })

		startApplication(descriptions, progress)
	})
}

func startApplication(descriptions chan string, progress chan int) {
	descriptions <- "Loading overlays..."
	overlays, err := LoadOverlays(filepath.Join(filepath.Dir(os.Args[0]), "overlays by category"))
	if err != nil {
		errorAndExit(err)
	}
	if len(overlays) == 0 {
		// I'm trying to use a message box here, but for some reason the
		// message appears twice and there's an error a closed channel.
		fmt.Println("No overlays", "No category overlays found. You can put overlay images in the folder 'overlays by category', where the filename is the game category.\n\nContinuing without overlays...")
	}

	descriptions <- "Looking for Steam directory..."
	installationDir, err := GetSteamInstallation()
	if err != nil {
		errorAndExit(err)
	}

	descriptions <- "Loading users..."
	users, err := GetUsers(installationDir)
	if err != nil {
		errorAndExit(err)
	}
	if len(users) == 0 {
		errorAndExit(errors.New("No users found at Steam/userdata. Have you used Steam before in this computer?"))
	}

	notFounds := make([]*Game, 0)
	searchFounds := make([]*Game, 0)

	for _, user := range users {
		descriptions <- "Loading games for " + user.Name

		RestoreBackup(user)

		games, err := GetGames(user)
		if err != nil {
			errorAndExit(err)
		}

		i := 0
		for _, game := range games {
			fmt.Println(game.Name)

			i++
			progress <- i * 100 / len(games)
			var name string
			if game.Name != "" {
				name = game.Name
			} else {
				name = "unknown game with id " + game.Id
			}
			descriptions <- fmt.Sprintf("Processing %v (%v/%v)",
				name, i, len(games))

			found, fromSearch, err := DownloadImage(game, user)
			if err != nil {
				errorAndExit(err)
			}
			if !found {
				notFounds = append(notFounds, game)
				continue
			}
			if fromSearch {
				searchFounds = append(searchFounds, game)
			}

			err = BackupGame(game)
			if err != nil {
				errorAndExit(err)
			}

			err = ApplyOverlay(game, overlays)
			if err != nil {
				errorAndExit(err)
			}
		}
	}

	close(progress)

	message := ""
	if len(notFounds) == 0 && len(searchFounds) == 0 {
		message += "All grid images downloaded and overlays applied!\n\n"
	} else {
		if len(searchFounds) >= 1 {
			message += fmt.Sprintf("%v images were found with a Google search and may not be accurate:\n", len(searchFounds))
			for _, game := range searchFounds {
				message += fmt.Sprintf("* %v (steam id %v)\n", game.Name, game.Id)
			}

			message += "\n\n"
		}

		if len(notFounds) >= 1 {
			message += fmt.Sprintf("%v images could not be found anywhere:\n", len(notFounds))
			for _, game := range notFounds {
				message += fmt.Sprintf("* %v (steam id %v)\n", game.Name, game.Id)
			}

			message += "\n\n"
		}
	}
	message += "Open Steam in grid view to see the results!"

	goui.Info("Results", message)
}
