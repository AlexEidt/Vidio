package vidio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

type Camera struct {
	name        string        // Camera device name.
	width       int           // Camera frame width.
	height      int           // Camera frame height.
	depth       int           // Camera frame depth.
	fps         float64       // Camera frame rate.
	codec       string        // Camera codec.
	framebuffer []byte        // Raw frame data.
	pipe        io.ReadCloser // Stdout pipe for ffmpeg process streaming webcam.
	cmd         *exec.Cmd     // ffmpeg command.
}

// Camera device name.
func (camera *Camera) Name() string {
	return camera.name
}

func (camera *Camera) Width() int {
	return camera.width
}

func (camera *Camera) Height() int {
	return camera.height
}

// Channels of video frames.
func (camera *Camera) Depth() int {
	return camera.depth
}

// Frames per second of video.
func (camera *Camera) FPS() float64 {
	return camera.fps
}

func (camera *Camera) Codec() string {
	return camera.codec
}

func (camera *Camera) FrameBuffer() []byte {
	return camera.framebuffer
}

func (camera *Camera) SetFrameBuffer(buffer []byte) error {
	size := camera.width * camera.height * camera.depth
	if len(buffer) < size {
		return fmt.Errorf("vidio: buffer size %d is smaller than frame size %d", len(buffer), size)
	}
	camera.framebuffer = buffer
	return nil
}

// Creates a new camera struct that can read from the device with the given stream index.
func NewCamera(stream int) (*Camera, error) {
	// Check if ffmpeg is installed on the users machine.
	if err := installed("ffmpeg"); err != nil {
		return nil, err
	}

	var device string
	switch runtime.GOOS {
	case "linux":
		device = fmt.Sprintf("/dev/video%d", stream)
	case "darwin":
		device = fmt.Sprintf(`"%d"`, stream)
	case "windows":
		// If OS is windows, we need to parse the listed devices to find which corresponds to the
		// given "stream" index.
		devices, err := getDevicesWindows()
		if err != nil {
			return nil, err
		}
		if stream < 0 || stream >= len(devices) {
			return nil, fmt.Errorf("vidio: could not find device with index: %d", stream)
		}
		device = fmt.Sprintf("video=%s", devices[stream])
	default:
		return nil, fmt.Errorf("vidio: unsupported OS: %s", runtime.GOOS)
	}

	camera := &Camera{name: device, depth: 4}
	if err := camera.getCameraData(device); err != nil {
		return nil, err
	}

	return camera, nil
}

// Parses the webcam metadata (width, height, fps, codec) from ffmpeg output.
func (camera *Camera) parseWebcamData(buffer string) {
	index := strings.Index(buffer, "Stream #")
	if index == -1 {
		index++
	}
	buffer = buffer[index:]
	// Dimensions. widthxheight.
	regex := regexp.MustCompile(`\d{2,}x\d{2,}`)
	match := regex.FindString(buffer)
	if len(match) > 0 {
		split := strings.Split(match, "x")
		camera.width = int(parse(split[0]))
		camera.height = int(parse(split[1]))
	}
	// FPS.
	regex = regexp.MustCompile(`\d+(.\d+)? fps`)
	match = regex.FindString(buffer)
	if len(match) > 0 {
		index = strings.Index(match, " fps")
		if index != -1 {
			match = match[:index]
		}
		camera.fps = parse(match)
	}
	// Codec.
	regex = regexp.MustCompile("Video: .+,")
	match = regex.FindString(buffer)
	if len(match) > 0 {
		match = match[len("Video: "):]
		index = strings.Index(match, "(")
		if index != -1 {
			match = match[:index]
		}
		index = strings.Index(match, ",")
		if index != -1 {
			match = match[:index]
		}
		camera.codec = strings.TrimSpace(match)
	}
}

// Get camera meta data such as width, height, fps and codec.
func (camera *Camera) getCameraData(device string) error {
	// Run command to get camera data.
	// Webcam will turn on and then off in quick succession.
	webcamDeviceName, err := webcam()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-f", webcamDeviceName,
		"-i", device,
	)

	// The command will fail since we do not give a file to write to, therefore
	// it will write the meta data to Stderr.
	pipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the command.
	if err := cmd.Start(); err != nil {
		return err
	}

	// Read ffmpeg output from Stdout.
	builder := bytes.Buffer{}
	buffer := make([]byte, 1024)
	for {
		n, err := pipe.Read(buffer)
		builder.Write(buffer[:n])
		if err == io.EOF {
			break
		}
	}

	// Wait for the command to finish.
	cmd.Wait()

	camera.parseWebcamData(builder.String())
	return nil
}

// Once the user calls Read() for the first time on a Camera struct,
// the ffmpeg command which is used to read the camera device is started.
func (camera *Camera) init() error {
	// If user exits with Ctrl+C, stop ffmpeg process.
	camera.cleanup()

	webcamDeviceName, err := webcam()
	if err != nil {
		return err
	}

	// Use ffmpeg to pipe webcam to stdout.
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "quiet",
		"-f", webcamDeviceName,
		"-i", camera.name,
		"-f", "image2pipe",
		"-pix_fmt", "rgba",
		"-vcodec", "rawvideo",
		"-",
	)

	camera.cmd = cmd
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	camera.pipe = pipe
	if err := cmd.Start(); err != nil {
		return err
	}

	if camera.framebuffer == nil {
		camera.framebuffer = make([]byte, camera.width*camera.height*camera.depth)
	}

	return nil
}

// Reads the next frame from the webcam and stores in the framebuffer.
func (camera *Camera) Read() bool {
	// If cmd is nil, video reading has not been initialized.
	if camera.cmd == nil {
		if err := camera.init(); err != nil {
			return false
		}
	}

	if _, err := io.ReadFull(camera.pipe, camera.framebuffer); err != nil {
		camera.Close()
		return false
	}

	return true
}

// Closes the pipe and stops the ffmpeg process.
func (camera *Camera) Close() {
	if camera.pipe != nil {
		camera.pipe.Close()
	}
	if camera.cmd != nil {
		camera.cmd.Process.Kill()
	}
}

// Stops the "cmd" process running when the user presses Ctrl+C.
// https://stackoverflow.com/questions/11268943/is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in-a-defe.
func (camera *Camera) cleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if camera.pipe != nil {
			camera.pipe.Close()
		}
		if camera.cmd != nil {
			camera.cmd.Process.Kill()
		}
		os.Exit(1)
	}()
}
