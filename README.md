# Vidio

A simple Video I/O library written in Go. This library relies on [FFmpeg](https://www.ffmpeg.org/), and [FFProbe](https://www.ffmpeg.org/) which must be downloaded before usage and added to the system path.

All frames are encoded and decoded in 8-bit RGB format.

## Installation

```
go get github.com/AlexEidt/Vidio
```

## `Video`

The `Video` struct stores data about a video file you give it. The code below shows an example of sequentially reading the frames of the given video.

```go
FileName() string
Width() int
Height() int
Depth() int
Bitrate() int
Frames() int
Duration() float64
FPS() float64
Codec() string
AudioCodec() string
FrameBuffer() string
```

```go
video := vidio.NewVideo("input.mp4")
for video.Read() {
	// "frame" stores the video frame as a flattened RGB image in row-major order
	frame := video.FrameBuffer() // stored as: RGBRGBRGBRGB...
	// Video processing here...
}
```

## `Camera`

The `Camera` can read from any cameras on the device running Vidio. It takes in the stream index. For **Windows**, the index corresponds to the order in which the devices names appear when running `ffmpeg -list_devices true -f dshow -i dummy`. Alternative device names are included in this index. On most machines the webcam device has index 0. Note that audio retrieval from the microphone is not yet supported.

```go
Name() string
Width() int
Height() int
Depth() int
FPS() float64
Codec() string
FrameBuffer() string
```

```go
camera := vidio.NewCamera(0) // Get Webcam
defer camera.Close()

// Stream the webcam
for camera.Read() {
	// "frame" stores the video frame as a flattened RGB image
	frame := camera.FrameBuffer() // stored as: RGBRGBRGBRGB...
	// Video processing here...
}
```

## `VideoWriter`

The `VideoWriter` is used to write frames to a video file. The only required parameters are the output file name, the width and height of the frames being written, and an `Options` struct. This contains all the desired properties of the new video you want to create.

```go
FileName() string
Width() int
Height() int
Bitrate() int
Loop() int
Delay() int
Macro() int
FPS() float64
Quality() float64
Codec() string
AudioCodec() string
```

```go
type Options struct {
	Bitrate     int             // Bitrate
	Loop        int             // For GIFs only. -1=no loop, 0=loop forever, >0=loop n times
	Delay       int             // Delay for Final Frame of GIFs. Default -1 (Use same delay as previous frame)
	Macro       int             // macro size for determining how to resize frames for codecs. Default 16
	FPS         float64         // Frames per second. Default 25
	Quality     float64         // If bitrate not given, use quality instead. Must be between 0 and 1. 0:best, 1:worst
	Codec       string          // Codec for video. Default libx264
	Audio       string          // File path for audio for the video. If no audio, audio=nil.
	AudioCodec  string          // Codec for audio. Default aac
}
```

```go
w, h, c := 1920, 1080, 3
options := vidio.Options{} // Will fill in defaults if empty

writer := vidio.NewVideoWriter("output.mp4", w, h, &options)
defer writer.Close()

frame := make([]byte, w*h*c) // Create Frame as RGB Image and modify
writer.Write(frame) // Write Frame to video
```

## Images

Vidio provides some convenience functions for reading and writing to images using an array of bytes. Currently, only `png` and `jpeg` formats are supported.

```go
// Read png image
w, h, img := vidio.Read("input.png")

// w - width of image
// h - height of image
// img - byte array in RGB format. RGBRGBRGBRGB...

vidio.Write("output.jpg", w, h, img)
```

## Examples

Copy `input.mp4` to `output.mp4`. Copy the audio from `input.mp4` to `output.mp4` as well.

```go
video := vidio.NewVideo("input.mp4")
options := vidio.Options{
	FPS: video.FPS(),
	Bitrate: video.Bitrate(),
	Audio: "input.mp4",
}

writer := vidio.NewVideoWriter("output.mp4", video.Width(), video.Height(), &options)
defer writer.Close()

for video.Read() {
    writer.Write(video.FrameBuffer())
}
```

Grayscale 1000 frames of webcam stream and store in `output.mp4`.

```go
webcam := vidio.NewCamera(0)
defer webcam.Close()

options := vidio.Options{FPS: webcam.FPS()}

writer := vidio.NewVideoWriter("output.mp4", webcam.Width(), webcam.Height(), &options)
defer writer.Close()

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
	writer.Write(frame)
	count++
	if count > 1000 {
		break
	}
}
```

Create a gif from a series of `png` files enumerated from 1 to 10 that loops continuously with a final frame delay of 1000 centiseconds.

```go
w, h, _ := vidio.Read("1.png") // Get frame dimensions from first image

options := vidio.Options{FPS: 1, Loop: -1, Delay: 1000}

gif := vidio.NewVideoWriter("output.gif", w, h, &options)
defer gif.Close()

for i := 1; i <= 10; i++ {
	_, _, img := vidio.Read(strconv.Itoa(i)+".png")
	gif.Write(img)
}
```

# Acknowledgements

* Special thanks to [Zulko](http://zulko.github.io/) and his [blog post](http://zulko.github.io/blog/2013/09/27/read-and-write-video-frames-in-python-using-ffmpeg/) about using FFmpeg to process video.
* The [ImageIO-FFMPEG](https://github.com/imageio/imageio-ffmpeg/) project on GitHub.