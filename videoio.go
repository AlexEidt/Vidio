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

func NewVideo(filename string) *Video {
	if !exists(filename) {
		panic("File: " + filename + " does not exist")
	}
	checkExists("ffmpeg")
	checkExists("ffprobe")
	// Extract video information with ffprobe.
	cmd := exec.Command(
		"ffprobe",
		"-show_streams",
		"-select_streams", "v", // Only show video data
		"-print_format", "compact",
		"-loglevel", "quiet",
		filename,
	)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	// Read ffprobe output from Stdout.
	buffer := make([]byte, 2<<10)
	total := 0
	for {
		n, err := pipe.Read(buffer[total:])
		total += n
		if err == io.EOF {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		panic(err)
	}
	video := &Video{filename: filename, depth: 3}
	parseFFprobe(buffer[:total], video)
	return video
}

func initVideoStream(video *Video) {
	// If user exits with Ctrl+C, stop ffmpeg process.
	video.cleanup()

	// map = {1: "gray", 2: "gray8a", 3: "rgb24", 4: "rgba"}

	cmd := exec.Command(
		"ffmpeg",
		"-i", video.filename,
		"-f", "image2pipe",
		"-loglevel", "quiet",
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
	video.framebuffer = make([]byte, video.width*video.height*video.depth)
}

func (video *Video) NextFrame() bool {
	// If cmd is nil, video reading has not been initialized.
	if video.cmd == nil {
		initVideoStream(video)
	}
	total := 0
	for total < video.width*video.height*video.depth {
		n, err := (*video.pipe).Read(video.framebuffer[total:])
		if err == io.EOF {
			video.Close()
			return false
		}
		total += n
	}
	return true
}

func (video *Video) Close() {
	if video.pipe != nil {
		(*video.pipe).Close()
	}
	if video.cmd != nil {
		video.cmd.Wait()
	}
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
