# Vidio

A simple Video I/O library written in Go. This library relies on [FFmpeg](https://www.ffmpeg.org/), and [FFProbe](https://www.ffmpeg.org/) which must be downloaded before usage.

## `Video`

The `Video` struct stores data about a video file you give it. The code below shows an example of sequentially reading the frames of the given video.

```go
video := NewVideo("input.mp4")
for video.NextFrame() {
	// "frame" stores the video frame as a flattened RGB image.
    frame := video.framebuffer // stored as: RGBRGBRGBRGB...
}
```

```go
type Video struct {
	filename    string
	width       int
	height      int
	depth       int
	bitrate     int
	frames      int
	duration    float64
	fps         float64
	codec       string
	pix_fmt     string
	framebuffer []byte
	pipe        *io.ReadCloser
	cmd         *exec.Cmd
}
```

## `Camera`

The `Camera` can read from any cameras on the device running Vidio.

```go
type Camera struct {
	name        string
	width       int
	height      int
	depth       int
	fps         float64
	codec       string
	framebuffer []byte
	pipe        *io.ReadCloser
	cmd         *exec.Cmd
}
```

```go
camera := NewCamera(0) // Get Webcam
defer camera.Close()

// Stream the webcam.
for camera.NextFrame() {
	// "frame" stores the video frame as a flattened RGB image.
	frame := camera.framebuffer // stored as: RGBRGBRGBRGB...
}
```

## `VideoWriter`

The `VideoWriter` is used to write frames to a video file. You first need to create a `Video` struct with all the desired properties of the new video you want to create such as width, height and framerate.

```go
type Options struct {
	width       int			// Width of Output Frames
	height      int			// Height of Output Frames
	bitrate     int			// Bitrate
	loop        int			// For GIFs only. -1=no loop, 0=loop forever, >0=loop n times
	delay       int			// Delay for Final Frame of GIFs
	macro       int			// macro size for determining how to resize frames for codecs
	fps         float64		// Frames per second
	codec       string		// Codec for video
	in_pix_fmt  string		// Pixel Format of incoming bytes
	out_pix_fmt string		// Pixel Format for video being written
}
```

```go
type VideoWriter struct {
	filename    string
	width       int
	height      int
	bitrate     int
	loop        int
	delay       int
	macro       int
	fps         float64
	codec       string
	in_pix_fmt  string
	out_pix_fmt string
	pipe        *io.WriteCloser
	cmd         *exec.Cmd
}
```

```go
w, h, c := 1920, 1080, 3
options = Options{width: w, height: w, bitrate: 100000}

writer := NewVideoWriter("output.mp4", &options)
defer writer.Close() // Make sure to close writer.

frame = make([]byte, w*h*c) // Create Frame as RGB Image and modify.
writer.Write(frame) // Write Frame to video.
```

## Examples

Copy `input` to `output`.

```go
video := NewVideo(input)
options := Options{
	width: video.width,
	height: video.height,
	fps: video.fps,
	bitrate: video.bitrate
}

writer := NewVideoWriter(output, &options)
defer writer.Close()

for video.NextFrame() {
    writer.Write(video.framebuffer)
}
```

# Acknowledgements

* Special thanks to [Zulko](http://zulko.github.io/) and his [blog post](http://zulko.github.io/blog/2013/09/27/read-and-write-video-frames-in-python-using-ffmpeg/) about using FFmpeg to process video.
* The [ImageIO-FFMPEG](https://github.com/imageio/imageio-ffmpeg/) project on GitHub.