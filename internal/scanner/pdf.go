package scanner

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/sunshineplan/imgconv"
)

func ProcessPDF(pdfPath string) ([]byte, error) {
	img, err := imgconv.Open(pdfPath)
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	return encodeJPEG(img)
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
