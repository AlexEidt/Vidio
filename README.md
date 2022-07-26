# Vidio

A simple Video I/O library written in Go. This library relies on [FFmpeg](https://www.ffmpeg.org/), and [FFProbe](https://www.ffmpeg.org/) which must be downloaded before usage and added to the system path.

All frames are encoded and decoded in 8-bit RGB format.

## Installation

```
go get github.com/AlexEidt/Vidio
```

## `Video`

The `Video` struct stores data about a video file you give it. The code below shows an example of sequentially reading the frames of the given video.

Calling the `Read()` function will fill in the `Video` struct `framebuffer` with the next frame data as 8-bit RGB data, stored in a flattened byte array in row-major order where each pixel is represented by three consecutive bytes representing the R, G and B component of that pixel.

```go
vidio.NewVideo() (*Video, error)

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
FrameBuffer() []byte
SetFrameBuffer(buffer []byte)

Read() bool
Close()
```

If all frames have been read, `video` will be closed automatically. If not all frames are read, call `video.Close()` to close the video.

## `Camera`

The `Camera` can read from any cameras on the device running Vidio. It takes in the stream index. On most machines the webcam device has index 0.

```go
vidio.NewCamera(stream int) (*Camera, error)

Name() string
Width() int
Height() int
Depth() int
FPS() float64
Codec() string
FrameBuffer() []byte
SetFrameBuffer(buffer []byte)

Read() bool
Close()
```

## `VideoWriter`

The `VideoWriter` is used to write frames to a video file. The only required parameters are the output file name, the width and height of the frames being written, and an `Options` struct. This contains all the desired properties of the new video you want to create.

```go
vidio.NewVideoWriter() (*VideoWriter, error)

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

Write(frame []byte) error
Close()
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
	Audio       string          // File path for audio for the video. If no audio, audio=""
	AudioCodec  string          // Codec for audio. Default aac
}
```

## The `SetFrameBuffer(buffer []byte)` function

For the `SetFrameBuffer()` function, the `buffer` parameter must have a length of at least `video.Width() * video.Height() * video.Depth()` bytes to store the incoming video frame. The length of the buffer is not checked. It may be useful to have multiple buffers to keep track of previous video frames without having to copy data around.

## Images

Vidio provides some convenience functions for reading and writing to images using an array of bytes. Currently, only `png` and `jpeg` formats are supported. When reading images, an optional `buffer` can be passed in to avoid array reallocation.

```go
Read(filename string, buffer ...[]byte) (int, int, []byte, error)
Write(filename string, width, height int, buffer []byte) error
```

## Examples

Copy `input.mp4` to `output.mp4`. Copy the audio from `input.mp4` to `output.mp4` as well.

```go
video, err := vidio.NewVideo("input.mp4")

options := vidio.Options{
	FPS: video.FPS(),
	Bitrate: video.Bitrate(),
}
if video.AudioCodec() != "" {
	options.Audio = "input.mp4"
}

writer, err := vidio.NewVideoWriter("output.mp4", video.Width(), video.Height(), &options)

defer writer.Close()

for video.Read() {
    writer.Write(video.FrameBuffer())
}
```

Grayscale 1000 frames of webcam stream and store in `output.mp4`.

```go
webcam, err := vidio.NewCamera(0)

defer webcam.Close()

options := vidio.Options{FPS: webcam.FPS()}

writer, err := vidio.NewVideoWriter("output.mp4", webcam.Width(), webcam.Height(), &options)

defer writer.Close()

count := 0
for webcam.Read() && count < 1000{
	frame := webcam.FrameBuffer()
	for i := 0; i < len(frame); i += 3 {
		r, g, b := frame[i+0], frame[i+1], frame[i+2]
		gray := uint8((3*int(r) + 4*int(g) + int(b)) / 8)
		frame[i] = gray
		frame[i+1] = gray
		frame[i+2] = gray
	}

	writer.Write(frame)

	count++
}
```

Create a gif from a series of `png` files enumerated from 1 to 10 that loops continuously with a final frame delay of 1000 centiseconds.

```go
w, h, img, err := vidio.Read("1.png") // Get frame dimensions from first image

options := vidio.Options{FPS: 1, Loop: 0, Delay: 1000}

gif, err := vidio.NewVideoWriter("output.gif", w, h, &options)

defer gif.Close()

for i := 1; i <= 10; i++ {
	w, h, img, err := vidio.Read(strconv.Itoa(i)+".png")
	gif.Write(img)
}
```

# Acknowledgements

* Special thanks to [Zulko](http://zulko.github.io/) and his [blog post](http://zulko.github.io/blog/2013/09/27/read-and-write-video-frames-in-python-using-ffmpeg/) about using FFmpeg to process video.
* The [ImageIO-FFMPEG](https://github.com/imageio/imageio-ffmpeg/) project on GitHub.