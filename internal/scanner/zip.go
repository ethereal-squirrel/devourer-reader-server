package scanner

import (
	"archive/zip"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".tiff": true, ".tif": true,
	".avif": true,
}

func isImageFile(name string) bool {
	return imageExts[strings.ToLower(filepath.Ext(name))]
}

func ProcessZip(archivePath string) (pageCount int, firstImageData []byte, err error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return 0, nil, err
	}
	defer r.Close()

	var images []*zip.File
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && isImageFile(f.Name) {
			images = append(images, f)
		}
	}
	if len(images) == 0 {
		return 0, nil, nil
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Name < images[j].Name
	})

	pageCount = len(images)

	rc, err := images[0].Open()
	if err != nil {
		return pageCount, nil, nil
	}
	defer rc.Close()
	firstImageData, err = io.ReadAll(rc)
	if err != nil {
		return pageCount, nil, nil
	}
	return pageCount, firstImageData, nil
}
