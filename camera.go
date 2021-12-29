package vidio

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

type Camera struct {
	name        string         // Camera device name.
	width       int            // Camera frame width.
	height      int            // Camera frame height.
	depth       int            // Camera frame depth.
	fps         float64        // Camera frame rate.
	codec       string         // Camera codec.
	framebuffer []byte         // Raw frame data.
	pipe        *io.ReadCloser // Stdout pipe for ffmpeg process streaming webcam.
	cmd         *exec.Cmd      // ffmpeg command.
}

// Returns the webcam device name.
// On windows, ffmpeg output from the -list_devices command is parsed to find the device name.
func getDevicesWindows() []string {
	// Run command to get list of devices.
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-list_devices", "true",
		"-f", "dshow",
		"-i", "dummy",
	)
	pipe, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	// Read list devices from Stdout.
	buffer := make([]byte, 2<<10)
	total := 0
	for {
		n, err := pipe.Read(buffer[total:])
		total += n
		if err == io.EOF {
			break
		}
	}
	cmd.Wait()
	devices := parseDevices(buffer)
	return devices
}

// Get camera meta data such as width, height, fps and codec.
func getCameraData(device string, camera *Camera) {
	// Run command to get camera data.
	// Webcam will turn on and then off in quick succession.
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-f", webcam(),
		"-i", device,
	)
	// The command will fail since we do not give a file to write to, therefore
	// it will write the meta data to Stderr.
	pipe, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	// Start the command.
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	// Read ffmpeg output from Stdout.
	buffer := make([]byte, 2<<11)
	total := 0
	for {
		n, err := pipe.Read(buffer[total:])
		total += n
		if err == io.EOF {
			break
		}
	}
	// Wait for the command to finish.
	cmd.Wait()

	parseWebcamData(buffer[:total], camera)
}

// Creates a new camera struct that can read from the device with the given stream index.
func NewCamera(stream int) *Camera {
	// Check if ffmpeg is installed on the users machine.
	checkExists("ffmpeg")

	var device string
	switch runtime.GOOS {
	case "linux":
		device = "/dev/video" + strconv.Itoa(stream)
		break
	case "darwin":
		device = strconv.Itoa(stream)
		break
	case "windows":
		// If OS is windows, we need to parse the listed devices to find which corresponds to the
		// given "stream" index.
		devices := getDevicesWindows()
		if stream >= len(devices) {
			panic("Could not find devices with index: " + strconv.Itoa(stream))
		}
		device = "video=" + devices[stream]
		break
	default:
		panic("Unsupported OS: " + runtime.GOOS)
	}

	camera := Camera{name: device, depth: 3}
	getCameraData(device, &camera)
	return &camera
}

// Once the user calls Read() for the first time on a Camera struct,
// the ffmpeg command which is used to read the camera device is started.
func initCamera(camera *Camera) {
	// If user exits with Ctrl+C, stop ffmpeg process.
	camera.cleanup()

	// Use ffmpeg to pipe webcam to stdout.
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "quiet",
		"-f", webcam(),
		"-i", camera.name,
		"-f", "image2pipe",
		"-pix_fmt", "rgb24",
		"-vcodec", "rawvideo", "-",
	)

	camera.cmd = cmd
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	camera.pipe = &pipe
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	camera.framebuffer = make([]byte, camera.width*camera.height*camera.depth)
}

// Reads the next frame from the webcam and stores in the framebuffer.
func (camera *Camera) Read() bool {
	// If cmd is nil, video reading has not been initialized.
	if camera.cmd == nil {
		initCamera(camera)
	}
	total := 0
	for total < camera.width*camera.height*camera.depth {
		n, _ := (*camera.pipe).Read(camera.framebuffer[total:])
		total += n
	}
	return true
}

// Closes the pipe and stops the ffmpeg process.
func (camera *Camera) Close() {
	if camera.pipe != nil {
		(*camera.pipe).Close()
	}
	if camera.cmd != nil {
		camera.cmd.Process.Kill()
	}
}

// Stops the "cmd" process running when the user presses Ctrl+C.
// https://stackoverflow.com/questions/11268943/is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in-a-defe.
func (camera *Camera) cleanup() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if camera.pipe != nil {
			(*camera.pipe).Close()
		}
		if camera.cmd != nil {
			camera.cmd.Process.Kill()
		}
		os.Exit(1)
	}()
}
