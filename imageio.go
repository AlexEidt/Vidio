package vidio

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
)

// Reads an image from a file. Currently only supports png and jpeg.
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
	size := bounds.X * bounds.Y * 3

	var data []byte
	if len(buffer) > 0 {
		if len(buffer[0]) < size {
			errmsg := fmt.Sprintf("Buffer size (%d) is smaller than image size (%d)", len(buffer[0]), size)
			return 0, 0, nil, errors.New(errmsg)
		}
		data = buffer[0]
	} else {
		data = make([]byte, size, size)
	}

	index := 0
	for h := 0; h < bounds.Y; h++ {
		for w := 0; w < bounds.X; w++ {
			r, g, b, _ := image.At(w, h).RGBA()
			r, g, b = r/256, g/256, b/256
			data[index] = byte(r)
			index++
			data[index] = byte(g)
			index++
			data[index] = byte(b)
			index++
		}
	}
	return bounds.X, bounds.Y, data, nil
}

// Writes an image to a file. Currently only supports png and jpeg.
func Write(filename string, width, height int, buffer []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	image := image.NewRGBA(image.Rect(0, 0, width, height))
	index := 0
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			r, g, b := buffer[index], buffer[index+1], buffer[index+2]
			image.Set(w, h, color.RGBA{r, g, b, 255})
			index += 3
		}
	}

	switch filepath.Ext(filename) {
	case ".png":
		if err := png.Encode(f, image); err != nil {
			return err
		}
	case ".jpg", ".jpeg":
		if err := jpeg.Encode(f, image, nil); err != nil {
			return err
		}
	}

	return nil
}
