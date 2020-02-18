package main

import (
	"bytes"
	"image"

	// "image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kettek/apng"
	"golang.org/x/image/draw"
)

// LoadOverlays from the given dir, returning a map of name -> image.
func LoadOverlays(dir string, artStyles map[string][]string) (overlays map[string]image.Image, err error) {
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
		for _, artStyleExtensions := range artStyles {
			if strings.HasSuffix(name, artStyleExtensions[1]) {
				name = strings.TrimSuffix(name, artStyleExtensions[1])
				name = strings.TrimRight(strings.ToLower(name), "s")
				name = name + artStyleExtensions[1]
			}
		}

		overlays[name] = img
	}

	return
}

// ApplyOverlay to the game image, depending on the category. The
// resulting image is saved over the original.
func ApplyOverlay(game *Game, overlays map[string]image.Image, artStyleExtensions []string) error {
	if game.CleanImageBytes == nil || len(game.Tags) == 0 {
		return nil
	}

	isApng := false
	var gameImage image.Image
	apngImage, err := apng.DecodeAll(bytes.NewBuffer(game.CleanImageBytes))
	if err == nil {
		if len(apngImage.Frames) > 1 {
			isApng = true
		} else {
			gameImage = apngImage.Frames[0].Image
		}
	} else {
		gameImage, _, err = image.Decode(bytes.NewBuffer(game.CleanImageBytes))
		if err != nil {
			return err
		}
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

		overlayImage, ok := overlays[tagName+artStyleExtensions[1]]
		if !ok {
			continue
		}

		overlaySize := overlayImage.Bounds().Max

		if isApng {
			originalSize := apngImage.Frames[0].Image.Bounds().Max

			for i, frame := range apngImage.Frames {
				// Scale overlay to imageSize so the images won't get that huge…
				overlayScaled := image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
				result := image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
				if originalSize.X != overlaySize.X && originalSize.Y != overlaySize.Y {
					// https://godoc.org/golang.org/x/image/draw#Kernel.Scale
					draw.ApproxBiLinear.Scale(overlayScaled, overlayScaled.Bounds(), overlayImage, overlayImage.Bounds(), draw.Over, nil)
				} else {
					draw.Draw(overlayScaled, overlayScaled.Bounds(), overlayImage, image.ZP, draw.Src)
				}
				// No idea why these offsets are negative:
				draw.Draw(result, result.Bounds(), frame.Image, image.Point{0 - frame.XOffset, 0 - frame.YOffset}, draw.Over)
				draw.Draw(result, result.Bounds(), overlayScaled, image.Point{0, 0}, draw.Over)
				apngImage.Frames[i].Image = result
				apngImage.Frames[i].XOffset = 0
				apngImage.Frames[i].YOffset = 0
				apngImage.Frames[i].BlendOp = apng.BLEND_OP_OVER
			}
			applied = true
		} else {
			originalSize := gameImage.Bounds().Max

			// We expect overlays in the correct format so we have to scale the image if it doesn't fit
			result := image.NewRGBA(image.Rect(0, 0, overlaySize.X, overlaySize.Y))
			if originalSize.X != overlaySize.X && originalSize.Y != overlaySize.Y {
				// scale to fit overlay
				// https://godoc.org/golang.org/x/image/draw#Kernel.Scale
				draw.ApproxBiLinear.Scale(result, result.Bounds(), gameImage, gameImage.Bounds(), draw.Over, nil)
			} else {
				draw.Draw(result, result.Bounds(), gameImage, image.ZP, draw.Src)
			}
			draw.Draw(result, result.Bounds(), overlayImage, image.Point{0, 0}, draw.Over)
			gameImage = result
			applied = true
		}
	}

	if !applied {
		return nil
	}

	buf := new(bytes.Buffer)
	if game.ImageExt == ".jpg" || game.ImageExt == ".jpeg" {
		err = jpeg.Encode(buf, gameImage, &jpeg.Options{95})
	} else if game.ImageExt == ".png" && isApng {
		err = apng.Encode(buf, apngImage)
	} else if game.ImageExt == ".png" {
		err = png.Encode(buf, gameImage)
	}
	if err != nil {
		return err
	}
	game.OverlayImageBytes = buf.Bytes()
	return nil
}
