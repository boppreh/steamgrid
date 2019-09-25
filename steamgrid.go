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
	// Dealing with Banner or Cover, maybe more in the future (hero?)
	artStyles := map[string][]string{
		// artStyle: ["idExtension", "fileExtension", steamExtension, googleExtensionX, googleExtensionY, steamGridDB]
		"Banner": []string{"", "", "header.jpg", "460", "215", "dimensions=legacy"},
		"Cover": []string{"p", ".p", "library_600x900_2x.jpg", "600", "900", "dimensions=600x900"},
	}

	steamGridDBApiKey := "";

	// Works for now, should be replaced by something better when more are added.
	// Maybe https://github.com/jessevdk/go-flags ?
	if len(os.Args) >= 3 {
		if os.Args[1] == "--steamgriddb" {
			steamGridDBApiKey = os.Args[2]
			fmt.Println("Support for SteamGridDB activated")
		}
	}

	fmt.Println("Loading overlays...")
	overlays, err := LoadOverlays(filepath.Join(filepath.Dir(os.Args[0]), "overlays by category"), artStyles)
	if err != nil {
		errorAndExit(err)
	}
	if len(overlays) == 0 {
		fmt.Println("No category overlays found. You can put overlay images in the folder 'overlays by category', where the filename is the game category.\n\nYou can find many user-created overlays at https://www.reddit.com/r/steamgrid/wiki/overlays .\n\nContinuing without overlays...\n")
	} else {
		fmt.Printf("Loaded %v overlays. \n\nYou can find many user-created overlays at https://www.reddit.com/r/steamgrid/wiki/overlays .\n\n", len(overlays))
	}

	fmt.Println("Looking for Steam directory...\nIf SteamGrid doesn´t find the directory automatically, launch it with an argument linking to the Steam directory.")
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
	var notFoundsBanner []*Game
	var notFoundsCover []*Game
	var steamGridDBBanner []*Game
	var steamGridDBCover []*Game
	var searchedGamesBanner []*Game
	var searchedGamesCover []*Game
	var failedGamesBanner []*Game
	var failedGamesCover []*Game
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

			var name string
			if game.Name == "" {
				game.Name = GetGameName(game.ID)
			}

			if game.Name != "" {
				name = game.Name
			} else {
				name = "unknown game with id " + game.ID
			}
			fmt.Printf("Processing %v (%v/%v)\n", name, i, len(games))

			for artStyle, artStyleExtensions := range artStyles {
				// Clear for multiple runs:
				game.ImageSource = ""
				game.ImageExt = ""
				game.CleanImageBytes = nil
				game.OverlayImageBytes = nil

				overridePath := filepath.Join(filepath.Dir(os.Args[0]), "games")
				LoadExisting(overridePath, gridDir, game, artStyleExtensions)
				// This cleans up unused backups and images for the same game but with different extensions.
				err = RemoveExisting(gridDir, game.ID, artStyleExtensions)
				if err != nil {
					fmt.Println(err.Error())
				}

				///////////////////////
				// Download if missing.
				///////////////////////
				if game.ImageSource == "" {
					from, err := DownloadImage(gridDir, game, artStyle, artStyleExtensions, steamGridDBApiKey)
					if err != nil && err.Error() == "401" {
						// Wrong api key
						steamGridDBApiKey = ""
						fmt.Println("Api key rejected, disabling SteamGridDB.")
					} else if err != nil {
						fmt.Println(err.Error())
					}

					if game.ImageSource == "" {
						if artStyle == "Banner" {
							notFoundsBanner = append(notFoundsBanner, game)
						} else if artStyle == "Cover" {
							notFoundsCover = append(notFoundsCover, game)
						}
						fmt.Printf("%v not found\n", artStyle)
						// Game has no image, skip it.
						continue
					} else if err == nil {
						nDownloaded++
					}

					if from == "SteamGridDB" {
						if artStyle == "Banner" {
							steamGridDBBanner = append(steamGridDBBanner, game)
						} else if artStyle == "Cover" {
							steamGridDBCover = append(steamGridDBCover, game)
						}
					} else if from == "search" {
						if artStyle == "Banner" {
							searchedGamesBanner = append(searchedGamesBanner, game)
						} else if artStyle == "Cover" {
							searchedGamesCover = append(searchedGamesCover, game)
						}
					}
				}
				fmt.Printf("%v found from %v\n", artStyle, game.ImageSource)

				///////////////////////
				// Apply overlay.
				//
				// Expecting name.artExt.imgExt:
				// Banner: favorites.png
				// Cover: favorites.p.png
				///////////////////////
				err := ApplyOverlay(game, overlays, artStyleExtensions)
				if err != nil {
					print(err.Error(), "\n")
					if artStyle == "Banner" {
						failedGamesBanner = append(failedGamesBanner, game)
					} else if artStyle == "Cover" {
						failedGamesCover = append(failedGamesCover, game)
					}
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
				err = BackupGame(gridDir, game, artStyleExtensions)
				if err != nil {
					errorAndExit(err)
				}

				imagePath := filepath.Join(gridDir, game.ID + artStyleExtensions[0] + game.ImageExt)
				err = ioutil.WriteFile(imagePath, game.OverlayImageBytes, 0666)
				if err != nil {
					fmt.Printf("Failed to write image for %v (%v) because: %v\n", game.Name, artStyle, err.Error())
				}
			}
		}
	}

	fmt.Printf("\n\n%v images downloaded and %v overlays applied.\n\n", nDownloaded, nOverlaysApplied)
	if len(searchedGamesBanner) + len(searchedGamesCover) >= 1 {
		fmt.Printf("%v images were found with a Google search and may not be accurate:\n", len(searchedGamesBanner) + len(searchedGamesCover))
		for _, game := range searchedGamesBanner {
			fmt.Printf("* %v (steam id %v, Banner)\n", game.Name, game.ID)
		}
		for _, game := range searchedGamesCover {
			fmt.Printf("* %v (steam id %v, Cover)\n", game.Name, game.ID)
		}


		fmt.Printf("\n\n")
	}

	if len(steamGridDBBanner) + len(steamGridDBCover) >= 1 {
		fmt.Printf("%v images were found on SteamGridDB and may not be in full quality or accurate:\n", len(steamGridDBBanner) + len(steamGridDBCover))
		for _, game := range steamGridDBBanner {
			fmt.Printf("* %v (steam id %v, Banner)\n", game.Name, game.ID)
		}
		for _, game := range steamGridDBCover {
			fmt.Printf("* %v (steam id %v, Cover)\n", game.Name, game.ID)
		}


		fmt.Printf("\n\n")
	}

	if len(notFoundsBanner) + len(notFoundsCover) >= 1 {
		fmt.Printf("%v images could not be found anywhere:\n", len(notFoundsBanner) + len(notFoundsCover))
		for _, game := range notFoundsBanner {
			fmt.Printf("- %v (id %v, Banner)\n", game.Name, game.ID)
		}
		for _, game := range notFoundsCover {
			fmt.Printf("- %v (id %v, Cover)\n", game.Name, game.ID)
		}

		fmt.Printf("\n\n")
	}

	if len(failedGamesBanner) + len(failedGamesCover) >= 1 {
		fmt.Printf("%v images were found but had errors and could not be overlaid:\n", len(failedGamesBanner) + len(failedGamesCover))
		for i, game := range failedGamesBanner {
			fmt.Printf("- %v (id %v, Banner) (%v)\n", game.Name, game.ID, errorMessages[i])
		}
		for i, game := range failedGamesCover {
			fmt.Printf("- %v (id %v, Cover) (%v)\n", game.Name, game.ID, errorMessages[i])
		}

		fmt.Printf("\n\n")
	}

	fmt.Println("Open Steam in grid view to see the results!\n\nPress enter to close.")

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
