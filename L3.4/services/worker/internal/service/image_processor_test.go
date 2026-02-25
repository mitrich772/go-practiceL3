package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"contracts/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	require.NoError(t, jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}))
	return buf.Bytes()
}

func createTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 100, B: 150, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	require.NoError(t, png.Encode(buf, img))
	return buf.Bytes()
}

func decodeResult(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	return img
}

func TestProcess_Resize_Success(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 400, 300)

	job := dto.ImageJob{
		ID:           "test-1",
		OriginalPath: "originals/test-1.jpg",
		Mode:         "resize",
		Width:        100,
		Height:       0,
	}

	out, ext, err := proc.Process(context.Background(), origBytes, job)
	require.NoError(t, err)
	assert.Equal(t, ".jpg", ext)
	assert.Equal(t, 100, decodeResult(t, out).Bounds().Dx())
}

func TestProcess_Resize_BothZero(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 200, 200)

	job := dto.ImageJob{
		ID:           "test-2",
		OriginalPath: "originals/test-2.jpg",
		Mode:         "resize",
		Width:        0,
		Height:       0,
	}

	out, ext, err := proc.Process(context.Background(), origBytes, job)
	require.NoError(t, err)
	assert.Equal(t, ".jpg", ext)
	assert.Equal(t, origBytes, out)
}

func TestProcess_Thumb_Success(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 800, 600)

	job := dto.ImageJob{
		ID:           "test-3",
		OriginalPath: "originals/test-3.jpg",
		Mode:         "thumb",
		Width:        150,
		Height:       150,
	}

	out, ext, err := proc.Process(context.Background(), origBytes, job)
	require.NoError(t, err)
	assert.Equal(t, ".jpg", ext)

	img := decodeResult(t, out)
	assert.Equal(t, 150, img.Bounds().Dx())
	assert.Equal(t, 150, img.Bounds().Dy())
}

func TestProcess_Thumb_ZeroSize(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 200, 200)

	job := dto.ImageJob{
		ID:           "test-4",
		OriginalPath: "originals/test-4.jpg",
		Mode:         "thumb",
		Width:        0,
		Height:       100,
	}

	_, _, err := proc.Process(context.Background(), origBytes, job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "width and height must be > 0")
}

func TestProcess_Watermark_Success(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 300, 300)

	job := dto.ImageJob{
		ID:            "test-5",
		OriginalPath:  "originals/test-5.jpg",
		Mode:          "watermark",
		WatermarkText: "TEST",
	}

	out, ext, err := proc.Process(context.Background(), origBytes, job)
	require.NoError(t, err)
	assert.Equal(t, ".jpg", ext)
	assert.NotEqual(t, origBytes, out)
}

func TestProcess_Watermark_EmptyText(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 200, 200)

	job := dto.ImageJob{
		ID:            "test-6",
		OriginalPath:  "originals/test-6.jpg",
		Mode:          "watermark",
		WatermarkText: "",
	}

	out, ext, err := proc.Process(context.Background(), origBytes, job)
	require.NoError(t, err)
	assert.Equal(t, ".jpg", ext)
	assert.NotEmpty(t, out)
}

func TestProcess_UnsupportedMode(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestJPEG(t, 100, 100)

	job := dto.ImageJob{
		ID:           "test-7",
		OriginalPath: "originals/test-7.jpg",
		Mode:         "blur",
	}

	_, _, err := proc.Process(context.Background(), origBytes, job)
	require.ErrorIs(t, err, ErrUnsupportedMode)
}

func TestProcess_InvalidBytes(t *testing.T) {
	proc := NewImageProcessor(nil)

	job := dto.ImageJob{
		ID:           "test-8",
		OriginalPath: "originals/test-8.jpg",
		Mode:         "resize",
		Width:        100,
	}

	_, _, err := proc.Process(context.Background(), []byte("not an image"), job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func TestProcess_PNG_Format(t *testing.T) {
	proc := NewImageProcessor(nil)
	origBytes := createTestPNG(t, 200, 200)

	job := dto.ImageJob{
		ID:           "test-9",
		OriginalPath: "originals/test-9.png",
		Mode:         "resize",
		Width:        50,
	}

	out, ext, err := proc.Process(context.Background(), origBytes, job)
	require.NoError(t, err)
	assert.Equal(t, ".png", ext)
	assert.Equal(t, 50, decodeResult(t, out).Bounds().Dx())
}
