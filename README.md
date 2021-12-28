# Vidio

A simple Video I/O library written in Go. This library relies on [FFmpeg](https://www.ffmpeg.org/), and [FFProbe](https://www.ffmpeg.org/) which must be downloaded before usage.

All frames are encoded and decoded in 8-bit RGB format.

## Installation

```
go get github.com/AlexEidt/Vidio
```

## `Video`

The `Video` struct stores data about a video file you give it. The code below shows an example of sequentially reading the frames of the given video.

```go
type Video struct {
	filename    string          // Video Filename
	width       int             // Width of Frames
	height      int             // Height of Frames
	depth       int             // Depth of Frames
	bitrate     int             // Bitrate for video encoding
	frames      int             // Total number of frames
	duration    float64         // Duration in seconds
	fps         float64         // Frames per second
	codec       string          // Codec used to encode video
	pix_fmt     string          // Pixel format video is stored in
	framebuffer []byte          // Raw frame data
	pipe        *io.ReadCloser  // Stdout pipe for ffmpeg process
	cmd         *exec.Cmd       // ffmpeg command
}
```

```go
video := vidio.NewVideo("input.mp4")
for video.Read() {
	// "frame" stores the video frame as a flattened RGB image
	frame := video.framebuffer // stored as: RGBRGBRGBRGB...
	// Video processing here...
}
```

## `Camera`

The `Camera` can read from any cameras on the device running Vidio. It takes in the stream index. On most machines the webcam device has index 0.

```go
type Camera struct {
	name        string          // Camera device name
	width       int             // Camera frame width
	height      int             // Camera frame height
	depth       int             // Camera frame depth
	fps         float64         // Camera frames per second
	codec       string          // Camera codec
	framebuffer []byte          // Raw frame data
	pipe        *io.ReadCloser  // Stdout pipe for ffmpeg process streaming webcam
	cmd         *exec.Cmd       // ffmpeg command
}
```

```go
camera := vidio.NewCamera(0) // Get Webcam
defer camera.Close()

// Stream the webcam
for camera.Read() {
	// "frame" stores the video frame as a flattened RGB image
	frame := camera.framebuffer // stored as: RGBRGBRGBRGB...
	// Video processing here...
}
```

## `VideoWriter`

The `VideoWriter` is used to write frames to a video file. The only required parameters are the output file name, the width and height of the frames being written, and an `Options` struct. This contains all the desired properties of the new video you want to create.

```go
type Options struct {
	bitrate     int             // Bitrate
	loop        int             // For GIFs only. -1=no loop, 0=loop forever, >0=loop n times
	delay       int             // Delay for Final Frame of GIFs. Default -1 (Use same delay as previous frame)
	macro       int             // macro size for determining how to resize frames for codecs. Default 16
	fps         float64         // Frames per second. Default 25
	quality     float64         // If bitrate not given, use quality instead. Must be between 0 and 1. 0:best, 1:worst
	codec       string          // Codec for video. Default libx264
}
```

```go
type VideoWriter struct {
	filename    string          // Output video filename
	width       int             // Frame width
	height      int             // Frame height
	bitrate     int             // Output video bitrate for encoding
	loop        int             // Number of times for GIF to loop
	delay       int             // Delay of final frame of GIF
	macro       int             // macro size for determining how to resize frames for codecs
	fps         float64         // Frames per second for output video
	quality     float64         // Used if bitrate not given
	codec       string          // Codec to encode video with
	pipe        *io.WriteCloser // Stdout pipe of ffmpeg process
	cmd         *exec.Cmd       // ffmpeg command
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

Copy `input.mp4` to `output.mp4`.

```go
video := vidio.NewVideo("input.mp4")
options := vidio.Options{
	fps: video.fps,
	bitrate: video.bitrate
}

writer := vidio.NewVideoWriter("output.mp4", video.width, video.height, &options)
defer writer.Close()

for video.Read() {
    writer.Write(video.framebuffer)
}
```

Grayscale 1000 frames of webcam stream and store in `output.mp4`.

```go
webcam := vidio.NewCamera(0)
defer webcam.Close()

options := vidio.Options{fps: webcam.fps}

writer := vidio.NewVideoWriter("output.mp4", webcam.width, webcam.height, &options)
defer writer.Close()

count := 0
for webcam.Read() {
	for i := 0; i < len(webcam.framebuffer); i += 3 {
		rgb := webcam.framebuffer[i : i+3]
		r, g, b := int(rgb[0]), int(rgb[1]), int(rgb[2])
		gray := uint8((3*r + 4*g + b) / 8)
		writer.framebuffer[i] = gray
		writer.framebuffer[i+1] = gray
		writer.framebuffer[i+2] = gray
	}
	writer.Write(webcam.framebuffer)
	count++
	if count > 1000 {
		break
	}
}
```

Create a gif from a series of `png` files enumerated from 1 to 10 that loops continuously with a final frame delay of 1000 centiseconds.

```go
w, h, _ := vidio.Read("1.png") // Get frame dimensions from first image

options := vidio.Options{fps: 1, loop: -1, delay: 1000}

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