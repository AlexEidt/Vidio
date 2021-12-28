package vidio

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Returns true if file exists, false otherwise.
// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
func exists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

// Checks if the given program is installed.
func checkExists(program string) {
	cmd := exec.Command(program, "-version")
	if err := cmd.Start(); err != nil {
		panic(program + " is not installed.")
	}
	if err := cmd.Wait(); err != nil {
		panic(program + " is not installed.")
	}
}

// Parse ffprobe output to fill in video data.
func parseFFprobe(input []byte, video *Video) {
	data := make(map[string]string)
	for _, line := range strings.Split(string(input), "|") {
		if strings.Contains(line, "=") {
			keyValue := strings.Split(line, "=")
			if _, ok := data[keyValue[0]]; !ok {
				data[keyValue[0]] = keyValue[1]
			}
		}
	}

	video.width = int(parse(data["width"]))
	video.height = int(parse(data["height"]))
	video.duration = float64(parse(data["duration"]))
	video.frames = int(parse(data["nb_frames"]))

	split := strings.Split(data["r_frame_rate"], "/")
	if len(split) == 2 && split[0] != "" && split[1] != "" {
		video.fps = parse(split[0]) / parse(split[1])
	}

	video.bitrate = int(parse(data["bit_rate"]))
	video.codec = data["codec_name"]
	video.pix_fmt = data["pix_fmt"]
}

// Parses the given data into a float64.
func parse(data string) float64 {
	n, err := strconv.ParseFloat(data, 64)
	if err != nil {
		return 0
	}
	return n
}

// Returns the webcam name used for the -f option with ffmpeg.
func webcam() string {
	os := runtime.GOOS
	switch os {
	case "linux":
		return "v4l2"
	case "darwin":
		return "avfoundation" // qtkit
	case "windows":
		return "dshow" // vfwcap
	default:
		panic("Unsupported OS: " + os)
	}
}

// For webcam streaming on windows, ffmpeg requires a device name.
// All device names are parsed and returned by this function.
func parseDevices(buffer []byte) []string {
	devices := make([]string, 0)
	bufferstr := string(buffer)

	index := strings.Index(bufferstr, "DirectShow video device")
	if index == -1 {
		return devices
	}
	bufferstr = bufferstr[index:]

	index = strings.Index(bufferstr, "DirectShow audio device")
	if index != -1 {
		bufferstr = bufferstr[:index]
	}
	// Find all device names surrounded by quotes. E.g "Windows Camera Front"
	r := regexp.MustCompile("\"[^\"]+\"")
	matches := r.FindAllStringSubmatch(bufferstr, -1)
	for _, match := range matches {
		device := match[0][1 : len(match[0])-1]
		// Don't include Alternate Names for devices.
		// Alternate names start with an '@'.
		if !strings.HasPrefix(device, "@") {
			devices = append(devices, device)
		}
	}

	return devices
}

// Parses the webcam metadata (width, height, fps, codec) from ffmpeg output.
func parseWebcamData(buffer []byte, camera *Camera) {
	bufferstr := string(buffer)
	index := strings.Index(bufferstr, "Stream #")
	if index == -1 {
		index++
	}
	bufferstr = bufferstr[index:]
	// Dimensions. widthxheight
	regex := regexp.MustCompile("\\d{2,}x\\d{2,}")
	match := regex.FindString(bufferstr)
	if len(match) > 0 {
		split := strings.Split(match, "x")
		camera.width = int(parse(split[0]))
		camera.height = int(parse(split[1]))
	}
	// FPS
	regex = regexp.MustCompile("\\d+(.\\d+)? fps")
	match = regex.FindString(bufferstr)
	if len(match) > 0 {
		index = strings.Index(match, " fps")
		if index != -1 {
			match = match[:index]
		}
		camera.fps = parse(match)
	}
	// Codec
	regex = regexp.MustCompile("Video: .+,")
	match = regex.FindString(bufferstr)
	if len(match) > 0 {
		match = match[7:]
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
