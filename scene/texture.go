package scene

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Texture holds CPU-side pixel data for a 2D texture.
// GLID is set by the OpenGL backend after upload; do not access directly.
type Texture struct {
	Name   string
	Width  int
	Height int
	// Pixels in RGBA8 format (4 bytes per pixel, row-major, top-to-bottom).
	Pixels []byte
	// GLID is the OpenGL texture object ID, set by opengl.UploadTexture.
	GLID uint32
}

// LoadTexture reads a PNG or JPEG file from disk and returns a CPU-side Texture.
// The image is converted to RGBA8 automatically.
func LoadTexture(path string) (*Texture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open texture %q: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode texture %q: %w", path, err)
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Convert to RGBA8
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return &Texture{
		Name:   path,
		Width:  w,
		Height: h,
		Pixels: rgba.Pix,
	}, nil
}

// NewSolidTexture creates a 1x1 texture with the given RGBA color values (0–255).
func NewSolidTexture(name string, r, g, b, a uint8) *Texture {
	return &Texture{
		Name:   name,
		Width:  1,
		Height: 1,
		Pixels: []byte{r, g, b, a},
	}
}
