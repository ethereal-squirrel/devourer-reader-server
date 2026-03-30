package scanner

import (
	"io"
	"sort"

	"github.com/bodgit/sevenzip"
)

// Process7z processes a .7z or .cb7 archive and returns the page count and
// the first image's raw bytes (used as the cover/preview).
func Process7z(archivePath string) (pageCount int, firstImageData []byte, err error) {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return 0, nil, err
	}
	defer r.Close()

	var images []*sevenzip.File
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
	firstImageData, err = io.ReadAll(io.LimitReader(rc, 10<<20))
	if err != nil {
		return pageCount, nil, nil
	}
	return pageCount, firstImageData, nil
}
