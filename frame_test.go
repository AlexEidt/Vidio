package vidio

import (
	"image/png"
	"os"
	"testing"
)

func TestGetFrameShouldReturnErrorOnInvalidFilePath(t *testing.T) {
	path := "test/koala-video-not-present.mp4"

	frame, err := GetVideoFrame(path, 2)

	if frame != nil {
		t.Errorf("Frame was expected to be nil")
	}

	if err == nil {
		t.Error("Error was expected to not be nil")
	}
}

func TestGetFrameShouldReturnErrorOnOutOfRangeFrame(t *testing.T) {
	path := "test/koala.mp4"
	framesCount := 101

	frame, err := GetVideoFrame(path, framesCount+1)

	if frame != nil {
		t.Error("Frames was expected to be nil")
	}

	if err == nil {
		t.Error("Error was expected to not be nil")
	}
}

func TestGetFrameShouldReturnCorrectFrame(t *testing.T) {
	path := "test/koala.mp4"

	expectedFrameFile, _ := os.Open("test/koala-frame5.png")
	defer expectedFrameFile.Close()

	expectedFrame, _ := png.Decode(expectedFrameFile)

	actualFrame, err := GetVideoFrame(path, 5)

	if actualFrame == nil {
		t.Error("Frame was expected to not be nil")
	}

	if err != nil {
		t.Error("Error was expected to be nil")
	}

	for xIndex := 0; xIndex < expectedFrame.Bounds().Dx(); xIndex += 1 {
		for yIndex := 0; yIndex < expectedFrame.Bounds().Dy(); yIndex += 1 {
			eR, eG, eB, eA := expectedFrame.At(xIndex, yIndex).RGBA()
			aR, aG, aB, aA := actualFrame.At(xIndex, yIndex).RGBA()

			if eR != aR || eG != aG || eB != aB || eA != aA {
				t.Error("The expected and actual frames were expected to be equal")
			}
		}
	}
}
