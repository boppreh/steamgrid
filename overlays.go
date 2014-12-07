package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

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
func ApplyOverlay(game *Game, overlays map[string]image.Image) (applied bool, err error) {
	if game.ImagePath == "" || game.ImageBytes == nil || len(game.Tags) == 0 {
		return false, nil
	}

	gameImage, _, err := image.Decode(bytes.NewBuffer(game.ImageBytes))
	if err != nil {
		return false, err
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
		applied = true
	}

	buf := new(bytes.Buffer)
	if strings.HasSuffix(game.ImagePath, "jpg") {
		err = jpeg.Encode(buf, gameImage, &jpeg.Options{90})
	} else if strings.HasSuffix(game.ImagePath, "png") {
		err = png.Encode(buf, gameImage)
	}
	if err != nil {
		return false, err
	}
	game.ImageBytes = buf.Bytes()
	err = ioutil.WriteFile(game.ImagePath, game.ImageBytes, 0666)
	return
}
