// Automatically downloads and configures Steam grid images for all games in a
// given Steam installation.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/boppreh/go-ui"
)

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

	nOverlaysApplied := 0
	nDownloaded := 0
	notFounds := make([]*Game, 0)
	searchFounds := make([]*Game, 0)

	for _, user := range users {
		descriptions <- "Loading games for " + user.Name

		RestoreBackup(user)

		games := GetGames(user)

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

			downloaded, found, fromSearch, err := DownloadImage(game, user)
			if err != nil {
				errorAndExit(err)
			}
			if downloaded {
				nDownloaded++
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

			applied, err := ApplyOverlay(game, overlays)
			if err != nil {
				errorAndExit(err)
			}
			if applied {
				nOverlaysApplied++
			}
		}
	}

	close(progress)

	message := fmt.Sprintf("%v images downloaded and %v overlays applied.\n\n", nDownloaded, nOverlaysApplied)
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
			message += fmt.Sprintf("- %v (id %v)\n", game.Name, game.Id)
		}

		message += "\n\n"
	}
	message += "Open Steam in grid view to see the results!"

	goui.Info("Results", message)
}
