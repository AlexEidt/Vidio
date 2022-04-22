package vidio

import (
	"image"
	"os"
	"strings"

	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
)

// Reads an image from a file. Currently only supports png and jpeg.
func Read(filename string) (int, int, []byte, error) {
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

	data := make([]byte, size)

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
func Write(filename string, width, height int, data []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	image := image.NewRGBA(image.Rect(0, 0, width, height))
	index := 0
	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			r, g, b := data[index], data[index+1], data[index+2]
			image.Set(w, h, color.RGBA{r, g, b, 255})
			index += 3
		}
	}

	if strings.HasSuffix(filename, ".png") {
		if err := png.Encode(f, image); err != nil {
			return err
		}
	} else if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		if err := jpeg.Encode(f, image, nil); err != nil {
			return err
		}
	}
	return nil
}
