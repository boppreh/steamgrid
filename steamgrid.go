// Automatically downloads and configures Steam grid images for all games in a
// given Steam installation.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Prints an error and quits.
func errorAndExit(err error) {
	fmt.Println(err.Error())
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(0)
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
		fmt.Println("No category overlays found. You can put overlay images in the folder 'overlays by category', where the filename is the game category.\n\nContinuing without overlays...")
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
	var notFounds []*Game
	var searchFounds []*Game
	var errors []*Game
	var errorMessages []string

	for _, user := range users {
		fmt.Println("Loading games for " + user.Name)

		games := GetGames(user)

		i := 0
		for _, game := range games {
			i++

			var name string
			if game.Name != "" {
				name = game.Name
			} else {
				name = "unknown game with id " + game.ID
			}
			fmt.Printf("Processing %v (%v/%v)", name, i, len(games))

			if game.ImageBytes == nil {
				err := DownloadImage(game)
				if err != nil {
					errorAndExit(err)
				}
				if game.ImageBytes != nil {
					nDownloaded++
				} else {
					notFounds = append(notFounds, game)
					fmt.Printf(" not found\n")
					// Game has no image, skip it.
					continue
				}
				if game.ImageSource == "search" {
					searchFounds = append(searchFounds, game)
				}
			}

			fmt.Printf(" found from %v\n", game.ImageSource)

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

			err = ioutil.WriteFile(game.ImagePath, game.ImageBytes, 0666)
			if err != nil {
				fmt.Printf("Failed to write image for %v because: %v\n", game.Name, err.Error())
			}
		}
	}

	fmt.Printf("\n\n%v images downloaded and %v overlays applied.\n\n", nDownloaded, nOverlaysApplied)
	if len(searchFounds) >= 1 {
		fmt.Printf("%v images were found with a Google search and may not be accurate:\n", len(searchFounds))
		for _, game := range searchFounds {
			fmt.Printf("* %v (steam id %v)\n", game.Name, game.ID)
		}

		fmt.Printf("\n\n")
	}

	if len(notFounds) >= 1 {
		fmt.Printf("%v images could not be found anywhere:\n", len(notFounds))
		for _, game := range notFounds {
			fmt.Printf("- %v (id %v)\n", game.Name, game.ID)
		}

		fmt.Printf("\n\n")
	}

	if len(errors) >= 1 {
		fmt.Printf("%v images were found but had errors and could not be overlaid:\n", len(errors))
		for i, game := range errors {
			fmt.Printf("- %v (id %v) (%v)\n", game.Name, game.ID, errorMessages[i])
		}

		fmt.Printf("\n\n")
	}

	fmt.Println("Open Steam in grid view to see the results!\n\nPress enter to close.")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
