package vidio

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

type VideoWriter struct {
	filename string
	width    int
	height   int
	bitrate  int
	loop     int // For GIFs. -1=no loop, 0=loop forever, >0=loop n times
	delay    int // Delay for Final Frame of GIFs.
	macro    int
	fps      float64
	quality  float64
	codec    string
	pipe     *io.WriteCloser
	cmd      *exec.Cmd
}

type Options struct {
	bitrate int
	loop    int
	delay   int
	macro   int
	fps     float64
	quality float64
	codec   string
}

func NewVideoWriter(filename string, width, height int, options *Options) *VideoWriter {
	// Check if ffmpeg is installed on the users machine.
	checkExists("ffmpeg")

	writer := VideoWriter{filename: filename}

	writer.width = width
	writer.height = height

	// Default Parameter options logic from:
	// https://github.com/imageio/imageio-ffmpeg/blob/master/imageio_ffmpeg/_io.py#L268

	// GIF settings
	writer.loop = options.loop // Default to infinite loop
	if options.delay == 0 {
		writer.delay = -1 // Default to frame delay of previous frame
	} else {
		writer.delay = options.delay
	}

	if options.macro == 0 {
		writer.macro = 16
	} else {
		writer.macro = options.macro
	}

	if options.fps == 0 {
		writer.fps = 25
	} else {
		writer.fps = options.fps
	}

	if options.quality == 0 {
		writer.quality = 0.5
	} else {
		writer.quality = options.quality
	}

	if options.codec == "" {
		if strings.HasSuffix(strings.ToLower(filename), ".wmv") {
			writer.codec = "msmpeg4"
		} else if strings.HasSuffix(strings.ToLower(filename), ".gif") {
			writer.codec = "gif"
		} else {
			writer.codec = "libx264"
		}
	} else {
		writer.codec = options.codec
	}
	return &writer
}

func initVideoWriter(writer *VideoWriter) {
	// If user exits with Ctrl+C, stop ffmpeg process.
	writer.cleanup()

	command := []string{
		"-y", // overwrite output file if it exists
		"-loglevel", "quiet",
		"-f", "rawvideo",
		"-vcodec", "rawvideo",
		"-s", fmt.Sprintf("%dx%d", writer.width, writer.height), // frame w x h
		"-pix_fmt", "rgb24",
		"-r", fmt.Sprintf("%.02f", writer.fps), // frames per second
		"-i", "-", // The input comes from stdin
		"-an", // Tells ffmpeg not to expect any audio
		"-vcodec", writer.codec,
		"-pix_fmt", "yuv420p", // Output is 8-bit RGB, no alpha
	}

	// Code from the imageio-ffmpeg project.
	// https://github.com/imageio/imageio-ffmpeg/blob/master/imageio_ffmpeg/_io.py#L399
	// If bitrate not given, use a default.
	if writer.bitrate == 0 {
		if writer.codec == "libx264" {
			// Quality between 0 an 51. 51 is worst.
			command = append(command, "-crf", fmt.Sprintf("%d", int(writer.quality*51)))
		} else {
			// Quality between 1 and 31. 31 is worst.
			command = append(command, "-qscale:v", fmt.Sprintf("%d", int(writer.quality*30)+1))
		}
	} else {
		command = append(command, "-b:v", fmt.Sprintf("%d", writer.bitrate))
	}

	if strings.HasSuffix(strings.ToLower(writer.filename), ".gif") {
		command = append(
			command,
			"-loop", fmt.Sprintf("%d", writer.loop),
			"-final_delay", fmt.Sprintf("%d", writer.delay),
		)
	}

	// Code from the imageio-ffmpeg project:
	// https://github.com/imageio/imageio-ffmpeg/blob/master/imageio_ffmpeg/_io.py#L415
	// Resizes the video frames to a size that works with most codecs.
	if writer.macro > 1 {
		if writer.width%writer.macro > 0 || writer.height%writer.macro > 0 {
			width := writer.width
			height := writer.height
			if writer.width%writer.macro > 0 {
				width += writer.macro - (writer.width % writer.macro)
			}
			if writer.height%writer.macro > 0 {
				height += writer.macro - (writer.height % writer.macro)
			}
			command = append(
				command,
				"-vf", fmt.Sprintf("scale=%d:%d", width, height),
			)
		}
	}

	command = append(command, writer.filename)
	cmd := exec.Command("ffmpeg", command...)
	writer.cmd = cmd

	pipe, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	writer.pipe = &pipe
	if err := cmd.Start(); err != nil {
		panic(err)
	}
}

func (writer *VideoWriter) Write(frame []byte) {
	// If cmd is nil, video writing has not been set up.
	if writer.cmd == nil {
		initVideoWriter(writer)
	}
	total := 0
	for total < len(frame) {
		n, err := (*writer.pipe).Write(frame[total:])
		if err != nil {
			fmt.Println("Likely cause is invalid parameters to ffmpeg.")
			panic(err)
		}
		total += n
	}
}

func (writer *VideoWriter) Close() {
	if writer.pipe != nil {
		(*writer.pipe).Close()
	}
	if writer.cmd != nil {
		writer.cmd.Wait()
	}
}

// Stops the "cmd" process running when the user presses Ctrl+C.
// https://stackoverflow.com/questions/11268943/is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in-a-defe
func (writer *VideoWriter) cleanup() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if writer.pipe != nil {
			(*writer.pipe).Close()
		}
		if writer.cmd != nil {
			writer.cmd.Process.Kill()
		}
		os.Exit(1)
	}()
}
