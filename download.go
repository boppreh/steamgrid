package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// When all else fails, Google it. Uses the regular web interface. There are
// two image search APIs, but one is deprecated and doesn't support exact size
// matching, and the other requires an API key limited to 100 searches a day.
const googleSearchFormat = `https://www.google.com.br/search?tbs=isz%3Aex%2Ciszw%3A460%2Ciszh%3A215&tbm=isch&num=5&q=`

// Possible Google result formats
var googleSearchResultPatterns = []string{`imgurl=(.+?\.(jpeg|jpg|png))&amp;imgrefurl=`, `\"ou\":\"(.+?)\",\"`}

// Returns the first steam grid image URL found by Google search of a given
// game name.
func getGoogleImage(gameName string) (string, error) {
	if gameName == "" {
		return "", nil
	}

	url := googleSearchFormat + url.QueryEscape(gameName)

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
const akamaiURLFormat = `https://steamcdn-a.akamaihd.net/steam/apps/%v/header.jpg`

// The subreddit mentions this as primary, but I've found Akamai to contain
// more images and answer faster.
const steamCdnURLFormat = `cdn.akamai.steamstatic.com/steam/apps/%v/header.jpg`

// Tries to load the grid image for a game from a number of alternative
// sources. Returns the final response received and a flag indicating if it was
// from a Google search (useful because we want to log the lower quality
// images).
func getImageAlternatives(game *Game) (response *http.Response, fromSearch bool, err error) {
	response, err = tryDownload(fmt.Sprintf(akamaiURLFormat, game.ID))
	if err == nil && response != nil {
		return
	}

	response, err = tryDownload(fmt.Sprintf(steamCdnURLFormat, game.ID))
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

// DownloadImage tries to download the game images, saving it in game.ImageBytes. Returns
// flags indicating if the operation succeeded and if the image downloaded was
// from a search.
func DownloadImage(gridDir string, game *Game) (bool, error) {
	response, fromSearch, err := getImageAlternatives(game)
	if response == nil || err != nil {
		return false, err
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

	imageBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if fromSearch {
		game.ImageSource = "search"
	} else {
		game.ImageSource = "download"
	}

	game.CleanImageBytes = imageBytes
	return fromSearch, nil
}

// Get game name from SteamDB as last resort.
const steamDBForamt = `https://steamdb.info/app/%v`

func GetGameName(gameId string) string {
	response, err := tryDownload(fmt.Sprintf(steamDBForamt, gameId))
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
