package vidio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Returns true if file exists, false otherwise.
// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go.
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
func checkExists(program string) error {
	cmd := exec.Command(program, "-version")
	errmsg := fmt.Errorf("%s is not installed", program)
	if err := cmd.Start(); err != nil {
		return errmsg
	}
	if err := cmd.Wait(); err != nil {
		return errmsg
	}
	return nil
}

// Runs ffprobe on the given file and returns a map of the metadata.
func ffprobe(filename, stype string) (map[string]string, error) {
	// "stype" is stream stype. "v" for video, "a" for audio.
	// Extract video information with ffprobe.
	cmd := exec.Command(
		"ffprobe",
		"-show_streams",
		"-select_streams", stype,
		"-print_format", "compact",
		"-loglevel", "quiet",
		filename,
	)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
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
		return nil, err
	}

	return parseFFprobe(buffer[:total]), nil
}

// Parse ffprobe output to fill in video data.
func parseFFprobe(input []byte) map[string]string {
	data := make(map[string]string)
	for _, line := range strings.Split(string(input), "|") {
		if strings.Contains(line, "=") {
			keyValue := strings.Split(line, "=")
			if _, ok := data[keyValue[0]]; !ok {
				data[keyValue[0]] = keyValue[1]
			}
		}
	}
	return data
}

// Adds Video data to the video struct from the ffprobe output.
func addVideoData(data map[string]string, video *Video) {
	if width, ok := data["width"]; ok {
		video.width = int(parse(width))
	}
	if height, ok := data["height"]; ok {
		video.height = int(parse(height))
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
	if pixfmt, ok := data["pix_fmt"]; ok {
		video.pixfmt = pixfmt
	}
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
func webcam() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return "v4l2", nil
	case "darwin":
		return "avfoundation", nil // qtkit
	case "windows":
		return "dshow", nil // vfwcap
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// For webcam streaming on windows, ffmpeg requires a device name.
// All device names are parsed and returned by this function.
func parseDevices(buffer []byte) []string {
	bufferstr := string(buffer)

	index := strings.Index(strings.ToLower(bufferstr), "directshow video device")
	if index != -1 {
		bufferstr = bufferstr[index:]
	}

	index = strings.Index(strings.ToLower(bufferstr), "directshow audio device")
	if index != -1 {
		bufferstr = bufferstr[:index]
	}

	type Pair struct {
		name string
		alt  string
	}
	// Parses ffmpeg output to get device names. Windows only.
	// Uses parsing approach from https://github.com/imageio/imageio/blob/master/imageio/plugins/ffmpeg.py#L681.

	pairs := []Pair{}
	// Find all device names surrounded by quotes. E.g "Windows Camera Front"
	regex := regexp.MustCompile("\"[^\"]+\"")
	for _, line := range strings.Split(strings.ReplaceAll(bufferstr, "\r\n", "\n"), "\n") {
		if strings.Contains(strings.ToLower(line), "alternative name") {
			match := regex.FindString(line)
			if len(match) > 0 {
				pairs[len(pairs)-1].alt = match[1 : len(match)-1]
			}
		} else {
			match := regex.FindString(line)
			if len(match) > 0 {
				pairs = append(pairs, Pair{name: match[1 : len(match)-1]})
			}
		}
	}

	devices := []string{}
	// If two devices have the same name, use the alternate name of the later device as its name.
	for _, pair := range pairs {
		if contains(devices, pair.name) {
			devices = append(devices, pair.alt)
		} else {
			devices = append(devices, pair.name)
		}
	}
	return devices
}

// Helper function. Array contains function.
func contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

// Parses the webcam metadata (width, height, fps, codec) from ffmpeg output.
func parseWebcamData(buffer []byte, camera *Camera) {
	bufferstr := string(buffer)
	index := strings.Index(bufferstr, "Stream #")
	if index == -1 {
		index++
	}
	bufferstr = bufferstr[index:]
	// Dimensions. widthxheight.
	regex := regexp.MustCompile(`\d{2,}x\d{2,}`)
	match := regex.FindString(bufferstr)
	if len(match) > 0 {
		split := strings.Split(match, "x")
		camera.width = int(parse(split[0]))
		camera.height = int(parse(split[1]))
	}
	// FPS.
	regex = regexp.MustCompile(`\d+(.\d+)? fps`)
	match = regex.FindString(bufferstr)
	if len(match) > 0 {
		index = strings.Index(match, " fps")
		if index != -1 {
			match = match[:index]
		}
		camera.fps = parse(match)
	}
	// Codec.
	regex = regexp.MustCompile("Video: .+,")
	match = regex.FindString(bufferstr)
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
