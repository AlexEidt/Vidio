package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type VideoWriter struct {
	filename string
	width    int
	height   int
	bitrate  int
	fps      float64
	codec    string
	pix_fmt  string
	pipe     *io.WriteCloser
	cmd      *exec.Cmd
}

func NewVideoWriter(filename string, video *Video) *VideoWriter {
	if video.width == 0 || video.height == 0 {
		panic("Video width and height must be set.")
	}
	if video.fps == 0 {
		video.fps = 25 // Default to 25 FPS.
	}
	return &VideoWriter{
		filename: filename,
		width:    video.width,
		height:   video.height,
		bitrate:  video.bitrate,
		fps:      video.fps,
		codec:    "mpeg4",
		pix_fmt:  "rgb24",
	}
}

func (writer *VideoWriter) initVideoWriter() {
	// If user exits with Ctrl+C, stop ffmpeg process.
	writer.cleanup()

	cmd := exec.Command(
		"ffmpeg",
		"-y", // overwrite output file if it exists
		"-f", "rawvideo",
		"-vcodec", "rawvideo",
		"-s", fmt.Sprintf("%dx%d", writer.width, writer.height), // frame w x h
		"-pix_fmt", writer.pix_fmt,
		"-r", fmt.Sprintf("%f", writer.fps), // frames per second
		"-i", "-", // The imput comes from stdin
		"-an", // Tells ffmpeg not to expect any audio
		"-vcodec", writer.codec,
		"-b:v", fmt.Sprintf("%dk", writer.bitrate), // bitrate
		writer.filename,
	)
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
		writer.initVideoWriter()
	}
	total := 0
	for total < len(frame) {
		n, err := (*writer.pipe).Write(frame[total:])
		if err != nil {
			defer fmt.Println("Likely cause is invalid parameters to ffmpeg.")
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
