# Video-IO

A simple Video I/O library written in Go. This library relies on [FFmpeg](https://www.ffmpeg.org/), which must be downloaded before usage.

## `Video`

The `Video` struct stores data about a video file you give it. The code below shows an example of sequentially reading the frames of the given video.

```go
video := NewVideo("input.mp4")
for video.NextFrame() {
    frame := video.framebuffer // "frame" stores the video frame as a flattened RGB image.
}
```

```
type Video struct {
	filename    string
	width       int
	height      int
	channels    int
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

Once the frame is read by calling the `NextFrame()` function, the resulting frame is stored in the `framebuffer` as shown above. The frame buffer is an array of bytes representing the most recently read frame as an RGB image. The framebuffer is flattened and contains image data in the form: `RGBRGBRGBRGB...`.

## `VideoWriter`

The `VideoWriter` is used to write frames to a video file. You first need to create a `Video` struct with all the desired properties of the new video you want to create such as width, height and framerate.

```go
video := Video{
    // width and height are required, defaults available for all other parameters.
    width:  1920,
    height: 1080,
    ... // Initialize other desired properties of the video you want to create.
}
writer := NewVideoWriter("output.mp4", video)
defer writer.Close() // Make sure to close writer.

w, h, c := 1920, 1080, 3
frame = make([]byte, w*h*c) // Create Frame as RGB Image and modify.
writer.Write(frame) // Write Frame to video.
...
```

Alternatively, you could manually create a `VideoWriter` struct and fill it in yourself.

```go
writer := VideoWriter{
    filename:   "output.mp4",
    width:      1920,
    height:     1080
    ...
}
defer writer.Close()

w, h, c := 1920, 1080, 3
frame = make([]byte, w*h*c) // Create Frame as RGB Image and modify.
writer.Write(frame) // Write Frame to video.
...
```

## Examples

Copy `input` to `output`.

```go
video := NewVideo(input)

writer := NewVideoWriter(output, video)
defer writer.Close()

for video.NextFrame() {
    writer.Write(video.framebuffer)
}
```

# Acknowledgements

* Special thanks to [Zulko](http://zulko.github.io/) and his [blog post](http://zulko.github.io/blog/2013/09/27/read-and-write-video-frames-in-python-using-ffmpeg/) about using FFmpeg to process video.
* The [ImageIO-FFMPEG](https://github.com/imageio/imageio-ffmpeg/) project on GitHub.