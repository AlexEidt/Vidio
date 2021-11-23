package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
)

type Image struct {
	width    int
	height   int
	channels int
	data     []byte
}

func ReadImage(filename string) *Image {
	// Read image from "filename".
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%s not found.", filename)
		return nil
	}
	defer file.Close()

	var im image.Image
	if strings.HasSuffix(filename, "jpg") {
		im, err = jpeg.Decode(file)
	} else if strings.HasSuffix(filename, "png") {
		im, err = png.Decode(file)
	} else {
		im, _, err = image.Decode(file)
	}

	if err != nil {
		fmt.Printf("%s is an invalid image format. Could not parse.\n", filename)
		return nil
	}
	bounds := im.Bounds().Max

	data := make([]byte, bounds.Y*bounds.X*4)
	// Fill in "data" with colors of the image.
	index := 0
	for y := 0; y < bounds.Y; y++ {
		for x := 0; x < bounds.X; x++ {
			r, g, b, a := im.At(x, y).RGBA()
			data[index] = byte(r)
			data[index+1] = byte(g)
			data[index+2] = byte(b)
			data[index+3] = byte(a)
			index += 4
		}
	}
	return &Image{width: bounds.X, height: bounds.Y, channels: 4, data: data}
}

func WriteImage(filename string, img *Image) {

}
