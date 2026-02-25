package textures

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sync"

	"render-engine/vulkan"
)

// Texture represents a GPU texture with its Vulkan resources
type Texture struct {
	Name   string
	Upload *vulkan.TextureUploadResult
	Width  uint32
	Height uint32
	Path   string // empty for procedural textures
}

// TextureManager manages loaded textures with caching
type TextureManager struct {
	textures map[string]*Texture
	mu       sync.RWMutex
	device   *vulkan.Device
}

// NewTextureManager creates a new texture manager
func NewTextureManager(device *vulkan.Device) *TextureManager {
	return &TextureManager{
		textures: make(map[string]*Texture),
		device:   device,
	}
}

// LoadTexture loads a texture from file, returning cached version if available
func (tm *TextureManager) LoadTexture(path string) (*Texture, error) {
	tm.mu.RLock()
	if tex, ok := tm.textures[path]; ok {
		tm.mu.RUnlock()
		return tex, nil
	}
	tm.mu.RUnlock()

	pixels, width, height, err := loadImageFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load image %s: %w", path, err)
	}

	tex, err := CreateTextureFromPixels(tm.device, path, width, height, pixels)
	if err != nil {
		return nil, err
	}
	tex.Path = path

	tm.mu.Lock()
	tm.textures[path] = tex
	tm.mu.Unlock()

	return tex, nil
}

// GetOrDefault returns the texture at path, or the default white texture
func (tm *TextureManager) GetOrDefault(path string) *Texture {
	if path == "" {
		return tm.GetDefaultTexture()
	}
	tex, err := tm.LoadTexture(path)
	if err != nil {
		fmt.Printf("Warning: failed to load texture %s: %v\n", path, err)
		return tm.GetDefaultTexture()
	}
	return tex
}

// GetDefaultTexture returns a 1x1 white texture
func (tm *TextureManager) GetDefaultTexture() *Texture {
	const key = "__default_white__"
	tm.mu.RLock()
	if tex, ok := tm.textures[key]; ok {
		tm.mu.RUnlock()
		return tex
	}
	tm.mu.RUnlock()

	pixels := []byte{255, 255, 255, 255}
	tex, err := CreateTextureFromPixels(tm.device, key, 1, 1, pixels)
	if err != nil {
		fmt.Printf("Error creating default texture: %v\n", err)
		return nil
	}

	tm.mu.Lock()
	tm.textures[key] = tex
	tm.mu.Unlock()

	return tex
}

// DestroyAll cleans up all loaded textures
func (tm *TextureManager) DestroyAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, tex := range tm.textures {
		tex.Destroy(tm.device)
	}
	tm.textures = make(map[string]*Texture)
}

// CreateTextureFromPixels creates a GPU texture from raw RGBA pixel data
func CreateTextureFromPixels(device *vulkan.Device, name string, width, height uint32, pixels []byte) (*Texture, error) {
	upload, err := vulkan.UploadTextureData(device, width, height, pixels)
	if err != nil {
		return nil, err
	}

	return &Texture{
		Name:   name,
		Upload: upload,
		Width:  width,
		Height: height,
	}, nil
}

// Destroy releases the texture's GPU resources
func (t *Texture) Destroy(device *vulkan.Device) {
	if t.Upload != nil {
		t.Upload.Destroy(device)
	}
}

// loadImageFile reads an image file and returns RGBA pixel data
func loadImageFile(path string) ([]byte, uint32, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}

	bounds := img.Bounds()
	width := uint32(bounds.Dx())
	height := uint32(bounds.Dy())
	pixels := make([]byte, width*height*4)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			idx := ((y-bounds.Min.Y)*int(width) + (x - bounds.Min.X)) * 4
			pixels[idx] = uint8(r >> 8)
			pixels[idx+1] = uint8(g >> 8)
			pixels[idx+2] = uint8(b >> 8)
			pixels[idx+3] = uint8(a >> 8)
		}
	}

	return pixels, width, height, nil
}

// CreateSolidColorTexture creates a 1x1 solid color texture
func CreateSolidColorTexture(device *vulkan.Device, name string, r, g, b, a uint8) (*Texture, error) {
	pixels := []byte{r, g, b, a}
	return CreateTextureFromPixels(device, name, 1, 1, pixels)
}

// CreateCheckerTexture creates a checkerboard pattern texture
func CreateCheckerTexture(device *vulkan.Device, name string, size uint32, c1, c2 color.RGBA) (*Texture, error) {
	pixels := make([]byte, size*size*4)
	blockSize := size / 8
	if blockSize < 1 {
		blockSize = 1
	}

	for y := uint32(0); y < size; y++ {
		for x := uint32(0); x < size; x++ {
			idx := (y*size + x) * 4
			isWhite := ((x/blockSize)+(y/blockSize))%2 == 0
			if isWhite {
				pixels[idx] = c1.R
				pixels[idx+1] = c1.G
				pixels[idx+2] = c1.B
				pixels[idx+3] = c1.A
			} else {
				pixels[idx] = c2.R
				pixels[idx+1] = c2.G
				pixels[idx+2] = c2.B
				pixels[idx+3] = c2.A
			}
		}
	}

	return CreateTextureFromPixels(device, name, size, size, pixels)
}
