package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestBuildDisplayImageEncodesJPEGWithinMaxSide(t *testing.T) {
	t.Parallel()

	source := image.NewRGBA(image.Rect(0, 0, 1600, 800))
	source.Set(0, 0, color.RGBA{R: 255, A: 255})
	var sourceBody bytes.Buffer
	if err := png.Encode(&sourceBody, source); err != nil {
		t.Fatalf("png.Encode returned error: %v", err)
	}

	displayBody, contentType, err := buildDisplayImage(sourceBody.Bytes())
	if err != nil {
		t.Fatalf("buildDisplayImage returned error: %v", err)
	}
	if contentType != displayImageContentType {
		t.Fatalf("expected %s, got %s", displayImageContentType, contentType)
	}

	display, err := jpeg.Decode(bytes.NewReader(displayBody))
	if err != nil {
		t.Fatalf("jpeg.Decode returned error: %v", err)
	}
	if display.Bounds().Dx() != displayImageMaxSide || display.Bounds().Dy() != 640 {
		t.Fatalf("unexpected display size: %dx%d", display.Bounds().Dx(), display.Bounds().Dy())
	}
}
