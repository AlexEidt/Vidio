package vidio

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type Video struct {
	filename    string         // Video Filename.
	width       int            // Width of frames.
	height      int            // Height of frames.
	depth       int            // Depth of frames.
	bitrate     int            // Bitrate for video encoding.
	frames      int            // Total number of frames.
	duration    float64        // Duration of video in seconds.
	fps         float64        // Frames per second.
	codec       string         // Codec used for video encoding.
	pix_fmt     string         // Pixel format video is stored in.
	framebuffer []byte         // Raw frame data.
	pipe        *io.ReadCloser // Stdout pipe for ffmpeg process.
	cmd         *exec.Cmd      // ffmpeg command.
}

// Creates a new Video struct.
// Uses ffprobe to get video information and fills in the Video struct with this data.
func NewVideo(filename string) *Video {
	if !exists(filename) {
		panic("File: " + filename + " does not exist")
	}
	// Check if ffmpeg and ffprobe are installed on the users machine.
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
	// Wait for ffprobe command to complete.
	if err := cmd.Wait(); err != nil {
		panic(err)
	}
	video := &Video{filename: filename, depth: 3}
	parseFFprobe(buffer[:total], video)
	return video
}

// Once the user calls Read() for the first time on a Video struct,
// the ffmpeg command which is used to read the video is started.
func initVideo(video *Video) {
	// If user exits with Ctrl+C, stop ffmpeg process.
	video.cleanup()
	// ffmpeg command to pipe video data to stdout in 8-bit RGB format.
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

// Reads the next frame from the video and stores in the framebuffer.
// If thelast frame has been read, returns false, otherwise true.
func (video *Video) Read() bool {
	// If cmd is nil, video reading has not been initialized.
	if video.cmd == nil {
		initVideo(video)
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

// Closes the pipe and stops the ffmpeg process.
func (video *Video) Close() {
	if video.pipe != nil {
		(*video.pipe).Close()
	}
	if video.cmd != nil {
		video.cmd.Wait()
	}
}

// Stops the "cmd" process running when the user presses Ctrl+C.
// https://stackoverflow.com/questions/11268943/is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in-a-defe.
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
