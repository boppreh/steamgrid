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
)

// Prints an error and quits.
func errorAndExit(err error) {
	panic(err.Error())
}

func main() {
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Second * 10
	startApplication()
}

func startApplication() {
	fmt.Println("Loading overlays...")
	overlays, err := LoadOverlays(filepath.Join(filepath.Dir(os.Args[0]), "overlays by category"))
	if err != nil {
		errorAndExit(err)
	}
	if len(overlays) == 0 {
		// I'm trying to use a message box here, but for some reason the
		// message appears twice and there's an error a closed channel.
		fmt.Println("No overlays", "No category overlays found. You can put overlay images in the folder 'overlays by category', where the filename is the game category.\n\nContinuing without overlays...")
	}

	fmt.Println("Looking for Steam directory...")
	installationDir, err := GetSteamInstallation()
	if err != nil {
		errorAndExit(err)
	}

	fmt.Println("Loading users...")
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
	errors := make([]*Game, 0)
	errorMessages := make([]string, 0)

	for _, user := range users {
		fmt.Println("Loading games for " + user.Name)

		RestoreBackup(user)

		games := GetGames(user)

		i := 0
		for _, game := range games {
			i += 1
			
			var name string
			if game.Name != "" {
				name = game.Name
			} else {
				name = "unknown game with id " + game.Id
			}
			fmt.Printf("Processing %v (%v/%v)\n", name, i, len(games))

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
				print(err.Error(), "\n")
				errors = append(errors, game)
				errorMessages = append(errorMessages, err.Error())
			}
			if applied {
				nOverlaysApplied++
			}
		}
	}

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

	if len(errors) >= 1 {
		message += fmt.Sprintf("%v images were found but had errors and could not be overlaid:\n", len(errors))
		for i, game := range errors {
			message += fmt.Sprintf("- %v (id %v) (%v)\n", game.Name, game.Id, errorMessages[i])
		}

		message += "\n\n"
	}

	message += "Open Steam in grid view to see the results!"

	fmt.Println(message)
}
