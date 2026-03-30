package scanner

import (
	"io"
	"os"
	"sort"

	"github.com/nwaples/rardecode"
)

func ProcessRar(archivePath string) (pageCount int, firstImageData []byte, err error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	reader, err := rardecode.NewReader(f, "")
	if err != nil {
		return 0, nil, err
	}

	type imgEntry struct {
		name string
		data []byte
	}
	var images []imgEntry

	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, nil, err
		}
		if hdr.IsDir || !isImageFile(hdr.Name) {
			io.Copy(io.Discard, reader)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(reader, 10<<20))
		if err != nil {
			return 0, nil, err
		}
		images = append(images, imgEntry{name: hdr.Name, data: data})
	}

	if len(images) == 0 {
		return 0, nil, nil
	}

	sort.Slice(images, func(i, j int) bool { return images[i].name < images[j].name })

	return len(images), images[0].data, nil
}
