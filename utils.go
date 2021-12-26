package main

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

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

// Checks if the given program is installed.
func CheckExists(program string) {
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

func parse(data string) float64 {
	n, err := strconv.ParseFloat(data, 64)
	if err != nil {
		return 0
	}
	return n
}

// func main() {
// 	os := runtime.GOOS
// 	switch os {
// 	case "windows":
// 		fmt.Println("Windows")
// 	case "darwin":
// 		fmt.Println("MAC operating system")
// 	case "linux":
// 		fmt.Println("Linux")
// 	default:
// 		fmt.Printf("%s.\n", os)
// 	}
// }
