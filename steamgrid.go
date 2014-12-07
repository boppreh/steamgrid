// Automatically downloads and configures Steam grid images for all games in a
// given Steam installation.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/boppreh/go-ui"
	"image/jpeg"
	"image/png"
)

// A Steam game in a library. May or may not be installed.
type Game struct {
	// Official Steam id.
	Id string
	// Warning, may contain Unicode characters.
	Name string
	// Tags, including user-created category and Steam's "Favorite" tag.
	Tags []string
	// Path for the grid image.
	ImagePath string
	// Raw bytes of the encoded image (usually jpg).
	ImageBytes []byte
}

// Pattern of game declarations in the public profile. It's actually JSON
// inside Javascript, but this way is easier to extract.
const profileGamePattern = `\{"appid":\s*(\d+),\s*"name":\s*"(.+?)"`

// Returns all games from a given user, using both the public profile and local
// files to gather the data. Returns a map of game by ID.
func GetGames(user User) (games map[string]*Game, err error) {
	profile, err := GetProfile(user)
	if err != nil {
		return
	}

	// Fetch game list from public profile.
	pattern := regexp.MustCompile(profileGamePattern)
	games = make(map[string]*Game, 0)
	for _, groups := range pattern.FindAllStringSubmatch(profile, -1) {
		gameId := groups[1]
		gameName := groups[2]
		tags := []string{""}
		imagePath := ""
		games[gameId] = &Game{gameId, gameName, tags, imagePath, nil}
	}

	// Fetch game categories from local file.
	sharedConfFile := filepath.Join(user.Dir, "7", "remote", "sharedconfig.vdf")
	if _, err := os.Stat(sharedConfFile); err != nil {
		// No categories file found, skipping this part.
		return games, nil
	}
	sharedConfBytes, err := ioutil.ReadFile(sharedConfFile)

	sharedConf := string(sharedConfBytes)
	// VDF patterN: "steamid" { "tags { "0" "category" } }
	gamePattern := regexp.MustCompile(`"([0-9]+)"\s*{[^}]+?"tags"\s*{([^}]+?)}`)
	tagsPattern := regexp.MustCompile(`"[0-9]+"\s*"(.+?)"`)
	for _, gameGroups := range gamePattern.FindAllStringSubmatch(sharedConf, -1) {
		gameId := gameGroups[1]
		tagsText := gameGroups[2]

		for _, tagGroups := range tagsPattern.FindAllStringSubmatch(tagsText, -1) {
			tag := tagGroups[1]

			game, ok := games[gameId]
			if ok {
				game.Tags = append(game.Tags, tag)
			} else {
				// If for some reason it wasn't included in the profile, create a new
				// entry for it now. Unfortunately we don't have a name.
				gameName := ""
				games[gameId] = &Game{gameId, gameName, []string{tag}, "", nil}
			}
		}
	}

	return
}

// Restores all images that were saved as backup. This avoid double applying
// overlays when running the program multiple times.
func RestoreBackup(user User) {
	baseDir := filepath.Join(user.Dir, "config", "grid")
	entries, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return
	}

	for _, file := range entries {
		if strings.Contains(file.Name(), " (original)") {
			backupPath := filepath.Join(baseDir, file.Name())
			mainPath := strings.Replace(backupPath, " (original)", "", 1)
			bytes, _ := ioutil.ReadFile(backupPath)
			_ = ioutil.WriteFile(mainPath, bytes, 0666)
		}
	}
}

// When all else fails, Google it. Unfortunately this is a deprecated API and
// may go offline at any time. Because this is last resort the number of
// requests shouldn't trigger any punishment.
const googleSearchFormat = `https://ajax.googleapis.com/ajax/services/search/images?v=1.0&rsz=8&q=`

// Returns the first steam grid image URL found by Google search of a given
// game name.
func getGoogleImage(gameName string) (string, error) {
	if gameName == "" {
		return "", nil
	}

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
	matches := pattern.FindStringSubmatch(string(responseBytes))
	if len(matches) >= 1 {
		return matches[1], nil
	} else {
		return "", nil
	}
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
// from a Google search (useful because we want to log the lower quality
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
	jpgFilename := filepath.Join(gridDir, game.Id+".jpg")
	pngFilename := filepath.Join(gridDir, game.Id+".png")

	if imageBytes, err := ioutil.ReadFile(jpgFilename); err == nil {
		game.ImagePath = jpgFilename
		game.ImageBytes = imageBytes
		return true, false, nil
	} else if imageBytes, err := ioutil.ReadFile(pngFilename); err == nil {
		game.ImagePath = pngFilename
		game.ImageBytes = imageBytes
		return true, false, nil
	}

	response, fromSearch, err := getImageAlternatives(game)
	if response == nil || err != nil {
		return false, false, err
	}

	imageBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	game.ImageBytes = imageBytes
	game.ImagePath = jpgFilename
	return true, fromSearch, ioutil.WriteFile(game.ImagePath, game.ImageBytes, 0666)
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
	overlays = make(map[string]image.Image, 0)

	if _, err = os.Stat(dir); err != nil {
		return overlays, nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	for _, file := range files {
		img, err := loadImage(filepath.Join(dir, file.Name()))
		if err != nil {
			return overlays, err
		}

		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		// Normalize overlay name.
		name = strings.TrimRight(strings.ToLower(name), "s")
		overlays[name] = img
	}

	return
}

// Applies an overlay to the game image, depending on the category. The
// resulting image is saved over the original.
func ApplyOverlay(game *Game, overlays map[string]image.Image) (err error) {
	if game.ImagePath == "" || game.ImageBytes == nil || len(game.Tags) == 0 {
		return nil
	}

	gameImage, _, err := image.Decode(bytes.NewBuffer(game.ImageBytes))
	if err != nil {
		return err
	}

	for _, tag := range game.Tags {
		// Normalize tag name by lower-casing it and remove trailing "s" from
		// plurals.
		tagName := strings.TrimRight(strings.ToLower(tag), "s")

		overlayImage, ok := overlays[tagName]
		if !ok {
			continue
		}


		result := image.NewRGBA(gameImage.Bounds().Union(overlayImage.Bounds()))
		draw.Draw(result, result.Bounds(), gameImage, image.ZP, draw.Src)
		draw.Draw(result, result.Bounds(), overlayImage, image.Point{0, 0}, draw.Over)
		gameImage = result
	}

	buf := new(bytes.Buffer)
	if strings.HasSuffix(game.ImagePath, "jpg") {
		err = jpeg.Encode(buf, gameImage, &jpeg.Options{90})
	} else if strings.HasSuffix(game.ImagePath, "png") {
		err = png.Encode(buf, gameImage)
	}
	if err != nil {
		return err
	}
	game.ImageBytes = buf.Bytes()
	return ioutil.WriteFile(game.ImagePath, game.ImageBytes, 0666)
}

func StoreBackup(game *Game) error {
	if game.ImagePath != "" && game.ImageBytes != nil {
		backupPath := strings.Replace(game.ImagePath, ".", " (original).", 1)
		return ioutil.WriteFile(backupPath, game.ImageBytes, 0666)
	} else {
		return nil
	}
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

			err = StoreBackup(game)
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
