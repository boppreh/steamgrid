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
type SteamGridDBResponse struct {
	Success bool
	Data []struct {
		Id int
		Score int
		Style string
		Url string
		Thumb string
		Tags []string
		Author struct {
			Name string
			Steam64 string
			Avatar string
		}
	}
}

type SteamGridDBSearchResponse struct {
	Success bool
	Data []struct {
		Id int
		Name string
		Types []string
		Verified bool
	}
}

// Search SteamGridDB for cover image
const SteamGridDBBaseURL = "https://www.steamgriddb.com/api/v2"

func SteamGridDBGetRequest(url string, steamGridDBApiKey string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer " + steamGridDBApiKey)
	if err != nil {
		return nil, err
	}

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

func getSteamGridDBImage(game *Game, artStyleExtensions []string, steamGridDBApiKey string) (string, error) {
	// Specify artType:
	// "alternate" "blurred" "white_logo" "material" "no_logo"
	artTypes := []string{"alternate"}
	filter := "?styles=" + strings.Join(artTypes, ",")

	// Try for HQ, then for LQ
	// It's possible to request both dimensions in one go but that'll give us scrambled results with no indicator which result has which size.
	for i := 0; i < 3; i += 2 {
		dimensions := filter + "&dimensions=" + artStyleExtensions[3 + i] + "x" + artStyleExtensions[4 + i]

		// Try with game.ID which is probably steams appID
		url := SteamGridDBBaseURL + "/grids/steam/" + game.ID + dimensions
		responseBytes, err := SteamGridDBGetRequest(url, steamGridDBApiKey)
		var jsonResponse SteamGridDBResponse

		// Authorization token is missing or invalid
	 	if err != nil && err.Error() == "401" {
			return "", errors.New("SteamGridDB authorization token is missing or invalid")
		// Could not find game with that id
		} else if err != nil && err.Error() == "404" {
			// Try searching for the name…
			url = SteamGridDBBaseURL + "/search/autocomplete/" + game.Name + dimensions
			responseBytes, err = SteamGridDBGetRequest(url, steamGridDBApiKey)
			if err != nil {
				return "", err
			}

			var jsonSearchResponse SteamGridDBSearchResponse
			err = json.Unmarshal(responseBytes, &jsonSearchResponse)
			if err != nil {
				return "", errors.New("Best search match doesn't has a " + strings.Join(artTypes, ",") + " type")
			}

			SteamGridDBGameId := -1
			if jsonSearchResponse.Success && len(jsonSearchResponse.Data) >= 1 {
				for _, n := range jsonSearchResponse.Data[0].Types {
					for _, m := range artTypes {
						if n == m {
							// This game has at least one of our requested artTypes
							SteamGridDBGameId = jsonSearchResponse.Data[0].Id
							break
						}
					}

					if SteamGridDBGameId != -1 {
						break
					}
				}
			}

			if SteamGridDBGameId == -1 {
				return "", nil
			}


			// …and get the url of the top result.
			url = SteamGridDBBaseURL + "/grids/game/" + strconv.Itoa(SteamGridDBGameId) + dimensions
			responseBytes, err = SteamGridDBGetRequest(url, steamGridDBApiKey)
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
			return jsonResponse.Data[0].Url, nil
		}
	}

	return "", nil
}

const IGDBImageURL = "https://images.igdb.com/igdb/image/upload/t_720p/%v.jpg"
const IGDBGameURL = "https://api-v3.igdb.com/games"
const IGDBCoverURL = "https://api-v3.igdb.com/covers"
const IGDBGameBody = `fields name,cover; search "%v";`
const IGDBCoverBody = `fields image_id; where id = %v;`

type IGDBGame struct {
	Id int
	Cover int
	Name string
}

type IGDBCover struct {
	Id int
	Image_id string
}

func IGDBPostRequest(url string, body string, IGDBApiKey string) ([]byte, error) {
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
	responseBytes, err := IGDBPostRequest(IGDBGameURL, fmt.Sprintf(IGDBGameBody, gameName), IGDBApiKey)
	if err != nil {
		return "", err
	}

	var jsonGameResponse []IGDBGame
	err = json.Unmarshal(responseBytes, &jsonGameResponse)
	if err != nil {
		return "", nil
	}

	if len(jsonGameResponse) < 1 || jsonGameResponse[0].Cover == 0 {
		return "", nil
	}

	responseBytes, err = IGDBPostRequest(IGDBCoverURL, fmt.Sprintf(IGDBCoverBody, jsonGameResponse[0].Cover), IGDBApiKey)
	if err != nil {
		return "", err
	}

	var jsonCoverResponse []IGDBCover
	err = json.Unmarshal(responseBytes, &jsonCoverResponse)
	if err != nil {
		return "", nil
	}

	if len(jsonCoverResponse) >= 1 {
		return fmt.Sprintf(IGDBImageURL, jsonCoverResponse[0].Image_id), nil
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
func getImageAlternatives(game *Game, artStyle string, artStyleExtensions []string, steamGridDBApiKey string, IGDBApiKey string) (response *http.Response, from string, err error) {
	from = "steam server"
	response, err = tryDownload(fmt.Sprintf(akamaiURLFormat + artStyleExtensions[2], game.ID))
	if err == nil && response != nil {
		return
	}

	response, err = tryDownload(fmt.Sprintf(steamCdnURLFormat + artStyleExtensions[2], game.ID))
	if err == nil && response != nil {
		return
	}

	url := ""
	if (artStyle == "Cover" || artStyle == "Banner") && steamGridDBApiKey != "" && url == "" {
		from = "SteamGridDB"
		url, err = getSteamGridDBImage(game, artStyleExtensions, steamGridDBApiKey)
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
	if artStyle == "Banner" && url == "" {
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
func DownloadImage(gridDir string, game *Game, artStyle string, artStyleExtensions []string, steamGridDBApiKey string, IGDBApiKey string) (string, error) {
	response, from, err := getImageAlternatives(game, artStyle, artStyleExtensions, steamGridDBApiKey, IGDBApiKey)
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
	if (artStyle == "Banner" && imageSize.X < imageSize.Y) {
		return "", nil
	} else if (artStyle == "Cover" && imageSize.X > imageSize.Y) {
		return "", nil
	}

	game.ImageSource = from;

	game.CleanImageBytes = imageBytes
	return from, nil
}

// Get game name from SteamDB as last resort.
const steamDBFormat = `https://steamdb.info/app/%v`

func GetGameName(gameId string) string {
	response, err := tryDownload(fmt.Sprintf(steamDBFormat, gameId))
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
	} else {
		return match[1]
	}
}
