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
		fmt.Println("No category overlays found. You can put overlay images in the folder 'overlays by category', where the filename is the game category.\n\nYou can find many user-created overlays at https://www.reddit.com/r/steamgrid/wiki/overlays .\n\nContinuing without overlays...\n")
	} else {
		fmt.Printf("Loaded %v overlays. \n\nYou can find many user-created overlays at https://www.reddit.com/r/steamgrid/wiki/overlays .\n\n", len(overlays))
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
	var searchedGames []*Game
	var failedGames []*Game
	var errorMessages []string

	for _, user := range users {
		fmt.Println("Loading games for " + user.Name)
		gridDir := filepath.Join(user.Dir, "config", "grid")

		err = os.MkdirAll(filepath.Join(gridDir, "originals"), 0777)
		if err != nil {
			errorAndExit(err)
		}

		games := GetGames(user)

		fmt.Println("Loading existing images and backups...")

		i := 0
		for _, game := range games {
			i++

			overridePath := filepath.Join(filepath.Dir(os.Args[0]), "games")
			LoadExisting(overridePath, gridDir, game)
			// This cleans up unused backups and images for the same game but with different extensions.
			err = RemoveExisting(gridDir, game.ID)
			if err != nil {
				fmt.Println(err.Error())
			}

			var name string
			if game.Name == "" {
				game.Name = GetGameName(game.ID)
			}

			if game.Name != "" {
				name = game.Name
			} else {
				name = "unknown game with id " + game.ID
			}
			fmt.Printf("Processing %v (%v/%v)", name, i, len(games))

			///////////////////////
			// Download if missing.
			///////////////////////
			if game.ImageSource == "" {
				fromSearch, err := DownloadImage(gridDir, game)
				if err != nil {
					fmt.Println(err.Error())
				}

				if game.ImageSource == "" {
					notFounds = append(notFounds, game)
					fmt.Printf(" not found\n")
					// Game has no image, skip it.
					continue
				} else if err == nil {
					nDownloaded++
				}

				if fromSearch {
					searchedGames = append(searchedGames, game)
				}
			}
			fmt.Printf(" found from %v\n", game.ImageSource)

			///////////////////////
			// Apply overlay.
			///////////////////////
			err := ApplyOverlay(game, overlays)
			if err != nil {
				print(err.Error(), "\n")
				failedGames = append(failedGames, game)
				errorMessages = append(errorMessages, err.Error())
			}
			if game.OverlayImageBytes != nil {
				nOverlaysApplied++
			} else {
				game.OverlayImageBytes = game.CleanImageBytes
			}

			///////////////////////
			// Save result.
			///////////////////////
			err = BackupGame(gridDir, game)
			if err != nil {
				errorAndExit(err)
			}
			imagePath := filepath.Join(gridDir, game.ID+game.ImageExt)
			err = ioutil.WriteFile(imagePath, game.OverlayImageBytes, 0666)
			if err != nil {
				fmt.Printf("Failed to write image for %v because: %v\n", game.Name, err.Error())
			}
		}
	}

	fmt.Printf("\n\n%v images downloaded and %v overlays applied.\n\n", nDownloaded, nOverlaysApplied)
	if len(searchedGames) >= 1 {
		fmt.Printf("%v images were found with a Google search and may not be accurate:\n", len(searchedGames))
		for _, game := range searchedGames {
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

	if len(failedGames) >= 1 {
		fmt.Printf("%v images were found but had errors and could not be overlaid:\n", len(failedGames))
		for i, game := range failedGames {
			fmt.Printf("- %v (id %v) (%v)\n", game.Name, game.ID, errorMessages[i])
		}

		fmt.Printf("\n\n")
	}

	fmt.Println("Open Steam in grid view to see the results!\n\nPress enter to close.")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
