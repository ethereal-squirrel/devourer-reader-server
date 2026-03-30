package image

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/sunshineplan/imgconv"
)

const (
	CoverMaxWidth   = 600
	PreviewMaxWidth = 512
	JPEGQuality     = 85
)

func ResizeAndSave(srcData []byte, destPath string, maxWidth int) error {
	img, err := decodeAny(srcData)
	if err != nil {
		return err
	}

	img = resizeWidth(img, maxWidth)

	jpegData, err := encodeJPEG(img)
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, jpegData, 0o644)
}

func ResizeAndSaveFile(srcPath, destPath string, maxWidth int) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return ResizeAndSave(data, destPath, maxWidth)
}

func EnsureDir(destPath string) error {
	return os.MkdirAll(filepath.Dir(destPath), 0o755)
}

func decodeAny(data []byte) (image.Image, error) {
	return imgconv.Decode(bytes.NewReader(data))
}

func resizeWidth(img image.Image, maxWidth int) image.Image {
	w := img.Bounds().Dx()
	if w <= maxWidth {
		return img
	}
	return imgconv.Resize(img, &imgconv.ResizeOption{Width: maxWidth})
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality})
	return buf.Bytes(), err
}
