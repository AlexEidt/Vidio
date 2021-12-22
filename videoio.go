package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// #############################################################################
// Video
// #############################################################################

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
	// ffmpeg output piped to Stderr.
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
	video := &Video{filename: filename, channels: 3}
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

// #############################################################################
// VideoWriter
// #############################################################################

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

// #############################################################################
// Utils
// #############################################################################

// Parses the duration of the video from the ffmpeg header.
func parseDurationBitrate(video *Video, data []string) {
	videoData := ""
	for _, line := range data {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Duration: ") {
			videoData = line
			break
		}
	}
	if videoData == "" {
		panic("Could not find duration in ffmpeg header.")
	}
	// Duration
	duration := strings.Split(strings.SplitN(strings.SplitN(videoData, ",", 2)[0], "Duration:", 2)[1], ":")
	seconds, _ := strconv.ParseFloat(duration[len(duration)-1], 64)
	minutes, _ := strconv.ParseFloat(duration[len(duration)-2], 64)
	hours, _ := strconv.ParseFloat(duration[len(duration)-3], 64)
	video.duration = seconds + minutes*60 + hours*3600

	// Bitrate
	bitrate := strings.SplitN(strings.TrimSpace(strings.SplitN(videoData, "bitrate:", 2)[1]), " ", 2)[0]
	video.bitrate, _ = strconv.Atoi(bitrate)
}

func parseVideoData(video *Video, data []string) {
	videoData := ""
	// Get string containing video data.
	for _, line := range data {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Stream") && strings.Contains(line, "Video:") {
			videoData = strings.TrimSpace(strings.SplitN(line, "Video:", 2)[1])
			break
		}
	}
	if videoData == "" {
		panic("No video data found in ffmpeg header.")
	}
	// Video Codec
	video.codec = strings.TrimSpace(strings.SplitN(videoData, " ", 2)[0])
	// FPS
	fpsstr := strings.SplitN(videoData, "fps", 2)[0]
	fps, _ := strconv.Atoi(strings.TrimSpace(fpsstr[strings.LastIndex(fpsstr, ",")+1:]))
	video.fps = float64(fps)
	// Pixel Format
	video.pix_fmt = strings.TrimSpace(strings.Split(videoData, ",")[1])
	// Width and Height
	r, _ := regexp.Compile("\\d+x\\d+")
	wh := r.FindAllString(videoData, -1)
	dims := strings.SplitN(wh[len(wh)-1], "x", 2)
	width, _ := strconv.Atoi(dims[0])
	height, _ := strconv.Atoi(dims[1])
	video.width = width
	video.height = height
}

// Parses the ffmpeg header.
// Code inspired by the imageio-ffmpeg project.
// GitHub: https://github.com/imageio/imageio-ffmpeg/blob/master/imageio_ffmpeg/_parsing.py#L113
func parseFFMPEGHeader(video *Video, header string) {
	data := strings.Split(strings.ReplaceAll(header, "\r\n", "\n"), "\n")
	parseDurationBitrate(video, data)
	parseVideoData(video, data)
}

// #############################################################################
// Utils
// #############################################################################

// Returns true if file exists, false otherwise.
// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
func Exists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}
