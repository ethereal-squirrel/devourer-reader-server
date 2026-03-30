package scanner

import (
	"fmt"
	"os"
	"path/filepath"
)

func listTopLevel(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.Name() != ".devourer" {
			names = append(names, e.Name())
		}
	}
	return names
}

func getAllFiles(dir string) []string {
	var files []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func removeCoverDir(metaBase, subdir string, id int64) {
	target := filepath.Join(metaBase, subdir, fmt.Sprintf("%d", id))
	os.RemoveAll(target)
}

func removePreview(metaBase string, seriesID int64, fileName string) {
	target := filepath.Join(metaBase, "series",
		fmt.Sprintf("%d", seriesID), "previews", fileName)
	os.Remove(target)
}
