package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math/rand"
	"net/http"

	"github.com/nsmith5/mjpeg"
)

const (
	wight  = 512
	height = 512
	size   = 64
)

func main() {
	stream := mjpeg.Handler{
		Next: func() (image.Image, error) {
			img := image.NewGray(image.Rect(0, 0, wight, height))
			for i := 0; i < wight; i += size {
				for j := 0; j < height; j += size {
					n := rand.Intn(256)
					gray := color.Gray{uint8(n)}
					for x := 0; x < size; x++ {
						for y := 0; y < size; y++ {
							img.SetGray(i+x, j+y, gray)
						}
					}
				}
			}
			return img, nil
		},
		Options: &jpeg.Options{Quality: 80},
	}

	mux := http.NewServeMux()
	mux.Handle("/stream", stream)
	log.Fatal(http.ListenAndServe(":45777", mux))
}
