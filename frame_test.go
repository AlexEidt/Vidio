package vidio

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func TestGetFrameShouldReturnErrorOnInvalidFilePath(t *testing.T) {
	path := "test/koala-video-not-present.mp4"
	img := image.NewRGBA(image.Rect(0, 0, 480, 270))

	err := GetVideoFrame(path, 2, img.Pix)

	if err == nil {
		t.Error("Error was expected to not be nil")
	}
}

func TestGetFrameShouldReturnErrorOnOutOfRangeFrame(t *testing.T) {
	path := "test/koala.mp4"
	img := image.NewRGBA(image.Rect(0, 0, 480, 270))
	framesCount := 101

	err := GetVideoFrame(path, framesCount+1, img.Pix)

	if err == nil {
		t.Error("Error was expected to not be nil")
	}
}

func TestGetFrameShouldReturnErrorOnNilFrameBuffer(t *testing.T) {
	path := "test/koala.mp4"
	var buffer []byte = nil

	err := GetVideoFrame(path, 0, buffer)

	if err == nil {
		t.Error("Error was expected to not be nil")
	}
}

func TestGetFrameShouldReturnErrorOnInvalidFrameBufferSize(t *testing.T) {
	path := "test/koala.mp4"
	img := image.NewRGBA(image.Rect(0, 0, 480/2, 270/2))

	err := GetVideoFrame(path, 0, img.Pix)

	if err == nil {
		t.Error("Error was expected to not be nil")
	}
}

func TestGetFrameShouldReturnCorrectFrame(t *testing.T) {
	path := "test/koala.mp4"
	img := image.NewRGBA(image.Rect(0, 0, 480, 270))

	expectedFrameFile, _ := os.Open("test/koala-frame5.png")
	defer expectedFrameFile.Close()

	expectedFrame, _ := png.Decode(expectedFrameFile)

	err := GetVideoFrame(path, 5, img.Pix)

	if err != nil {
		t.Error("Error was expected to be nil")
	}

	for xIndex := 0; xIndex < expectedFrame.Bounds().Dx(); xIndex += 1 {
		for yIndex := 0; yIndex < expectedFrame.Bounds().Dy(); yIndex += 1 {
			eR, eG, eB, eA := expectedFrame.At(xIndex, yIndex).RGBA()
			aR, aG, aB, aA := img.At(xIndex, yIndex).RGBA()

			if eR != aR || eG != aG || eB != aB || eA != aA {
				t.Error("The expected and actual frames were expected to be equal")
			}
		}
	}
}
