package vidio

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// Return the N-th frame of the video specified by the filename as a pointer to a RGBA image. The frames are indexed from 0.
func GetVideoFrame(filename string, n int) (*image.RGBA, error) {
	if !exists(filename) {
		return nil, fmt.Errorf("vidio: video file %s does not exist", filename)
	}

	if err := installed("ffmpeg"); err != nil {
		return nil, err
	}

	if err := installed("ffprobe"); err != nil {
		return nil, err
	}

	selectExpression, err := buildSelectExpression(n)
	if err != nil {
		return nil, fmt.Errorf("vidio: failed to parse the specified frame index: %w", err)
	}

	imageRect, stream, err := probeVideo(filename)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i", filename,
		"-f", "image2pipe",
		"-loglevel", "quiet",
		"-pix_fmt", "rgba",
		"-vcodec", "rawvideo",
		"-map", fmt.Sprintf("0:v:%d", stream),
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

	imageBuffer := image.NewRGBA(imageRect)
	if _, err := io.ReadFull(stdoutPipe, imageBuffer.Pix); err != nil {
		return nil, fmt.Errorf("vidio: failed to read the ffmpeg cmd result to the image buffer: %w", err)
	}

	if err := stdoutPipe.Close(); err != nil {
		return nil, fmt.Errorf("vidio: failed to close the ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("vidio: failed to free resources after the ffmpeg cmd: %w", err)
	}

	return imageBuffer, nil
}

// Helper function used to extract the target frame size and stream index
func probeVideo(filename string) (image.Rectangle, int, error) {
	videoData, err := ffprobe(filename, "v")
	if err != nil {
		return image.Rectangle{}, 0, fmt.Errorf("vidio: no video data found in %s: %w", filename, err)
	}

	if len(videoData) == 0 {
		return image.Rectangle{}, 0, fmt.Errorf("vidio: no video streams found in %s", filename)
	}

	var (
		width  int = 0
		height int = 0
	)

	if widthStr, ok := videoData[0]["width"]; !ok {
		return image.Rectangle{}, 0, errors.New("vidio: failed to access the image width")
	} else {
		if widthParsed, err := strconv.Atoi(widthStr); err != nil {
			return image.Rectangle{}, 0, fmt.Errorf("vidio: failed to parse the image width: %w", err)
		} else {
			width = widthParsed
		}
	}

	if heightStr, ok := videoData[0]["height"]; !ok {
		return image.Rectangle{}, 0, errors.New("vidio: failed to access the image height")
	} else {
		if heightParsed, err := strconv.Atoi(heightStr); err != nil {
			return image.Rectangle{}, 0, fmt.Errorf("vidio: failed to parse the image height: %w", err)
		} else {
			height = heightParsed
		}
	}

	return image.Rect(0, 0, width, height), 0, nil
}

// Error representing a strings.Builder failure in the buildSelectExpression func.
var errExpressionBuilder = errors.New("vidio: failed to write tokens to the frame select expresion")

// Helper function used to generate a "-vf select" expression that specifies which video frames should be exported.
func buildSelectExpression(n ...int) (string, error) {
	sb := strings.Builder{}
	if _, err := sb.WriteString("select='"); err != nil {
		return "", errExpressionBuilder
	}

	for index, frame := range n {
		if index != 0 {
			if _, err := sb.WriteRune('+'); err != nil {
				return "", errExpressionBuilder
			}
		}

		if _, err := sb.WriteString("eq(n\\,"); err != nil {

			return "", errExpressionBuilder
		}

		if _, err := sb.WriteString(strconv.Itoa(frame)); err != nil {

			return "", errExpressionBuilder
		}

		if _, err := sb.WriteRune(')'); err != nil {

			return "", errExpressionBuilder
		}
	}

	if _, err := sb.WriteRune('\''); err != nil {

		return "", errExpressionBuilder
	}

	return sb.String(), nil
}
