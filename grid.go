// Automatically downloads and configures Steam grid images for all games in a
// given Steam installation.
package main

import (
	"image"
	"image/draw"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"image/jpeg"
	_ "image/png"
)

// User in the local steam installation.
type User struct {
	Name string
	Dir  string
}

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

// Steam profile URL format.
const urlFormat = `http://steamcommunity.com/id/%v/games?tab=all`

// Returns the public Steam profile for a given user, in HTML.
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

// A Steam game in a library. May or may not be installed.
type Game struct {
	// Official Steam id.
	Id string
	// Warning, may contain Unicode characters.
	Name string
	// User created category. May be blank.
	Category string
	// Path for the grid image.
	ImagePath string
}

// Pattern of game declarations in the public profile. It's actually JSON
// inside Javascript, but this way is easier to extract.
const profileGamePattern = `\{"appid":\s*(\d+),\s*"name":\s*"(.+?)"`

// Returns all games from a given user, using both the public profile and local
// files to gather the data. Returns a map of game by ID.
func GetGames(user User) (map[string]*Game, error) {
	profile, err := GetProfile(user.Name)
	if err != nil {
		return nil, err
	}

	// Fetch game list from public profile.
	pattern := regexp.MustCompile(profileGamePattern)
	games := make(map[string]*Game, 0)
	for _, groups := range pattern.FindAllStringSubmatch(profile, -1) {
		gameId := groups[1]
		gameName := groups[2]
		category := ""
		imagePath := ""
		games[gameId] = &Game{gameId, gameName, category, imagePath}
	}

	// Fetch game categories from local file.
	sharedConfFile := filepath.Join(user.Dir, "7", "remote", "sharedconfig.vdf")
	sharedConfBytes, err := ioutil.ReadFile(sharedConfFile)

	sharedConf := string(sharedConfBytes)
	// VDF patterN: "steamid" { "tags { "0" "category" } }
	pattern = regexp.MustCompile(`"([0-9]+)"\s*{[^}]+?"tags"\s*{\s*"0"\s*"([^"]+)"`)
	for _, groups := range pattern.FindAllStringSubmatch(sharedConf, -1) {
		gameId := groups[1]
		category := groups[2]

		game, ok := games[gameId]
		if ok {
			game.Category = category
		} else {
			// If for some reason it wasn't included in the profile, create a new
			// entry for it now. Unfortunately we don't have a name.
			gameName := ""
			games[gameId] = &Game{gameId, gameName, category, ""}
		}
	}

	return games, nil
}

// When all else fails, Google it. Unfortunately this is a deprecated API and
// may go offline at any time. Because this is last resort the number of
// requests shouldn't trigger any punishment.
const googleSearchFormat = `https://ajax.googleapis.com/ajax/services/search/images?v=1.0&q=`

// Returns the first steam grid image URL found by Google search of a given
// game name.
func getGoogleImage(gameName string) (string, error) {
	url := googleSearchFormat + url.QueryEscape("steam grid OR header"+gameName)
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	response.Body.Close()
	// Again, we could parse JSON. This may be a little too lazy, the pattern
	// is very loose. The order could be wrong, for example.
	pattern := regexp.MustCompile(`"width":"460","height":"215",[^}]+"unescapedUrl":"(.+?)"`)
	imageUrl := pattern.FindStringSubmatch(string(responseBytes))[1]
	return imageUrl, nil
}

// Tries to fetch a URL, returning the response only if it was positive.
func tryDownload(url string) (*http.Response, error) {
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

// Primary URL for downloading grid images.
const akamaiUrlFormat = `https://steamcdn-a.akamaihd.net/steam/apps/%v/header.jpg`

// The subreddit mentions this as primary, but I've found Akamai to contain
// more images and answer faster.
const steamCdnUrlFormat = `http://cdn.steampowered.com/v/gfx/apps/%v/header.jpg`

// Tries to load the grid image for a game from a number of alternative
// sources. Returns the final response received and a flag indicating if it was
// fro ma Google search (useful because we want to log the lower quality
// images).
func getImageAlternatives(game *Game) (response *http.Response, fromSearch bool, err error) {
	response, err = tryDownload(fmt.Sprintf(akamaiUrlFormat, game.Id))
	if err == nil && response != nil {
		return
	}

	response, err = tryDownload(fmt.Sprintf(steamCdnUrlFormat, game.Id))
	if err == nil && response != nil {
		return
	}

	fromSearch = true
	url, err := getGoogleImage(game.Name)
	if err != nil {
		return
	}
	response, err = tryDownload(url)
	if err == nil && response != nil {
		return
	}

	return nil, false, nil
}

// Downloads the grid image for a game into the user's grid directory. Returns
// flags indicating if the operation succeeded and if the image downloaded was
// from a search.
func DownloadImage(game *Game, user User) (found bool, fromSearch bool, err error) {
	gridDir := filepath.Join(user.Dir, "config", "grid")
	filename := filepath.Join(gridDir, game.Id+".jpg")
	if _, err := os.Stat(filename); err == nil {
		// File already exists, skip it.
		return true, false, nil
	}

	response, fromSearch, err := getImageAlternatives(game)
	if response == nil || err != nil {
		return false, false, err
	}

	imageBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	game.ImagePath = filename
	return true, fromSearch, ioutil.WriteFile(filename, imageBytes, 0666)
}

// Loads an image from a given path.
func loadImage(path string) (img image.Image, err error) {
	reader, err := os.Open(path)
	if err != nil {
		return
	}
	defer reader.Close()

	img, _, err = image.Decode(reader)
	return
}
// Loads the overlays from the given dir, returning a map of name -> image.
func LoadOverlays(dir string) (overlays map[string]image.Image, err error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	overlays = make(map[string]image.Image, 0)

	for _, file := range files {
		img, err := loadImage(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}

		// Normalize overlay name.
		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		overlays[strings.ToLower(name)] = img
	}

	return
}

func ApplyOverlay(game *Game, overlays map[string]image.Image) (err error) {
	gameImage, err := loadImage(game.ImagePath)
	if err != nil {
		return err
	}

	// Normalize overlay name.
	categoryName := strings.ToLower(game.Category)

	overlayImage, ok := overlays[categoryName]
	if !ok {
		return
	}

    result := image.NewRGBA(gameImage.Bounds().Union(overlayImage.Bounds()))
    draw.Draw(result, result.Bounds(), gameImage, image.ZP, draw.Src)
    draw.Draw(result, result.Bounds(), overlayImage, image.Point{0,0}, draw.Over)

    resultFile, _ := os.Create(game.ImagePath)
    defer resultFile.Close()

    return jpeg.Encode(resultFile, result, &jpeg.Options{90})
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

// Prints a progress bar, overriding the previous line. It looks like this:
// [=========>        ] (50/100)
func PrintProgress(current int, total int) {
	// \r moves the cursor back to the start of the line.
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

// Prints an error and quits.
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

		games, err := GetGames(user)
		if err != nil {
			errorAndExit(err)
		}

		notFounds := make([]*Game, 0)
		searchFounds := make([]*Game, 0)
		fmt.Printf("Found %v games. Downloading images...\n\n", len(games))

		i := 0
		for _, game := range games {
			i++
			PrintProgress(i, len(games))
			found, fromSearch, err := DownloadImage(game, user)
			if err != nil {
				errorAndExit(err)
			}
			if !found {
				notFounds = append(notFounds, game)
			}
			if fromSearch {
				searchFounds = append(searchFounds, game)
			}
		}
		fmt.Print("\n\n\n")

		if len(notFounds) == 0 && len(searchFounds) == 0 {
			fmt.Println("All grid images downloaded!")
		} else {
			if len(searchFounds) >= 1 {
				fmt.Printf("%v images were found with a Google search and may not be accurate:.\n", len(searchFounds))
				for _, game := range searchFounds {
					fmt.Printf("* %v (steam id %v)\n", game.Name, game.Id)
				}
			}

			fmt.Print("\n\n")

			if len(notFounds) >= 1 {
				fmt.Printf("%v images could not be found:\n", len(notFounds))
				for _, game := range notFounds {
					fmt.Printf("* %v (steam id %v)\n", game.Name, game.Id)
				}
			}
		}
	}

	fmt.Print("\n\n")
	fmt.Println("You can press enter to close this window.")
	os.Stdin.Read(make([]byte, 1))
}
