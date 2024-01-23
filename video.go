package vidio

import (
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

type Video struct {
	filename    string            // Video Filename.
	width       int               // Width of frames.
	height      int               // Height of frames.
	depth       int               // Depth of frames.
	bitrate     int               // Bitrate for video encoding.
	frames      int               // Total number of frames.
	stream      int               // Stream Index.
	duration    float64           // Duration of video in seconds.
	fps         float64           // Frames per second.
	codec       string            // Codec used for video encoding.
	hasstreams  bool              // Flag storing whether file has additional data streams.
	framebuffer []byte            // Raw frame data.
	metadata    map[string]string // Video metadata.
	pipe        io.ReadCloser     // Stdout pipe for ffmpeg process.
	cmd         *exec.Cmd         // ffmpeg command.

	closeCleanupChan chan struct{} // exit from cleanup goroutine to avoid chan and goroutine leak
	cleanupClosed bool
}

func (video *Video) FileName() string {
	return video.filename
}

func (video *Video) Width() int {
	return video.width
}

func (video *Video) Height() int {
	return video.height
}

// Channels of video frames.
func (video *Video) Depth() int {
	return video.depth
}

// Bitrate of video in bits/s.
func (video *Video) Bitrate() int {
	return video.bitrate
}

// Total number of frames in video.
func (video *Video) Frames() int {
	return video.frames
}

// Returns the zero-indexed video stream index.
func (video *Video) Stream() int {
	return video.stream
}

// Video duration in seconds.
func (video *Video) Duration() float64 {
	return video.duration
}

// Frames per second of video.
func (video *Video) FPS() float64 {
	return video.fps
}

func (video *Video) Codec() string {
	return video.codec
}

// Returns true if file has any audio, subtitle, data or attachment streams.
func (video *Video) HasStreams() bool {
	return video.hasstreams
}

func (video *Video) FrameBuffer() []byte {
	return video.framebuffer
}

// Raw Metadata from ffprobe output for the video file.
func (video *Video) MetaData() map[string]string {
	return video.metadata
}

func (video *Video) SetFrameBuffer(buffer []byte) error {
	size := video.width * video.height * video.depth
	if len(buffer) < size {
		return fmt.Errorf("vidio: buffer size %d is smaller than frame size %d", len(buffer), size)
	}
	video.framebuffer = buffer
	return nil
}

func NewVideo(filename string) (*Video, error) {
	streams, err := NewVideoStreams(filename)
	if streams == nil {
		return nil, err
	}

	return streams[0], err
}

// Read all video streams from the given file.
func NewVideoStreams(filename string) ([]*Video, error) {
	if !exists(filename) {
		return nil, fmt.Errorf("vidio: video file %s does not exist", filename)
	}
	// Check if ffmpeg and ffprobe are installed on the users machine.
	if err := installed("ffmpeg"); err != nil {
		return nil, err
	}
	if err := installed("ffprobe"); err != nil {
		return nil, err
	}

	videoData, err := ffprobe(filename, "v")
	if err != nil {
		return nil, err
	}

	if len(videoData) == 0 {
		return nil, fmt.Errorf("vidio: no video data found in %s", filename)
	}

	// Loop over all stream types. a: Audio, s: Subtitle, d: Data, t: Attachments
	hasstream := false
	for _, c := range "asdt" {
		data, err := ffprobe(filename, string(c))
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			hasstream = true
			break
		}
	}

	streams := make([]*Video, len(videoData))
	for i, data := range videoData {
		video := &Video{
			filename:   filename,
			depth:      4,
			stream:     i,
			hasstreams: hasstream,
			metadata:   data,

			closeCleanupChan: make(chan struct{}, 1),
		}

		video.addVideoData(data)

		streams[i] = video
	}

	return streams, nil
}

// Adds Video data to the video struct from the ffprobe output.
func (video *Video) addVideoData(data map[string]string) {
	if width, ok := data["width"]; ok {
		video.width = int(parse(width))
	}
	if height, ok := data["height"]; ok {
		video.height = int(parse(height))
	}
	if rotation, ok := data["tag:rotate"]; ok && (rotation == "90" || rotation == "270") {
		video.width, video.height = video.height, video.width
	}
	if duration, ok := data["duration"]; ok {
		video.duration = float64(parse(duration))
	}
	if frames, ok := data["nb_frames"]; ok {
		video.frames = int(parse(frames))
	}
	if fps, ok := data["r_frame_rate"]; ok {
		split := strings.Split(fps, "/")
		if len(split) == 2 && split[0] != "" && split[1] != "" {
			video.fps = parse(split[0]) / parse(split[1])
		}
	}
	if bitrate, ok := data["bit_rate"]; ok {
		video.bitrate = int(parse(bitrate))
	}
	if codec, ok := data["codec_name"]; ok {
		video.codec = codec
	}
}

// Once the user calls Read() for the first time on a Video struct,
// the ffmpeg command which is used to read the video is started.
func (video *Video) init() error {
	// If user exits with Ctrl+C, stop ffmpeg process.
	video.cleanup()
	// ffmpeg command to pipe video data to stdout in 8-bit RGBA format.
	cmd := exec.Command(
		"ffmpeg",
		"-i", video.filename,
		"-f", "image2pipe",
		"-loglevel", "quiet",
		"-pix_fmt", "rgba",
		"-vcodec", "rawvideo",
		"-map", fmt.Sprintf("0:v:%d", video.stream),
		"-",
	)

	video.cmd = cmd
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	video.pipe = pipe

	if err := cmd.Start(); err != nil {
		return err
	}

	if video.framebuffer == nil {
		video.framebuffer = make([]byte, video.width*video.height*video.depth)
	}

	return nil
}

// Reads the next frame from the video and stores in the framebuffer.
// If the last frame has been read, returns false, otherwise true.
func (video *Video) Read() bool {
	// If cmd is nil, video reading has not been initialized.
	if video.cmd == nil {
		if err := video.init(); err != nil {
			return false
		}
	}

	if _, err := io.ReadFull(video.pipe, video.framebuffer); err != nil {
		video.Close()
		return false
	}
	return true
}

// Reads the N-th frame from the video and stores it in the framebuffer. If the index is out of range or
// the operation failes, the function will return an error. The frames are indexed from 0.
func (video *Video) ReadFrame(n int) error {
	if n >= video.frames {
		return fmt.Errorf("vidio: provided frame index %d is not in frame count range", n)
	}

	if video.framebuffer == nil {
		video.framebuffer = make([]byte, video.width*video.height*video.depth)
	}

	selectExpression, err := buildSelectExpression(n)
	if err != nil {
		return fmt.Errorf("vidio: failed to parse the specified frame index: %w", err)
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i", video.filename,
		"-f", "image2pipe",
		"-loglevel", "quiet",
		"-pix_fmt", "rgba",
		"-vcodec", "rawvideo",
		"-map", fmt.Sprintf("0:v:%d", video.stream),
		"-vf", selectExpression,
		"-vsync", "0",
		"-",
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("vidio: failed to access the ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("vidio: failed to start the ffmpeg cmd: %w", err)
	}

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interruptChan
		if stdoutPipe != nil {
			stdoutPipe.Close()
		}
		if cmd != nil {
			cmd.Process.Kill()
		}
		os.Exit(1)
	}()

	if _, err := io.ReadFull(stdoutPipe, video.framebuffer); err != nil {
		return fmt.Errorf("vidio: failed to read the ffmpeg cmd result to the image buffer: %w", err)
	}

	if err := stdoutPipe.Close(); err != nil {
		return fmt.Errorf("vidio: failed to close the ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("vidio: failed to free resources after the ffmpeg cmd: %w", err)
	}

	return nil
}

// Read the N-amount of frames with the given indexes and return them as a slice of RGBA image pointers. If one of
// the indexes is out of range, the function will return an error. The frames are indexes from 0.
func (video *Video) ReadFrames(n ...int) ([]*image.RGBA, error) {
	if len(n) == 0 {
		return nil, fmt.Errorf("vidio: no frames indexes specified")
	}

	for _, nValue := range n {
		if nValue >= video.frames {
			return nil, fmt.Errorf("vidio: provided frame index %d is not in frame count range", nValue)
		}
	}

	selectExpression, err := buildSelectExpression(n...)
	if err != nil {
		return nil, fmt.Errorf("vidio: failed to parse the specified frame index: %w", err)
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i", video.filename,
		"-f", "image2pipe",
		"-loglevel", "quiet",
		"-pix_fmt", "rgba",
		"-vcodec", "rawvideo",
		"-map", fmt.Sprintf("0:v:%d", video.stream),
		"-vf", selectExpression,
		"-vsync", "0",
		"-",
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("vidio: failed to access the ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("vidio: failed to start the ffmpeg cmd: %w", err)
	}

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interruptChan
		if stdoutPipe != nil {
			stdoutPipe.Close()
		}
		if cmd != nil {
			cmd.Process.Kill()
		}
		os.Exit(1)
	}()

	frames := make([]*image.RGBA, len(n))
	for frameIndex := range frames {
		frames[frameIndex] = image.NewRGBA(image.Rect(0, 0, video.width, video.height))

		if _, err := io.ReadFull(stdoutPipe, frames[frameIndex].Pix); err != nil {
			return nil, fmt.Errorf("vidio: failed to read the ffmpeg cmd result to the image buffer: %w", err)
		}
	}

	if err := stdoutPipe.Close(); err != nil {
		return nil, fmt.Errorf("vidio: failed to close the ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("vidio: failed to free resources after the ffmpeg cmd: %w", err)
	}

	return frames, nil
}

// Closes the pipe and stops the ffmpeg process.
func (video *Video) Close() {
	if !video.cleanupClosed {
		video.cleanupClosed = true
		video.closeCleanupChan <- struct{}{}
		close(video.closeCleanupChan)
	}
	if video.pipe != nil {
		video.pipe.Close()
	}
	if video.cmd != nil {
		video.cmd.Wait()
	}
}

// Stops the "cmd" process running when the user presses Ctrl+C.
// https://stackoverflow.com/questions/11268943/is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in-a-defe.
func (video *Video) cleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			if video.pipe != nil {
				video.pipe.Close()
			}
			if video.cmd != nil {
				video.cmd.Process.Kill()
			}
			os.Exit(1)
		case <-video.closeCleanupChan:
			signal.Stop(c)
			close(c)
		}
	}()
}
