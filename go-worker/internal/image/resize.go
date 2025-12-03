package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

func ResizeToMinDimension(imagePath string, targetSize int) (string, error) {
	// Open the original image
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Printf("Original image size: %dx%d (format: %s)\n", width, height, format)

	// Calculate new dimensions
	var newWidth, newHeight int
	if width < height {
		// Width is shorter
		newWidth = targetSize
		newHeight = int(float64(height) * float64(targetSize) / float64(width))
	} else {
		// Height is shorter (or equal)
		newHeight = targetSize
		newWidth = int(float64(width) * float64(targetSize) / float64(height))
	}

	// Skip resize if image is already small enough
	if (width <= newWidth && height <= newHeight) || (width < targetSize && height < targetSize) {
		fmt.Printf("Image is already small enough (%dx%d), skipping resize\n", width, height)
		return imagePath, nil
	}

	fmt.Printf("Resizing to: %dx%d\n", newWidth, newHeight)

	// Create new image with target dimensions
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Use high-quality bilinear interpolation
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// Create temporary file for resized image
	tmpFile, err := os.CreateTemp("", "resized-*.jpg")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Encode based on original format, but prefer JPEG for compatibility
	ext := strings.ToLower(filepath.Ext(imagePath))
	if ext == ".png" {
		err = png.Encode(tmpFile, dst)
	} else {
		// Use JPEG for all other formats (jpg, jpeg, webp, etc.)
		err = jpeg.Encode(tmpFile, dst, &jpeg.Options{Quality: 90})
	}

	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to encode resized image: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	fmt.Printf("Resized image saved to: %s\n", tmpPath)
	return tmpPath, nil
}
