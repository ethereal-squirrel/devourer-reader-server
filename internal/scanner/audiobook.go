package scanner

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	imgconvert "github.com/devourer/server/internal/image"
	"github.com/devourer/server/internal/metadata"
)

var validAudioExts = map[string]bool{
	".mp3":  true,
	".m4a":  true,
	".m4b":  true,
	".ogg":  true,
	".flac": true,
	".aac":  true,
	".opus": true,
	".wav":  true,
}

func isAudioFile(path string) bool {
	return validAudioExts[strings.ToLower(filepath.Ext(path))]
}

func findAudiobookFolders(root string) []string {
	var folders []string

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if d.Name() == ".devourer" {
			return filepath.SkipDir
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil
		}
		for _, e := range entries {
			if !e.IsDir() && isAudioFile(e.Name()) {
				folders = append(folders, path)
				break
			}
		}
		return nil
	})

	sort.Strings(folders)
	return folders
}

func ScanAudiobookLibrary(cfg *Config, lib *db.Library, _ []string) {
	audiobookFolders := findAudiobookFolders(lib.Path)

	folderNames := make([]string, len(audiobookFolders))
	for i, p := range audiobookFolders {
		folderNames[i] = filepath.Base(p)
	}

	status := &ScanStatus{
		InProgress:  true,
		LibraryType: lib.Type,
		TotalSeries: len(audiobookFolders),
	}
	for _, name := range folderNames {
		status.Series = append(status.Series, ScanProgress{
			Series:      name,
			LibraryType: lib.Type,
			Status:      "scanning",
		})
	}
	setScanStatus(lib.ID, status)

	for i, folderPath := range audiobookFolders {
		name := folderNames[i]
		updateSeriesStatus(lib.ID, name, "scanning")
		if err := ProcessAudiobook(cfg, lib, folderPath); err != nil {
			log.Printf("[Scanner] Error processing audiobook %s: %v", name, err)
			updateSeriesError(lib.ID, name, err.Error())
		} else {
			updateSeriesStatus(lib.ID, name, "complete")
		}
	}

	allSeries, _ := queries.ListAudiobookSeriesByLibrary(cfg.DB, lib.ID)
	for _, s := range allSeries {
		if _, err := os.Stat(s.Path); os.IsNotExist(err) {
			log.Printf("[Scanner] Removing stale audiobook series: %s", s.Title)
			queries.DeleteAudiobookFilesBySeries(cfg.DB, s.ID)
			queries.DeleteAudiobookSeries(cfg.DB, s.ID)
		}
	}
	log.Printf("[Scanner] Audiobook scan complete for library %d", lib.ID)
}

func ProcessAudiobook(cfg *Config, lib *db.Library, folderPath string) error {
	folderName := filepath.Base(folderPath)

	series, err := queries.GetAudiobookSeriesByPath(cfg.DB, folderPath)
	if err != nil {
		audiobookDataJSON := fetchAudiobookMetadata(lib, folderName)
		totalDuration := durationSecondsFromMetadata(audiobookDataJSON)
		series, err = queries.CreateAudiobookSeries(cfg.DB, &db.AudiobookSeries{
			Title:                folderName,
			Path:                 folderPath,
			LibraryID:            lib.ID,
			AudiobookData:        audiobookDataJSON,
			TotalDurationSeconds: totalDuration,
		})
		if err != nil {
			return fmt.Errorf("create audiobook series: %w", err)
		}

		coverDir := filepath.Join(lib.MetaBase(), "series", fmt.Sprintf("%d", series.ID))
		os.MkdirAll(coverDir, 0o755)

		if len(audiobookDataJSON) > 0 {
			var meta map[string]any
			if json.Unmarshal(audiobookDataJSON, &meta) == nil {
				if coverURL, ok := meta["coverImage"].(string); ok && coverURL != "" {
					coverPath := filepath.Join(coverDir, "cover.jpg")
					imgconvert.DownloadAndSave(coverURL, coverPath, imgconvert.CoverMaxWidth)
				}
			}
		}
	}

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("read audiobook folder: %w", err)
	}
	var audioFiles []string
	for _, e := range entries {
		if !e.IsDir() {
			fullPath := filepath.Join(folderPath, e.Name())
			if isAudioFile(fullPath) {
				audioFiles = append(audioFiles, fullPath)
			}
		}
	}

	if len(audioFiles) == 0 {
		log.Printf("[Scanner] No audio files in: %s", folderPath)
		return nil
	}

	existing, _ := queries.ListAudiobookFilesPathBySeries(cfg.DB, series.ID)
	existingPaths := make(map[string]int64, len(existing))
	for _, af := range existing {
		existingPaths[af.Path] = af.ID
	}

	coverPath := filepath.Join(lib.MetaBase(), "series", fmt.Sprintf("%d", series.ID), "cover.jpg")
	_, coverExists := os.Stat(coverPath)
	coverSaved := coverExists == nil

	var totalNewDuration int

	for _, filePath := range audioFiles {
		if _, exists := existingPaths[filePath]; exists {
			delete(existingPaths, filePath)
			continue
		}

		baseName := filepath.Base(filePath)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))

		tags, _ := ReadAudioTags(filePath)

		trackNum := 0
		durationSecs := 0
		fileMeta := map[string]any{}

		if tags != nil {
			trackNum = tags.TrackNumber
			durationSecs = tags.DurationSeconds
			if tags.Title != "" {
				fileMeta["title"] = tags.Title
			}
			if tags.Artist != "" {
				fileMeta["artist"] = tags.Artist
			}
			if tags.Album != "" {
				fileMeta["album"] = tags.Album
			}

			if !coverSaved && len(tags.CoverData) > 0 {
				if imgconvert.ResizeAndSave(tags.CoverData, coverPath, imgconvert.CoverMaxWidth) == nil {
					coverSaved = true
				}
			}
		}

		totalNewDuration += durationSecs

		fileMetaJSON, _ := json.Marshal(fileMeta)
		queries.CreateAudiobookFile(cfg.DB, &db.AudiobookFile{
			Path:            filePath,
			FileName:        baseName,
			FileFormat:      ext,
			TrackNumber:     trackNum,
			DurationSeconds: durationSecs,
			SeriesID:        series.ID,
			Metadata:        json.RawMessage(fileMetaJSON),
		})

		log.Printf("[Scanner] Added audio track: %s (track=%d dur=%ds)", baseName, trackNum, durationSecs)
	}

	if totalNewDuration > 0 {
		queries.UpdateAudiobookSeriesTotalDuration(cfg.DB, series.ID, series.TotalDurationSeconds+totalNewDuration)
	}

	for _, id := range existingPaths {
		queries.DeleteAudiobookFile(cfg.DB, id)
	}

	return nil
}

func DeleteAudiobook(d *sql.DB, libraryPath, filePath string) error {
	af, err := queries.GetAudiobookFileByPath(d, filePath)
	if err != nil {
		return nil
	}

	queries.DeleteRecentlyReadByFileID(d, af.ID)
	queries.DeleteReadingStatusByFileID(d, af.ID)
	queries.DeleteUserRatingsByFileID(d, af.ID)
	queries.DeleteUserTagsByFileID(d, af.ID)
	queries.DeleteAudiobookFile(d, af.ID)

	lib2, err2 := queries.GetLibraryByPath(d, libraryPath)
	metaBase2 := filepath.Join(libraryPath, ".devourer")
	if err2 == nil {
		metaBase2 = lib2.MetaBase()
	}

	remaining, _ := queries.ListAudiobookFilesBySeries(d, af.SeriesID)
	if len(remaining) == 0 {
		queries.DeleteAudiobookSeries(d, af.SeriesID)
		removeCoverDir(metaBase2, "series", af.SeriesID)
	}
	return nil
}

func durationSecondsFromMetadata(data json.RawMessage) int {
	if len(data) == 0 {
		return 0
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return 0
	}
	switch v := meta["runtime_minutes"].(type) {
	case float64:
		return int(v * 60)
	}
	return 0
}

func fetchAudiobookMetadata(lib *db.Library, title string) json.RawMessage {
	log.Printf("[Audible] fetchAudiobookMetadata called for title: %q", title)

	var region string
	if lib.Metadata != nil {
		var libMeta map[string]any
		if json.Unmarshal(lib.Metadata, &libMeta) == nil {
			if r, ok := libMeta["region"].(string); ok {
				region = r
			}
		}
	}

	if region == "" {
		region = "us"
	}

	log.Printf("[Audible] using region %q for title %q", region, title)

	log.Printf("[Audible] waiting on rate limiter for %q", title)
	metadata.AudibleLimiter.Wait()

	result, err := metadata.AudibleSearch(title, region)

	if err != nil {
		log.Printf("[Audible] search failed for %q: %v", title, err)
		return json.RawMessage(`{}`)
	}

	if result == nil {
		log.Printf("[Audible] no result for %q", title)
		return json.RawMessage(`{}`)
	}

	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("[Audible] failed to marshal result for %q: %v", title, err)
		return json.RawMessage(`{}`)
	}

	log.Printf("[Audible] result for %q: %s", title, string(data))

	return json.RawMessage(data)
}
