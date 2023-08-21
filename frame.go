package vidio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// Return the N-th frame of the video specified by the filename in the RGBA format and stored it to the provided frame buffer. The frames are indexed from 0.
func GetVideoFrame(filename string, n int, frameBuffer []byte) error {
	if !exists(filename) {
		return fmt.Errorf("vidio: video file %s does not exist", filename)
	}

	if err := installed("ffmpeg"); err != nil {
		return err
	}

	if err := installed("ffprobe"); err != nil {
		return err
	}

	frameBufferSize, framesCount, err := probeVideo(filename)
	if err != nil {
		return err
	}

	if n >= framesCount {
		return errors.New("vidio: provided frame index is not in frame count range")
	}

	if frameBuffer == nil {
		return errors.New("vidio: provided frame buffer reference is nil")
	}

	if len(frameBuffer) < frameBufferSize {
		return errors.New("vidio: provided frame buffer size is smaller than the frame size")
	}

	selectExpression, err := buildSelectExpression(n)
	if err != nil {
		return fmt.Errorf("vidio: failed to parse the specified frame index: %w", err)
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i", filename,
		"-f", "image2pipe",
		"-loglevel", "quiet",
		"-pix_fmt", "rgba",
		"-vcodec", "rawvideo",
		"-map", "0:v:0",
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

	if _, err := io.ReadFull(stdoutPipe, frameBuffer); err != nil {
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

// Helper function used to extract the target frame buffer size and frames count
func probeVideo(filename string) (int, int, error) {
	videoData, err := ffprobe(filename, "v")
	if err != nil {
		return 0, 0, fmt.Errorf("vidio: no video data found in %s: %w", filename, err)
	}

	if len(videoData) == 0 {
		return 0, 0, fmt.Errorf("vidio: no video streams found in %s", filename)
	}

	var (
		width  int = 0
		height int = 0
		frames int = 0
	)

	if widthStr, ok := videoData[0]["width"]; !ok {
		return 0, 0, errors.New("vidio: failed to access the image width")
	} else {
		if widthParsed, err := strconv.Atoi(widthStr); err != nil {
			return 0, 0, fmt.Errorf("vidio: failed to parse the image width: %w", err)
		} else {
			width = widthParsed
		}
	}

	if heightStr, ok := videoData[0]["height"]; !ok {
		return 0, 0, errors.New("vidio: failed to access the image height")
	} else {
		if heightParsed, err := strconv.Atoi(heightStr); err != nil {
			return 0, 0, fmt.Errorf("vidio: failed to parse the image height: %w", err)
		} else {
			height = heightParsed
		}
	}

	if framesStr, ok := videoData[0]["nb_frames"]; !ok {
		return 0, 0, errors.New("vidio: failed to access the frames count")
	} else {
		if framesParsed, err := strconv.Atoi(framesStr); err != nil {
			return 0, 0, fmt.Errorf("vidio: failed to parse the frames count: %w", err)
		} else {
			frames = framesParsed
		}
	}

	frameBufferSize := width * height * 4
	return frameBufferSize, frames, nil
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
