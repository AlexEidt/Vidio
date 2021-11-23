package main

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type Video struct {
	filename    string
	width       int
	height      int
	channels    int
	bitrate     int
	duration    float64
	fps         float64
	codec       string
	pix_fmt     string
	framebuffer []byte
	pipe        *io.ReadCloser
	cmd         *exec.Cmd
}

func NewVideo(filename string) *Video {
	if !Exists(filename) {
		panic("File: " + filename + " does not exist")
	}
	// Execute ffmpeg -i command to get video information.
	cmd := exec.Command("ffmpeg", "-i", filename, "-")
	// FFMPEG output piped to Stderr.
	pipe, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	buffer := make([]byte, 2<<12)
	total := 0
	for {
		n, err := pipe.Read(buffer[total:])
		total += n
		if err == io.EOF {
			break
		}
	}
	cmd.Wait()
	video := &Video{filename: filename, channels: 3, pipe: nil, framebuffer: nil}
	parseFFMPEGHeader(video, string(buffer))
	return video
}

func (video *Video) initVideoStream() {
	// If user exits with Ctrl+C, stop ffmpeg process.
	video.cleanup()

	cmd := exec.Command(
		"ffmpeg",
		"-loglevel", "quiet",
		"-i", video.filename,
		"-f", "image2pipe",
		"-pix_fmt", "rgb24",
		"-vcodec", "rawvideo", "-",
	)
	video.cmd = cmd
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	video.pipe = &pipe
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	video.framebuffer = make([]byte, video.width*video.height*video.channels)
}

func (video *Video) NextFrame() bool {
	// If cmd is nil, video reading has not been initialized.
	if video.cmd == nil {
		video.initVideoStream()
	}
	total := 0
	for total < video.width*video.height*video.channels {
		n, err := (*video.pipe).Read(video.framebuffer[total:])
		if err == io.EOF {
			(*video.pipe).Close()
			if err := video.cmd.Wait(); err != nil {
				panic(err)
			}
			return false
		}
		total += n
	}
	return true
}

// Stops the "cmd" process running when the user presses Ctrl+C.
// https://stackoverflow.com/questions/11268943/is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in-a-defe
func (video *Video) cleanup() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if video.pipe != nil {
			(*video.pipe).Close()
		}
		if video.cmd != nil {
			video.cmd.Process.Kill()
		}
		os.Exit(1)
	}()
}
