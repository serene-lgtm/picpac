package service

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
)

const displayImageContentType = "image/jpeg"
const displayImageMaxSide = 1280
const displayImageJPEGQuality = 82

func buildDisplayImage(body []byte) ([]byte, string, error) {
	source, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	display := resizeImageWithin(source, displayImageMaxSide)
	var output bytes.Buffer
	if err := jpeg.Encode(&output, display, &jpeg.Options{Quality: displayImageJPEGQuality}); err != nil {
		return nil, "", fmt.Errorf("encode display image: %w", err)
	}

	return output.Bytes(), displayImageContentType, nil
}

func resizeImageWithin(source image.Image, maxSide int) image.Image {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 || maxSide <= 0 {
		return source
	}

	targetWidth := width
	targetHeight := height
	if width > maxSide || height > maxSide {
		if width >= height {
			targetWidth = maxSide
			targetHeight = max(1, height*maxSide/width)
		} else {
			targetHeight = maxSide
			targetWidth = max(1, width*maxSide/height)
		}
	}

	target := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			sourceX := bounds.Min.X + x*width/targetWidth
			sourceY := bounds.Min.Y + y*height/targetHeight
			target.Set(x, y, flattenTransparentPixel(source.At(sourceX, sourceY)))
		}
	}

	return target
}

func flattenTransparentPixel(pixel color.Color) color.Color {
	nonPremultiplied := color.NRGBAModel.Convert(pixel).(color.NRGBA)
	if nonPremultiplied.A == 255 {
		return pixel
	}

	alpha := float64(nonPremultiplied.A) / 255
	red := uint8((float64(nonPremultiplied.R) * alpha) + (255 * (1 - alpha)))
	green := uint8((float64(nonPremultiplied.G) * alpha) + (255 * (1 - alpha)))
	blue := uint8((float64(nonPremultiplied.B) * alpha) + (255 * (1 - alpha)))

	return color.RGBA{R: red, G: green, B: blue, A: 255}
}
