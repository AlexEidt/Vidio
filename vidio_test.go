package vidio

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"
)

func assertEquals(actual, expected interface{}) {
	if expected != actual {
		panic(fmt.Sprintf("Expected %v, got %v", expected, actual))
	}
}

func TestSetBuffer(t *testing.T) {
	video, err := NewVideo("test/koala.mp4")
	if err != nil {
		panic(err)
	}
	defer video.Close()

	size := video.width*video.height*video.depth + 101
	video.SetFrameBuffer(make([]uint8, size))

	video.Read()

	assertEquals(len(video.framebuffer), size)

	fmt.Println("Set Buffer Test Passed")
}

func TestVideoMetaData(t *testing.T) {
	video, err := NewVideo("test/koala.mp4")
	if err != nil {
		panic(err)
	}
	defer video.Close()

	assertEquals(video.filename, "test/koala.mp4")
	assertEquals(video.width, 480)
	assertEquals(video.height, 270)
	assertEquals(video.depth, 4)
	assertEquals(video.bitrate, 170549)
	assertEquals(video.frames, 101)
	assertEquals(video.duration, 3.366667)
	assertEquals(video.fps, float64(30))
	assertEquals(video.codec, "h264")
	assertEquals(video.stream, 0)
	assertEquals(video.hasstreams, true)
	assertEquals(len(video.framebuffer), 0)

	if video.pipe != nil {
		panic("Expected video.pipe to be nil")
	}
	if video.cmd != nil {
		panic("Expected video.cmd to be nil")
	}

	fmt.Println("Video Meta Data Test Passed")
}

func TestVideoFrame(t *testing.T) {
	video, err := NewVideo("test/koala.mp4")
	if err != nil {
		panic(err)
	}
	defer video.Close()

	video.Read()
	// [203 222 134 255 203 222 134 255 203 222 134 255 203]
	assertEquals(video.framebuffer[0], uint8(203))
	assertEquals(video.framebuffer[1], uint8(222))
	assertEquals(video.framebuffer[2], uint8(134))
	assertEquals(video.framebuffer[3], uint8(255))

	assertEquals(video.framebuffer[4], uint8(203))
	assertEquals(video.framebuffer[5], uint8(222))
	assertEquals(video.framebuffer[6], uint8(134))
	assertEquals(video.framebuffer[7], uint8(255))

	assertEquals(video.framebuffer[8], uint8(203))
	assertEquals(video.framebuffer[9], uint8(222))
	assertEquals(video.framebuffer[10], uint8(134))
	assertEquals(video.framebuffer[11], uint8(255))

	assertEquals(video.framebuffer[12], uint8(203))

	fmt.Println("Video Frame Test Passed")
}

func TestVideoWriting(t *testing.T) {
	testWriting := func(input, output string) {
		video, err := NewVideo(input)
		if err != nil {
			panic(err)
		}
		options := Options{
			FPS:     video.FPS(),
			Codec:   video.Codec(),
			Bitrate: video.Bitrate(),
		}
		if video.HasStreams() {
			options.StreamFile = video.FileName()
		}

		writer, err := NewVideoWriter(output, video.width, video.height, &options)
		if err != nil {
			panic(err)
		}

		for video.Read() {
			writer.Write(video.FrameBuffer())
		}

		writer.Close()

		os.Remove(output)
	}

	testWriting("test/koala.mp4", "test/koala-out.mp4")
	fmt.Println("Video Writing (with Audio) Test Passed")
	testWriting("test/koala-noaudio.mp4", "test/koala-noaudio-out.mp4")
	fmt.Println("Video Writing (without Audio) Test Passed")
}

func TestCameraIO(t *testing.T) {
	webcam, err := NewCamera(0)
	if err != nil {
		panic(err)
	}

	options := Options{FPS: webcam.FPS()}

	writer, err := NewVideoWriter("test/camera.mp4", webcam.width, webcam.height, &options)
	if err != nil {
		panic(err)
	}

	count := 0
	for webcam.Read() {
		frame := webcam.FrameBuffer()
		err := writer.Write(frame)
		if err != nil {
			panic(err)
		}
		count++
		if count > 100 {
			break
		}
	}

	webcam.Close()
	writer.Close()

	os.Remove("test/camera.mp4")
	fmt.Println("Camera IO Test Passed")
}

func TestFFprobe(t *testing.T) {
	koalaVideo, err := ffprobe("test/koala.mp4", "v")
	if err != nil {
		panic(err)
	}
	assertEquals(koalaVideo[0]["width"], "480")
	assertEquals(koalaVideo[0]["height"], "270")
	assertEquals(koalaVideo[0]["duration"], "3.366667")
	assertEquals(koalaVideo[0]["bit_rate"], "170549")
	assertEquals(koalaVideo[0]["codec_name"], "h264")
	koalaAudio, err := ffprobe("test/koala.mp4", "a")
	if err != nil {
		panic(err)
	}
	assertEquals(koalaAudio[0]["codec_name"], "aac")

	koalaVideo, err = ffprobe("test/koala-noaudio.mp4", "v")
	if err != nil {
		panic(err)
	}
	assertEquals(koalaVideo[0]["width"], "480")
	assertEquals(koalaVideo[0]["height"], "270")
	assertEquals(koalaVideo[0]["duration"], "3.366667")
	assertEquals(koalaVideo[0]["bit_rate"], "170549")
	assertEquals(koalaVideo[0]["codec_name"], "h264")
	koalaAudio, err = ffprobe("test/koala-noaudio.mp4", "a")
	if err != nil {
		panic(err)
	}
	assertEquals(len(koalaAudio), 0)

	fmt.Println("FFprobe Test Passed")
}

// Linux and MacOS allow the user to directly choose a camera stream by index.
// Windows requires the user to give the device name.
func TestDeviceParsingWindows(t *testing.T) {
	// Sample string taken from FFmpeg wiki:
	data := parseDevices(
		`ffmpeg version N-45279-g6b86dd5... --enable-runtime-cpudetect
  libavutil      51. 74.100 / 51. 74.100
  libavcodec     54. 65.100 / 54. 65.100
  libavformat    54. 31.100 / 54. 31.100
  libavdevice    54.  3.100 / 54.  3.100
  libavfilter     3. 19.102 /  3. 19.102
  libswscale      2.  1.101 /  2.  1.101
  libswresample   0. 16.100 /  0. 16.100
[dshow @ 03ACF580] DirectShow video devices
[dshow @ 03ACF580]  "Integrated Camera"
[dshow @ 03ACF580]  "screen-capture-recorder"
[dshow @ 03ACF580] DirectShow audio devices
[dshow @ 03ACF580]  "Internal Microphone (Conexant 2"
[dshow @ 03ACF580]  "virtual-audio-capturer"
dummy: Immediate exit requested`,
	)

	assertEquals(data[0], "Integrated Camera")
	assertEquals(data[1], "screen-capture-recorder")

	fmt.Println("Device Parsing for Windows Test Passed")
}

func TestWebcamParsing(t *testing.T) {
	camera := &Camera{}
	err := camera.getCameraData(
		`Input #0, dshow, from 'video=Integrated Camera':
  Duration: N/A, start: 1367309.442000, bitrate: N/A
  Stream #0:0: Video: mjpeg (Baseline) (MJPG / 0x47504A4D), yuvj422p(pc, bt470bg/unknown/unknown), 1280x720, 30 fps, 30 tbr, 10000k tbn
At least one output file must be specified`,
	)

	if err != nil {
		panic(err)
	}

	assertEquals(camera.width, 1280)
	assertEquals(camera.height, 720)
	assertEquals(camera.fps, float64(30))
	assertEquals(camera.codec, "mjpeg")

	fmt.Println("Webcam Parsing Test Passed")
}

func TestImageRead(t *testing.T) {
	w, h, img, err := Read("test/bananas.jpg")
	if err != nil {
		panic(err)
	}

	assertEquals(w, 200)
	assertEquals(h, 133)
	assertEquals(len(img), 200*133*4)
	// [255 221 189 255 255 221 189 255 255 222 186 255 255]
	assertEquals(img[0], uint8(255))
	assertEquals(img[1], uint8(221))
	assertEquals(img[2], uint8(189))
	assertEquals(img[3], uint8(255))

	assertEquals(img[4], uint8(255))
	assertEquals(img[5], uint8(221))
	assertEquals(img[6], uint8(189))
	assertEquals(img[7], uint8(255))

	assertEquals(img[8], uint8(255))
	assertEquals(img[9], uint8(222))
	assertEquals(img[10], uint8(186))
	assertEquals(img[11], uint8(255))

	assertEquals(img[12], uint8(255))

	fmt.Println("Image Reading Test Passed")
}

func TestImageWrite(t *testing.T) {
	w, h, img, err := Read("test/bananas.jpg")
	if err != nil {
		panic(err)
	}
	err = Write("test/bananas-out.png", w, h, img)
	if err != nil {
		panic(err)
	}

	os.Remove("test/bananas-out.png")

	fmt.Println("Image Writing Test Passed")
}

func TestReadFrameShouldReturnErrorOnOutOfRangeFrame(t *testing.T) {
	path := "test/koala.mp4"

	video, err := NewVideo(path)
	if err != nil {
		t.Errorf("Failed to create the video: %s", err)
	}

	err = video.ReadFrame(video.Frames() + 1)
	if err == nil {
		t.Error("Error was expected to no be nil")
	}
}

func TestReadFrameShouldReturnCorrectFrame(t *testing.T) {
	path := "test/koala.mp4"

	expectedFrameFile, err := os.Open("test/koala-frame5.png")
	if err != nil {
		t.Errorf("Failed to arrange the test: %s", err)
	}

	defer expectedFrameFile.Close()

	expectedFrame, err := png.Decode(expectedFrameFile)
	if err != nil {
		t.Errorf("Failed to arrange the test: %s", err)
	}

	video, err := NewVideo(path)
	if err != nil {
		t.Errorf("Failed to create the video: %s", err)
	}

	actualFrame := image.NewRGBA(expectedFrame.Bounds())
	if err := video.SetFrameBuffer(actualFrame.Pix); err != nil {
		t.Errorf("Failed to set the frame buffer: %s", err)
	}

	if err := video.ReadFrame(5); err != nil {
		t.Errorf("Failed to read the given frame: %s", err)
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

func TestReadFramesShouldReturnErrorOnNoFramesSpecified(t *testing.T) {
	path := "test/koala.mp4"

	video, err := NewVideo(path)
	if err != nil {
		t.Errorf("Failed to create the video: %s", err)
	}

	_, err = video.ReadFrames()
	if err == nil {
		t.Error("Error was expected to no be nil")
	}
}

func TestReadFramesShouldReturnErrorOnOutOfRangeFrame(t *testing.T) {
	path := "test/koala.mp4"

	video, err := NewVideo(path)
	if err != nil {
		t.Errorf("Failed to create the video: %s", err)
	}

	_, err = video.ReadFrames(0, video.Frames()-1, video.Frames()+1)
	if err == nil {
		t.Error("Error was expected to no be nil")
	}
}

func TestReadFramesShouldReturnCorrectFrames(t *testing.T) {
	path := "test/koala.mp4"

	expectedFrames := make([]image.Image, 0, 2)

	expectedFrameFile, err := os.Open("test/koala-frame5.png")
	if err != nil {
		t.Errorf("Failed to arrange the test: %s", err)
	}

	expectedFrame, err := png.Decode(expectedFrameFile)
	if err != nil {
		t.Errorf("Failed to arrange the test: %s", err)
	}

	expectedFrameFile.Close()
	expectedFrames = append(expectedFrames, expectedFrame)

	expectedFrameFile, err = os.Open("test/koala-frame15.png")
	if err != nil {
		t.Errorf("Failed to arrange the test: %s", err)
	}

	expectedFrame, err = png.Decode(expectedFrameFile)
	if err != nil {
		t.Errorf("Failed to arrange the test: %s", err)
	}

	expectedFrameFile.Close()
	expectedFrames = append(expectedFrames, expectedFrame)

	video, err := NewVideo(path)
	if err != nil {
		t.Errorf("Failed to create the video: %s", err)
	}

	frames, err := video.ReadFrames(5, 15)
	if err != nil {
		t.Errorf("Failed to read frames: %s", err)
	}

	for index, actualFrame := range frames {
		expectedFrame := expectedFrames[index]
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
}
