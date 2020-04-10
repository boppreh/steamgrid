package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/deanishe/awgo/fuzzy"
)

// When all else fails, Google it. Uses the regular web interface. There are
// two image search APIs, but one is deprecated and doesn't support exact size
// matching, and the other requires an API key limited to 100 searches a day.
const googleSearchFormat = `https://www.google.com.br/search?tbs=isz%%3Aex%%2Ciszw%%3A%v%%2Ciszh%%3A%v&tbm=isch&num=5&q=`

// Possible Google result formats
var googleSearchResultPatterns = []string{`imgurl=(.+?\.(jpeg|jpg|png))&amp;imgrefurl=`, `\"ou\":\"(.+?)\",\"`}

// Returns the first steam grid image URL found by Google search of a given
// game name.
func getGoogleImage(gameName string, artStyleExtensions []string) (string, error) {
	if gameName == "" {
		return "", nil
	}

	url := fmt.Sprintf(googleSearchFormat, artStyleExtensions[5], artStyleExtensions[6]) + url.QueryEscape(gameName)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// If we don't set an user agent, Google will block us because we are a
	// bot. If we set something like "SteamGrid Image Search" it'll work, but
	// Google will serve a simple HTML page without direct image links.
	// So we have to lie.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	response.Body.Close()

	for _, googleSearchResultPattern := range googleSearchResultPatterns {
		pattern := regexp.MustCompile(googleSearchResultPattern)
		matches := pattern.FindStringSubmatch(string(responseBytes))

		if len(matches) >= 1 {
			return matches[1], nil
		}
	}
	return "", nil
}

// https://www.steamgriddb.com/api/v2
type steamGridDBResponse struct {
	Success bool
	Data    []struct {
		ID     int
		Score  int
		Style  string
		URL    string
		Thumb  string
		Tags   []string
		Author struct {
			Name    string
			Steam64 string
			Avatar  string
		}
	}
}

type steamGridDBSearchResponse struct {
	Success bool
	Data    []struct {
		ID       int
		Name     string
		Types    []string
		Verified bool
	}
}

// Enable fuzzy sorting
// Default sort.Interface methods
func (results steamGridDBSearchResponse) Len() int { return len(results.Data) }
func (results steamGridDBSearchResponse) Swap(i, j int) {
	results.Data[i], results.Data[j] = results.Data[j], results.Data[i]
}
func (results steamGridDBSearchResponse) Less(i, j int) bool {
	return results.Data[i].Name < results.Data[j].Name
}

// Keywords implements Sortable.
// Comparisons are based on the the full name of the contact.
func (results steamGridDBSearchResponse) Keywords(i int) string { return results.Data[i].Name }

// Search SteamGridDB for cover image
const steamGridDBBaseURL = "https://www.steamgriddb.com/api/v2"

func steamGridDBGetRequest(url string, steamGridDBApiKey string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+steamGridDBApiKey)

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == 401 {
		// Authorization token is missing or invalid
		return nil, errors.New("401")
	} else if response.StatusCode == 404 {
		// Could not find game with that id
		return nil, errors.New("404")
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	response.Body.Close()

	return responseBytes, nil
}

func getSteamGridDBImage(game *Game, artStyleExtensions []string, steamGridDBApiKey string, steamGridFilter string) (string, error) {
	// Try for HQ, then for LQ
	// It's possible to request both dimensions in one go but that'll give us scrambled results with no indicator which result has which size.
	for i := 0; i < 3; i += 2 {
		filter := steamGridFilter + "&dimensions=" + artStyleExtensions[3+i] + "x" + artStyleExtensions[4+i]

		// Try with game.ID which is probably steams appID
		var baseURL string
		switch artStyleExtensions[1] {
		case ".banner":
			baseURL = steamGridDBBaseURL + "/grids"
		case ".cover":
			baseURL = steamGridDBBaseURL + "/grids"
		case ".hero":
			baseURL = steamGridDBBaseURL + "/heroes"
		case ".logo":
			baseURL = steamGridDBBaseURL + "/logos"
		}
		url := baseURL + "/steam/" + game.ID + filter

		var jsonResponse steamGridDBResponse
		var responseBytes []byte
		var err error

		// Skip requests with appID for custom games
		if !game.Custom {
			responseBytes, err = steamGridDBGetRequest(url, steamGridDBApiKey)
		} else {
			err = errors.New("404")
		}

		// Authorization token is missing or invalid
		if err != nil && err.Error() == "401" {
			return "", errors.New("SteamGridDB authorization token is missing or invalid")
			// Could not find game with that id
		} else if err != nil && err.Error() == "404" {
			// Try searching for the name…
			url = steamGridDBBaseURL + "/search/autocomplete/" + game.Name + filter
			responseBytes, err = steamGridDBGetRequest(url, steamGridDBApiKey)
			if err != nil && err.Error() == "401" {
				return "", errors.New("SteamGridDB authorization token is missing or invalid")
			} else if err != nil {
				return "", err
			}

			var jsonSearchResponse steamGridDBSearchResponse
			err = json.Unmarshal(responseBytes, &jsonSearchResponse)
			if err != nil {
				return "", errors.New("Best search match doesn't has a requested type or style")
			}

			SteamGridDBGameID := -1
			if jsonSearchResponse.Success && len(jsonSearchResponse.Data) >= 1 {
				fuzzy.Sort(jsonSearchResponse, game.Name)
				SteamGridDBGameID = jsonSearchResponse.Data[0].ID
			}

			if SteamGridDBGameID == -1 {
				return "", nil
			}

			// …and get the url of the top result.
			url = baseURL + "/game/" + strconv.Itoa(SteamGridDBGameID) + filter
			responseBytes, err = steamGridDBGetRequest(url, steamGridDBApiKey)
			if err != nil {
				return "", err
			}
		} else if err != nil {
			return "", err
		}

		err = json.Unmarshal(responseBytes, &jsonResponse)
		if err != nil {
			return "", err
		}

		if jsonResponse.Success && len(jsonResponse.Data) >= 1 {
			return jsonResponse.Data[0].URL, nil
		}
	}

	return "", nil
}

const igdbImageURL = "https://images.igdb.com/igdb/image/upload/t_720p/%v.jpg"
const igdbGameURL = "https://api-v3.igdb.com/games"
const igdbCoverURL = "https://api-v3.igdb.com/covers"
const igdbGameBody = `fields name,cover; search "%v";`
const igdbCoverBody = `fields image_id; where id = %v;`

type igdbGame struct {
	ID    int
	Cover int
	Name  string
}

type igdbCover struct {
	ID       int
	Image_ID string
}

func igdbPostRequest(url string, body string, IGDBApiKey string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Add("user-key", IGDBApiKey)
	req.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	response.Body.Close()

	return responseBytes, nil
}

func getIGDBImage(gameName string, IGDBApiKey string) (string, error) {
	responseBytes, err := igdbPostRequest(igdbGameURL, fmt.Sprintf(igdbGameBody, gameName), IGDBApiKey)
	if err != nil {
		return "", err
	}

	var jsonGameResponse []igdbGame
	err = json.Unmarshal(responseBytes, &jsonGameResponse)
	if err != nil {
		return "", nil
	}

	if len(jsonGameResponse) < 1 || jsonGameResponse[0].Cover == 0 {
		return "", nil
	}

	responseBytes, err = igdbPostRequest(igdbCoverURL, fmt.Sprintf(igdbCoverBody, jsonGameResponse[0].Cover), IGDBApiKey)
	if err != nil {
		return "", err
	}

	var jsonCoverResponse []igdbCover
	err = json.Unmarshal(responseBytes, &jsonCoverResponse)
	if err != nil {
		return "", nil
	}

	if len(jsonCoverResponse) >= 1 {
		return fmt.Sprintf(igdbImageURL, jsonCoverResponse[0].Image_ID), nil
	}

	return "", nil
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
	} else if response.StatusCode >= 400 {
		// Other errors should be reported, though.
		return nil, errors.New("Failed to download image " + url + ": " + response.Status)
	}

	return response, nil
}

// Primary URL for downloading grid images.
const akamaiURLFormat = `https://steamcdn-a.akamaihd.net/steam/apps/%v/`

// The subreddit mentions this as primary, but I've found Akamai to contain
// more images and answer faster.
const steamCdnURLFormat = `cdn.akamai.steamstatic.com/steam/apps/%v/`

// Tries to load the grid image for a game from a number of alternative
// sources. Returns the final response received and a flag indicating if it was
// from a Google search (useful because we want to log the lower quality
// images).
func getImageAlternatives(game *Game, artStyle string, artStyleExtensions []string, skipSteam bool, steamGridDBApiKey string, steamGridFilter string, IGDBApiKey string, skipGoogle bool, onlyMissingArtwork bool) (response *http.Response, from string, err error) {
	from = "steam server"
	if !skipSteam {
		response, err = tryDownload(fmt.Sprintf(akamaiURLFormat+artStyleExtensions[2], game.ID))
		if err == nil && response != nil {
			if onlyMissingArtwork {
				// Abort if image is available
				return nil, "", nil
			}
			return
		}

		response, err = tryDownload(fmt.Sprintf(steamCdnURLFormat+artStyleExtensions[2], game.ID))
		if err == nil && response != nil {
			if onlyMissingArtwork {
				// Abort if image is available
				return nil, "", nil
			}
			return
		}
	}

	url := ""
	if steamGridDBApiKey != "" && url == "" {
		from = "SteamGridDB"
		url, err = getSteamGridDBImage(game, artStyleExtensions, steamGridDBApiKey, steamGridFilter)
		if err != nil {
			return
		}
	}

	// IGDB has mostly cover styles
	if artStyle == "Cover" && IGDBApiKey != "" && url == "" {
		from = "IGDB"
		url, err = getIGDBImage(game.Name, IGDBApiKey)
		if err != nil {
			return
		}
	}

	// Skip for Covers, bad results
	if !skipGoogle && artStyle == "Banner" && url == "" {
		from = "search"
		url, err = getGoogleImage(game.Name, artStyleExtensions)
		if err != nil {
			return
		}
	}

	response, err = tryDownload(url)
	if err == nil && response != nil {
		return
	}

	return nil, "", nil
}

// DownloadImage tries to download the game images, saving it in game.ImageBytes. Returns
// flags indicating if the operation succeeded and if the image downloaded was
// from a search.
func DownloadImage(gridDir string, game *Game, artStyle string, artStyleExtensions []string, skipSteam bool, steamGridDBApiKey string, steamGridFilter string, IGDBApiKey string, skipGoogle bool, onlyMissingArtwork bool) (string, error) {
	response, from, err := getImageAlternatives(game, artStyle, artStyleExtensions, skipSteam, steamGridDBApiKey, steamGridFilter, IGDBApiKey, skipGoogle, onlyMissingArtwork)
	if response == nil || err != nil {
		return "", err
	}

	contentType := response.Header.Get("Content-Type")
	urlExt := filepath.Ext(response.Request.URL.Path)
	if contentType != "" {
		game.ImageExt = "." + strings.Split(contentType, "/")[1]
	} else if urlExt != "" {
		game.ImageExt = urlExt
	} else {
		// Steam is forgiving on image extensions.
		game.ImageExt = "jpg"
	}

	if game.ImageExt == ".jpeg" {
		// The new library ignores .jpeg
		game.ImageExt = ".jpg"
	} else if game.ImageExt == ".octet-stream" {
		// Amazonaws (steamgriddb) gives us an .octet-stream
		game.ImageExt = ".png"
	}

	imageBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	// catch false aspect ratios
	image, _, err := image.Decode(bytes.NewBuffer(imageBytes))
	if err != nil {
		return "", err
	}
	imageSize := image.Bounds().Max
	if artStyle == "Banner" && imageSize.X < imageSize.Y {
		return "", nil
	} else if artStyle == "Cover" && imageSize.X > imageSize.Y {
		return "", nil
	}

	game.ImageSource = from

	game.CleanImageBytes = imageBytes
	return from, nil
}

// Get game name from SteamDB as last resort.
const steamDBFormat = `https://steamdb.info/app/%v`

func getGameName(gameID string) string {
	response, err := tryDownload(fmt.Sprintf(steamDBFormat, gameID))
	if err != nil || response == nil {
		return ""
	}
	page, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ""
	}
	response.Body.Close()

	pattern := regexp.MustCompile("<tr>\n<td>Name</td>\\s*<td itemprop=\"name\">(.*?)</td>")
	match := pattern.FindStringSubmatch(string(page))
	if match == nil || len(match) == 0 {
		return ""
	}

	return match[1]
}
