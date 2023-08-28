# Vidio

A simple Video I/O library written in Go. This library relies on [FFmpeg](https://www.ffmpeg.org/), and [FFProbe](https://www.ffmpeg.org/) which must be downloaded before usage and added to the system path.

All frames are encoded and decoded in 8-bit RGBA format.

For Audio I/O using FFmpeg, see the [`aio`](https://github.com/AlexEidt/aio) project.

## Installation

```
go get github.com/AlexEidt/Vidio
```

## `Video`

The `Video` struct stores data about a video file you give it. The code below shows an example of sequentially reading the frames of the given video.

Calling the `Read()` function will fill in the `Video` struct `framebuffer` with the next frame data as 8-bit RGBA data, stored in a flattened byte array in row-major order where each pixel is represented by four consecutive bytes representing the R, G, B and A components of that pixel. Note that the A (alpha) component will always be 255. When iteration over the entire video file is not required, we can lookup a specific frame by calling `ReadFrame(n int)`. By calling `ReadFrames(n ...int)`, we can immediately access multiple frames as `[][]byte` and skip the `framebuffer`.

```go
vidio.NewVideo(filename string) (*vidio.Video, error)
vidio.NewVideoStreams(filename string) ([]*vidio.Video, error)

FileName() string
Width() int
Height() int
Depth() int
Bitrate() int
Frames() int
Stream() int
Duration() float64
FPS() float64
Codec() string
HasStreams() bool
FrameBuffer() []byte
MetaData() map[string]string
SetFrameBuffer(buffer []byte) error

Read() bool
ReadFrame(n int) error
ReadFrames(n ...int) ([][]byte, error)
Close()
```

If all frames have been read, `video` will be closed automatically. If not all frames are read, call `video.Close()` to close the video.

## `Camera`

The `Camera` can read from any cameras on the device running `Vidio`. It takes in the stream index. On most machines the webcam device has index 0.

```go
vidio.NewCamera(stream int) (*vidio.Camera, error)

Name() string
Width() int
Height() int
Depth() int
FPS() float64
Codec() string
FrameBuffer() []byte
SetFrameBuffer(buffer []byte) error

Read() bool
Close()
```

## `VideoWriter`

The `VideoWriter` is used to write frames to a video file. The only required parameters are the output file name, the width and height of the frames being written, and an `Options` struct. This contains all the desired properties of the new video you want to create.

```go
vidio.NewVideoWriter(filename string, width, height int, options *vidio.Options) (*vidio.VideoWriter, error)

FileName() string
StreamFile() string
Width() int
Height() int
Bitrate() int
Loop() int
Delay() int
Macro() int
FPS() float64
Quality() float64
Codec() string

Write(frame []byte) error
Close()
```

```go
type Options struct {
	Bitrate    int     // Bitrate.
	Loop       int     // For GIFs only. -1=no loop, 0=infinite loop, >0=number of loops.
	Delay      int     // Delay for final frame of GIFs in centiseconds.
	Macro      int     // Macroblock size for determining how to resize frames for codecs.
	FPS        float64 // Frames per second for output video.
	Quality    float64 // If bitrate not given, use quality instead. Must be between 0 and 1. 0:best, 1:worst.
	Codec      string  // Codec for video.
	StreamFile string  // File path for extra stream data.
}
```

The `Options.StreamFile` parameter is intended for users who wish to process a video stream and keep the audio (or other streams). Instead of having to process the video and store in a file and then combine with the original audio later, the user can simply pass in the original file path via the `Options.StreamFile` parameter. This will combine the video with all other streams in the given file (Audio, Subtitle, Data, and Attachments Streams) and will cut all streams to be the same length. **Note that `Vidio` is not a audio/video editing library.**

This means that adding extra stream data from a file will only work if the filename being written to is a container format.

## Images

`Vidio` provides some convenience functions for reading and writing to images using an array of bytes. Currently, only `png` and `jpeg` formats are supported. When reading images, an optional `buffer` can be passed in to avoid array reallocation.

```go
Read(filename string, buffer ...[]byte) (int, int, []byte, error)
Write(filename string, width, height int, buffer []byte) error
```

## Examples

Copy `input.mp4` to `output.mp4`. Copy the audio from `input.mp4` to `output.mp4` as well.

```go
video, _ := vidio.NewVideo("input.mp4")
options := vidio.Options{
	FPS: video.FPS(),
	Bitrate: video.Bitrate(),
}
if video.HasStreams() {
	options.StreamFile = video.FileName()
}

writer, _ := vidio.NewVideoWriter("output.mp4", video.Width(), video.Height(), &options)

defer writer.Close()

for video.Read() {
    writer.Write(video.FrameBuffer())
}
```

Read 1000 frames of a webcam stream and store in `output.mp4`.

```go
webcam, _ := vidio.NewCamera(0)
defer webcam.Close()

options := vidio.Options{FPS: webcam.FPS()}
writer, _ := vidio.NewVideoWriter("output.mp4", webcam.Width(), webcam.Height(), &options)
defer writer.Close()

count := 0
for webcam.Read() && count < 1000 {
	writer.Write(webcam.FrameBuffer())
	count++
}
```

Create a gif from a series of `png` files enumerated from 1 to 10 that loops continuously with a final frame delay of 1000 centiseconds.

```go
w, h, _, _ := vidio.Read("1.png") // Get frame dimensions from first image

options := vidio.Options{FPS: 1, Loop: 0, Delay: 1000}
gif, _ := vidio.NewVideoWriter("output.gif", w, h, &options)
defer gif.Close()

for i := 1; i <= 10; i++ {
	w, h, img, _ := vidio.Read(fmt.Sprintf("%d.png", i))
	gif.Write(img)
}
```

Write all frames of `video.mp4` as `jpg` images.

```go
video, _ := vidio.NewVideo("video.mp4")

img := image.NewRGBA(image.Rect(0, 0, video.Width(), video.Height()))
video.SetFrameBuffer(img.Pix)

frame := 0
for video.Read() {
	f, _ := os.Create(fmt.Sprintf("%d.jpg", frame))
	jpeg.Encode(f, img, nil)
	f.Close()
	frame++
}
```

Write the last frame of `video.mp4` as `jpg` image (without iterating over all video frames).

```go
video, _ := video.NewVideo("video.mp4")

img := image.NewRGBA(image.Rect(0, 0, video.Width(), video.Height()))
video.SetFrameBuffer(img.Pix)

video.ReadFrame(video.Frames() - 1)

f, _ := os.Create(fmt.Sprintf("%d.jpg", video.Frames() - 1))
jpeg.Encode(f, img, nil)
f.Close()
```

Write the first and last frames of `video.mp4` as `jpg` images (without iterating over all video frames).

```go
video, _ := vidio.NewVideo("video.mp4")

frames, _ := video.ReadFrames(0, video.Frames() - 1)

img := image.NewRGBA(image.Rect(0, 0, video.Width(), video.Height()))
for index, frame := range frames {
	copy(img.Pix, frame)

	f, _ := os.Create(fmt.Sprintf("%d.jpg", index))
	jpeg.Encode(f, img, nil)
	f.Close()
}
```

# Acknowledgements

* Special thanks to [Zulko](http://zulko.github.io/) and his [blog post](http://zulko.github.io/blog/2013/09/27/read-and-write-video-frames-in-python-using-ffmpeg/) about using FFmpeg to process video.
* The [ImageIO-FFMPEG](https://github.com/imageio/imageio-ffmpeg/) project on GitHub.