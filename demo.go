package main

import "fmt"

func main() {
	// Try it yourself!
	// Update "filename" to a video file on your system and
	// create and output file you'd like to copy this video to.
	filename := "input.mp4"
	output := "output.mp4"
	video := NewVideo(filename)

	writer := NewVideoWriter(output, video)
	defer writer.Close()

	count := 0
	for video.NextFrame() {
		writer.Write(video.framebuffer)
		count += 1
		fmt.Println(count)
	}
}
