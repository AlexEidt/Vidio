package vidio

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"image/jpeg"
	"image/png"
)

// Reads an image into an rgba byte buffer from a file. Currently only supports png and jpeg.
func Read(filename string, buffer ...[]byte) (int, int, []byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, 0, nil, err
	}
	defer f.Close()

	image, _, err := image.Decode(f)
	if err != nil {
		return 0, 0, nil, err
	}

	bounds := image.Bounds().Max
	size := bounds.X * bounds.Y * 4

	var data []byte
	if len(buffer) > 0 {
		if len(buffer[0]) < size {
			return 0, 0, nil, fmt.Errorf("vidio: buffer size (%d) is smaller than image size (%d)", len(buffer[0]), size)
		}
		data = buffer[0]
	} else {
		data = make([]byte, size)
	}

	index := 0
	for h := 0; h < bounds.Y; h++ {
		for w := 0; w < bounds.X; w++ {
			r, g, b, _ := image.At(w, h).RGBA()
			r, g, b = r/256, g/256, b/256
			data[index+0] = byte(r)
			data[index+1] = byte(g)
			data[index+2] = byte(b)
			data[index+3] = 255
			index += 4
		}
	}
	return bounds.X, bounds.Y, data, nil
}

// Writes a rgba byte buffer to a file. Currently only supports png and jpeg.
func Write(filename string, width, height int, buffer []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	image := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(image.Pix, buffer)

	switch filepath.Ext(filename) {
	case ".png":
		return png.Encode(f, image)
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, image, nil)
	default:
		return fmt.Errorf("vidio: unsupported file extension: %s", filepath.Ext(filename))
	}
}
