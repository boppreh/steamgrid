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

// LoadOverlays from the given dir, returning a map of name -> image.
func LoadOverlays(dir string) (overlays map[string]image.Image, err error) {
	overlays = make(map[string]image.Image, 0)

	if _, err = os.Stat(dir); err != nil {
		return overlays, nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	imageExtensions := []string{"png", "jpg", "jpeg", "gif"}

	for _, file := range files {
		isImage := false
		for _, extension := range imageExtensions {
			isImage = isImage || strings.HasSuffix(file.Name(), extension)
		}
		if !isImage {
			continue
		}

		reader, err := os.Open(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		img, _, err := image.Decode(reader)
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

// ApplyOverlay to the game image, depending on the category. The
// resulting image is saved over the original.
func ApplyOverlay(game *Game, overlays map[string]image.Image) error {
	if game.CleanImageBytes == nil || len(game.Tags) == 0 {
		return nil
	}

	gameImage, _, err := image.Decode(bytes.NewBuffer(game.CleanImageBytes))
	if err != nil {
		return err
	}

	applied := false
	for _, tag := range game.Tags {
		// Normalize tag name by lower-casing it and remove trailing "s" from
		// plurals. Also, <, > and / are replaced with - because you can't have
		// them in Windows paths.
		tagName := strings.TrimRight(strings.ToLower(tag), "s")
		tagName = strings.Replace(tagName, "<", "-", -1)
		tagName = strings.Replace(tagName, ">", "-", -1)
		tagName = strings.Replace(tagName, "/", "-", -1)

		overlayImage, ok := overlays[tagName]
		if !ok {
			continue
		}

		result := image.NewRGBA(image.Rect(0, 0, 460, 215))
		originalSize := gameImage.Bounds().Max
		if (originalSize.X != 460 || originalSize.Y != 215) && false {
			// TODO: downscale incoming images. There are some official images with >22mb (!)
			// "golang.org/x/image/draw"
			//draw.ApproxBilinear.Scale(result, result.Bounds(), gameImage, gameImage.Bounds(), draw.Over, nil)
		} else {
			draw.Draw(result, result.Bounds(), gameImage, image.ZP, draw.Src)
		}
		draw.Draw(result, result.Bounds(), overlayImage, image.Point{0, 0}, draw.Over)
		gameImage = result
		applied = true
	}

	if !applied {
		return nil
	}

	buf := new(bytes.Buffer)
	if game.ImageExt == ".jpg" || game.ImageExt == ".jpeg" {
		err = jpeg.Encode(buf, gameImage, &jpeg.Options{95})
	} else if game.ImageExt == ".png" {
		err = png.Encode(buf, gameImage)
	}
	if err != nil {
		return err
	}
	game.OverlayImageBytes = buf.Bytes()
	return nil
}
