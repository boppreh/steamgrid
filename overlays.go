package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"runtime/debug"

	// "image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kmicki/apng"
	"github.com/kmicki/webpanimation"
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
func ApplyOverlay(game *Game, overlays map[string]image.Image, artStyleExtensions []string, convertWebpToApng bool, convertWebpToApngCoversBanners bool, maxMem uint64) error {
	if game.CleanImageBytes == nil || len(game.Tags) == 0 {
		return nil
	}

	buf := new(bytes.Buffer)
	bufReady := false
	var errBuff error
	errBuff = nil

	convertWebpToApng = convertWebpToApng || (convertWebpToApngCoversBanners &&
		(strings.Contains(artStyleExtensions[1], "cover")) || (strings.Contains(artStyleExtensions[1], "banner")))

	isApng := false
	isWebp := false
	formatFound := false

	var err error
	var webpImage *webpanimation.WebpAnimationDecoded
	defer func() {
		if webpImage != nil {
			webpanimation.ReleaseDecoder(webpImage)
		}
	}()

	// Try WEBP
	var gameImage image.Image
	webpImage, err = webpanimation.GetInfo(bytes.NewBuffer(game.CleanImageBytes))
	if err == nil {
		formatFound = true
		if err != nil {
			formatFound = false
		} else if webpImage.FrameCnt <= 1 {
			webpFrame, ok := webpanimation.GetNextFrame(webpImage)
			if ok {
				gameImage = webpFrame.Image
			} else {
				err = errors.New("can't get the first frame of single-frame WEBP image")
			}
		} else {
			isWebp = true
			memNeeded := uint64(webpImage.Width) * uint64(webpImage.Height) * 4 * uint64(webpImage.FrameCnt)
			if convertWebpToApng && maxMem > 0 {
				if memNeeded > maxMem {
					fmt.Println("WEBP animation too big to convert to APNG. Leaving WEBP.")
					convertWebpToApng = false
				} else if memNeeded > maxMem/2 {
					// free up memory for big conversion
					debug.FreeOSMemory()
				}
			}
		}
	}

	// Try APNG
	var apngImage apng.APNG
	if !formatFound {
		apngImage, err = apng.DecodeAll(bytes.NewBuffer(game.CleanImageBytes))
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
	}

	applied := false
	var webpanim *webpanimation.WebpAnimation
	defer func() {
		if webpanim != nil {
			webpanim.ReleaseMemory()
			//fmt.Println("WEBPAnim Memory Released")
		}
	}()
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
			fmt.Printf("Apply Overlay to APNG.")
			originalSize := apngImage.Frames[0].Image.Bounds().Max

			// Scale overlay to imageSize so the images won't get that huge…
			overlayScaled := image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
			if originalSize.X != overlaySize.X && originalSize.Y != overlaySize.Y {
				// https://godoc.org/golang.org/x/image/draw#Kernel.Scale
				draw.ApproxBiLinear.Scale(overlayScaled, overlayScaled.Bounds(), overlayImage, overlayImage.Bounds(), draw.Over, nil)
			} else {
				draw.Draw(overlayScaled, overlayScaled.Bounds(), overlayImage, image.Point{}, draw.Src)
			}

			for i, frame := range apngImage.Frames {
				result := image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
				// No idea why these offsets are negative:
				draw.Draw(result, result.Bounds(), frame.Image, image.Point{0 - frame.XOffset, 0 - frame.YOffset}, draw.Over)
				draw.Draw(result, result.Bounds(), overlayScaled, image.Point{0, 0}, draw.Over)
				apngImage.Frames[i].Image = result
				apngImage.Frames[i].XOffset = 0
				apngImage.Frames[i].YOffset = 0
				apngImage.Frames[i].BlendOp = apng.BLEND_OP_OVER
				fmt.Printf("\rApply Overlay to APNG. Overlayed frame %8d/%d", i, len(apngImage.Frames))
			}
			applied = true
			fmt.Printf("\rOverlay applied to %v frames of APNG                                              \n", len(apngImage.Frames))
		} else if isWebp {
			fmt.Printf("Apply Overlay to WEBP.")
			if webpImage == nil {
				fmt.Printf("\rWebPImage not initialized.\n")
				continue
			}
			originalSize := image.Point{webpImage.Width, webpImage.Height}
			var webpConfig webpanimation.WebPConfig
			var encoder *apng.FrameByFrameEncoder
			if convertWebpToApng {
				bufReady = true
				encoder = apng.InitializeEncoding(buf, uint32(webpImage.FrameCnt), uint(webpImage.LoopCount))
			} else {
				webpanim = webpanimation.NewWebpAnimation(webpImage.Width, webpImage.Height, webpImage.LoopCount)
				webpanim.WebPAnimEncoderOptions.SetKmin(9)
				webpanim.WebPAnimEncoderOptions.SetKmax(17)
				webpConfig = webpanimation.NewWebpConfig()
				webpConfig.SetLossless(1)
			}

			// Scale overlay to imageSize so the images won't get that huge…
			overlayScaled := image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
			var result *image.RGBA
			if originalSize.X != overlaySize.X && originalSize.Y != overlaySize.Y {
				// https://godoc.org/golang.org/x/image/draw#Kernel.Scale
				draw.ApproxBiLinear.Scale(overlayScaled, overlayScaled.Bounds(), overlayImage, overlayImage.Bounds(), draw.Over, nil)
			} else {
				draw.Draw(overlayScaled, overlayScaled.Bounds(), overlayImage, image.Point{}, draw.Src)
			}

			i := 0
			var lastTimestamp int
			frame, ok := webpanimation.GetNextFrame(webpImage)
			for ok {
				if v, o := frame.Image.(*image.RGBA); o {
					result = v
				} else {
					result = image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
					draw.Draw(result, result.Bounds(), frame.Image, image.Point{0, 0}, draw.Over)
				}
				draw.Draw(result, result.Bounds(), overlayScaled, image.Point{0, 0}, draw.Over)

				var delay uint16
				if i == 0 {
					delay = 0
				} else {
					delay = uint16(frame.Timestamp - lastTimestamp)
				}
				lastTimestamp = frame.Timestamp

				if convertWebpToApng {
					apngFrame := apng.Frame{
						Image:            result,
						IsDefault:        false,
						XOffset:          0,
						YOffset:          0,
						DisposeOp:        apng.DISPOSE_OP_NONE,
						BlendOp:          apng.BLEND_OP_OVER,
						DelayNumerator:   delay,
						DelayDenominator: 1000,
					}
					encoder.EncodeFrame(apngFrame)

					fmt.Printf("\rApply Overlay to WEBP as APNG. Overlayed frame %8d/%d", i, webpImage.FrameCnt)
				} else {
					err = webpanim.AddFrame(result, frame.Timestamp, webpConfig)
					fmt.Printf("\rApply Overlay to WEBP. Overlayed frame %8d/%d", i, webpImage.FrameCnt)
				}
				i++
				frame, ok = webpanimation.GetNextFrame(webpImage)
			}
			applied = true
			if convertWebpToApng {
				errBuff = encoder.Finish()
				fmt.Printf("\rOverlay applied to %v frames of WEBP as APNG                                                             \n", webpImage.FrameCnt)
			} else {
				fmt.Printf("\rOverlay applied to %v frames of WEBP                                                              \n", webpImage.FrameCnt)
			}
		} else {
			fmt.Printf("Apply Overlay to Single Image.")
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
			fmt.Printf("\rApplied Overlay to Single Image.\n")
		}
	}

	if !applied {
		if isWebp && convertWebpToApng {
			bufReady = true

			// Convert to APNG without overlay
			fmt.Printf("Convert WEBP to APNG.")
			if webpImage == nil {
				fmt.Printf("\rWebPImage not initialized.\n")
				return nil
			}
			originalSize := image.Point{webpImage.Width, webpImage.Height}
			encoder := apng.InitializeEncoding(buf, uint32(webpImage.FrameCnt), uint(webpImage.LoopCount))

			i := 0
			var lastTimestamp int
			frame, ok := webpanimation.GetNextFrame(webpImage)
			var result *image.RGBA
			for ok {
				if v, o := frame.Image.(*image.RGBA); o {
					result = v
				} else {
					result = image.NewRGBA(image.Rect(0, 0, originalSize.X, originalSize.Y))
					draw.Draw(result, result.Bounds(), frame.Image, image.Point{0, 0}, draw.Over)
				}

				var delay uint16
				if i == 0 {
					delay = 0
				} else {
					delay = uint16(frame.Timestamp - lastTimestamp)
				}
				lastTimestamp = frame.Timestamp

				apngFrame := apng.Frame{
					Image:            result,
					IsDefault:        false,
					XOffset:          0,
					YOffset:          0,
					DisposeOp:        apng.DISPOSE_OP_NONE,
					BlendOp:          apng.BLEND_OP_OVER,
					DelayNumerator:   delay,
					DelayDenominator: 1000,
				}
				encoder.EncodeFrame(apngFrame)

				fmt.Printf("\rConvert to WEBP as APNG. Frame %8d/%d", i, webpImage.FrameCnt)
				i++
				frame, ok = webpanimation.GetNextFrame(webpImage)
			}

			errBuff = encoder.Finish()
			applied = true
			fmt.Printf("\rConverted %v frames from WEBP to APNG                                                             \n", webpImage.FrameCnt)
		} else {
			return nil
		}
	}

	if bufReady {
		err = errBuff
	} else {
		if game.ImageExt == ".jpg" || game.ImageExt == ".jpeg" {
			err = jpeg.Encode(buf, gameImage, &jpeg.Options{Quality: 95})
		} else if (game.ImageExt == ".png" && isApng) || (isWebp && convertWebpToApng) {
			err = apng.Encode(buf, apngImage)
		} else if (game.ImageExt == ".png" && !isWebp) || (formatFound && !isWebp) {
			err = png.Encode(buf, gameImage)
		} else if isWebp {
			err = webpanim.Encode(buf)
		}
	}

	if err != nil {
		return err
	}
	game.OverlayImageBytes = buf.Bytes()
	return nil
}
