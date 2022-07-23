package vidio

import (
	"fmt"
	"os"
	"testing"
)

func assertEquals(actual, expected interface{}) {
	if expected != actual {
		panic(fmt.Sprintf("Expected %v, got %v", expected, actual))
	}
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
	assertEquals(video.depth, 3)
	assertEquals(video.bitrate, 170549)
	assertEquals(video.frames, 101)
	assertEquals(video.duration, 3.366667)
	assertEquals(video.fps, float64(30))
	assertEquals(video.codec, "h264")
	assertEquals(video.audioCodec, "aac")
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
	// [203 222 134 203 222 134 203 222 134 203]
	assertEquals(video.framebuffer[0], uint8(203))
	assertEquals(video.framebuffer[1], uint8(222))
	assertEquals(video.framebuffer[2], uint8(134))
	assertEquals(video.framebuffer[3], uint8(203))
	assertEquals(video.framebuffer[4], uint8(222))
	assertEquals(video.framebuffer[5], uint8(134))
	assertEquals(video.framebuffer[6], uint8(203))
	assertEquals(video.framebuffer[7], uint8(222))
	assertEquals(video.framebuffer[8], uint8(134))
	assertEquals(video.framebuffer[9], uint8(203))

	fmt.Println("Video Frame Test Passed")
}

func TestVideoWriting(t *testing.T) {
	testWriting := func(input, output string, audio bool) {
		video, err := NewVideo(input)
		if err != nil {
			panic(err)
		}
		options := Options{
			FPS:     video.FPS(),
			Bitrate: video.Bitrate(),
			Codec:   video.Codec(),
		}
		if audio {
			options.Audio = input
		}

		writer, err := NewVideoWriter(output, video.width, video.height, &options)
		if err != nil {
			panic(err)
		}
		for video.Read() {
			err := writer.Write(video.FrameBuffer())
			if err != nil {
				panic(err)
			}
		}
		writer.Close()

		os.Remove(output)
	}

	testWriting("test/koala.mp4", "test/koala-out.mp4", true)
	fmt.Println("Video Writing (with Audio) Test Passed")
	testWriting("test/koala-noaudio.mp4", "test/koala-noaudio-out.mp4", false)
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
		for i := 0; i < len(frame); i += 3 {
			rgb := frame[i : i+3]
			r, g, b := int(rgb[0]), int(rgb[1]), int(rgb[2])
			gray := uint8((3*r + 4*g + b) / 8)
			frame[i] = gray
			frame[i+1] = gray
			frame[i+2] = gray
		}
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
	assertEquals(koalaVideo["width"], "480")
	assertEquals(koalaVideo["height"], "270")
	assertEquals(koalaVideo["duration"], "3.366667")
	assertEquals(koalaVideo["bit_rate"], "170549")
	assertEquals(koalaVideo["codec_name"], "h264")
	koalaAudio, err := ffprobe("test/koala.mp4", "a")
	if err != nil {
		panic(err)
	}
	assertEquals(koalaAudio["codec_name"], "aac")

	koalaVideo, err = ffprobe("test/koala-noaudio.mp4", "v")
	if err != nil {
		panic(err)
	}
	assertEquals(koalaVideo["width"], "480")
	assertEquals(koalaVideo["height"], "270")
	assertEquals(koalaVideo["duration"], "3.366667")
	assertEquals(koalaVideo["bit_rate"], "170549")
	assertEquals(koalaVideo["codec_name"], "h264")
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
		[]byte(`ffmpeg version N-45279-g6b86dd5... --enable-runtime-cpudetect
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
		),
	)

	assertEquals(data[0], "Integrated Camera")
	assertEquals(data[1], "screen-capture-recorder")

	fmt.Println("Device Parsing for Windows Test Passed")
}

func TestWebcamParsing(t *testing.T) {
	camera := &Camera{}
	err := getCameraData(
		`Input #0, dshow, from 'video=Integrated Camera':
  Duration: N/A, start: 1367309.442000, bitrate: N/A
  Stream #0:0: Video: mjpeg (Baseline) (MJPG / 0x47504A4D), yuvj422p(pc, bt470bg/unknown/unknown), 1280x720, 30 fps, 30 tbr, 10000k tbn
At least one output file must be specified`,
		camera,
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
	assertEquals(len(img), 200*133*3)
	// [255 221 189 255 221 189 255 222 186 255]
	assertEquals(img[0], uint8(255))
	assertEquals(img[1], uint8(221))
	assertEquals(img[2], uint8(189))
	assertEquals(img[3], uint8(255))
	assertEquals(img[4], uint8(221))
	assertEquals(img[5], uint8(189))
	assertEquals(img[6], uint8(255))
	assertEquals(img[7], uint8(222))
	assertEquals(img[8], uint8(186))
	assertEquals(img[9], uint8(255))

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
