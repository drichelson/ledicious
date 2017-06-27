package usb

import (
	"fmt"
	"github.com/lucasb-eyer/go-colorful"
	"testing"
)

func TestNormalizeBrightness(t *testing.T) {
	var r, g, b uint8

	r, g, b = normalizeBrightness(colorful.Color{R: 0.0, G: 0.0, B: 0.0})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

	r, g, b = normalizeBrightness(colorful.Color{R: 0.01, G: 0.01, B: 0.01})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

	r, g, b = normalizeBrightness(colorful.Color{R: 0.05, G: 0.05, B: 0.05})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

	r, g, b = normalizeBrightness(colorful.Color{R: 0.1, G: 0.1, B: 0.1})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

	r, g, b = normalizeBrightness(colorful.Color{R: 0.5, G: 0.5, B: 0.5})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

	r, g, b = normalizeBrightness(colorful.Color{R: 0.9, G: 0.9, B: 0.9})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

	r, g, b = normalizeBrightness(colorful.Color{R: 1.0, G: 1.0, B: 1.0})
	fmt.Printf("r: %v g: %v b: %v\n", r, g, b)

}
