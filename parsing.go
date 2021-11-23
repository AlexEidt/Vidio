package main

import (
	"regexp"
	"strconv"
	"strings"
)

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
