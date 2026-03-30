package scanner

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	imgconvert "github.com/devourer/server/internal/image"
	"github.com/devourer/server/internal/metadata"
)

var (
	volumePatterns  = []*regexp.Regexp{regexp.MustCompile(`(?i)v(?:ol(?:ume)?)?\.?\s*(\d+)`), regexp.MustCompile(`(?i)\(v(\d+)\)`)}
	chapterPatterns = []*regexp.Regexp{regexp.MustCompile(`(?i)ch(?:apter)?\.?\s*(\d+\.?\d*)`), regexp.MustCompile(`(?i)c(\d+\.?\d*)`)}
	bracketRe       = regexp.MustCompile(`[\[\(].*?[\]\)]`)
	numberRe        = regexp.MustCompile(`\d+`)
)

func ExtractChapterAndVolume(name string) (volume, chapter int) {
	for _, p := range volumePatterns {
		if m := p.FindStringSubmatch(name); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				volume = v
				break
			}
		}
	}
	for _, p := range chapterPatterns {
		if m := p.FindStringSubmatch(name); m != nil {
			if c, err := strconv.ParseFloat(m[1], 64); err == nil {
				chapter = int(c)
				break
			}
		}
	}
	if volume == 0 && chapter == 0 {
		clean := bracketRe.ReplaceAllString(name, "")
		if m := numberRe.FindString(clean); m != "" {
			if n, err := strconv.Atoi(m); err == nil {
				chapter = n
			}
		}
	}
	return
}

func ProcessManga(cfg *Config, lib *db.Library, folderName string) error {
	seriesPath := filepath.Join(lib.Path, folderName)

	series, err := queries.GetMangaSeriesByLibraryAndTitle(cfg.DB, lib.ID, folderName)
	if err != nil {
		mangaDataJSON := fetchMangaMetadata(cfg.Providers, lib, folderName)
		series, err = queries.CreateMangaSeries(cfg.DB, &db.MangaSeries{
			Title:     folderName,
			Path:      seriesPath,
			LibraryID: lib.ID,
			MangaData: mangaDataJSON,
		})
		if err != nil {
			return fmt.Errorf("create manga series: %w", err)
		}

		previewDir := filepath.Join(lib.MetaBase(), "series",
			fmt.Sprintf("%d", series.ID), "previews")
		os.MkdirAll(previewDir, 0o755)

		if len(mangaDataJSON) > 0 {
			var meta map[string]any
			if json.Unmarshal(mangaDataJSON, &meta) == nil {
				if coverURL, ok := meta["coverImage"].(string); ok && coverURL != "" {
					coverPath := filepath.Join(lib.MetaBase(), "series",
						fmt.Sprintf("%d", series.ID), "cover.jpg")
					imgconvert.DownloadAndSave(coverURL, coverPath, imgconvert.CoverMaxWidth)
				}
			}
		}
	}

	allFiles := getAllFiles(seriesPath)
	var archiveFiles []string
	for _, f := range allFiles {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".zip" || ext == ".cbz" || ext == ".rar" || ext == ".cbr" || ext == ".7z" || ext == ".cb7" {
			archiveFiles = append(archiveFiles, f)
		}
	}

	if len(archiveFiles) == 0 {
		log.Printf("[Scanner] No archive files in series: %s", folderName)
		return nil
	}

	existing, _ := queries.ListMangaFilesPathBySeries(cfg.DB, series.ID)
	existingPaths := make(map[string]int64, len(existing))
	for _, mf := range existing {
		existingPaths[mf.Path] = mf.ID
	}

	seriesCoverPath := filepath.Join(lib.MetaBase(), "series",
		fmt.Sprintf("%d", series.ID), "cover.jpg")
	previewDir := filepath.Join(lib.MetaBase(), "series",
		fmt.Sprintf("%d", series.ID), "previews")

	for _, f := range archiveFiles {
		if _, exists := existingPaths[f]; exists {
			delete(existingPaths, f)
			continue
		}

		baseName := filepath.Base(f)
		vol, ch := ExtractChapterAndVolume(baseName)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f), "."))

		previewPath := filepath.Join(previewDir, baseName+".jpg")

		var pageCount int
		var firstImage []byte

		switch ext {
		case "cbz", "zip":
			pageCount, firstImage, _ = ProcessZip(f)
		case "cbr", "rar":
			pageCount, firstImage, _ = ProcessRar(f)
		case "7z", "cb7":
			pageCount, firstImage, _ = Process7z(f)
		}

		if len(firstImage) > 0 {
			imgconvert.ResizeAndSave(firstImage, previewPath, imgconvert.PreviewMaxWidth)

			if _, err := os.Stat(seriesCoverPath); os.IsNotExist(err) {
				imgconvert.ResizeAndSave(firstImage, seriesCoverPath, imgconvert.CoverMaxWidth)
			}
		}

		mfMeta, _ := json.Marshal(map[string]any{})
		queries.CreateMangaFile(cfg.DB, &db.MangaFile{
			Path:       f,
			FileName:   baseName,
			FileFormat: ext,
			Volume:     vol,
			Chapter:    ch,
			TotalPages: pageCount,
			SeriesID:   series.ID,
			Metadata:   json.RawMessage(mfMeta),
		})

		log.Printf("[Scanner] Added manga file: %s (vol=%d ch=%d pages=%d)", baseName, vol, ch, pageCount)
	}

	for _, id := range existingPaths {
		queries.DeleteMangaFile(cfg.DB, id)
	}

	return nil
}

func ScanMangaLibrary(cfg *Config, lib *db.Library, topLevel []string) {
	for _, entry := range topLevel {
		updateSeriesStatus(lib.ID, entry, "scanning")
		if err := ProcessManga(cfg, lib, entry); err != nil {
			log.Printf("[Scanner] Error processing manga %s: %v", entry, err)
			updateSeriesError(lib.ID, entry, err.Error())
		} else {
			updateSeriesStatus(lib.ID, entry, "complete")
		}
	}

	allSeries, _ := queries.ListMangaSeriesByLibrary(cfg.DB, lib.ID)
	for _, s := range allSeries {
		if _, err := os.Stat(s.Path); os.IsNotExist(err) {
			log.Printf("[Scanner] Removing stale series: %s", s.Title)
			queries.DeleteMangaFilesBySeries(cfg.DB, s.ID)
			queries.DeleteMangaSeries(cfg.DB, s.ID)
		}
	}
	log.Printf("[Scanner] Manga scan complete for library %d", lib.ID)
}

func fetchMangaMetadata(providers map[string]*metadata.Provider, lib *db.Library, title string) json.RawMessage {
	if providers == nil {
		return json.RawMessage(`{}`)
	}

	var providerKey, apiKey string
	if lib.Metadata != nil {
		var libMeta map[string]any
		if json.Unmarshal(lib.Metadata, &libMeta) == nil {
			if p, ok := libMeta["provider"].(string); ok {
				providerKey = p
			}
			if k, ok := libMeta["api_key"].(string); ok {
				apiKey = k
			}
		}
	}
	if providerKey == "" {
		providerKey = "jikan"
	}

	var limiter *metadata.RateLimiter
	switch providerKey {
	case "comicvine":
		limiter = metadata.ComicVineLimiter
	default:
		limiter = metadata.JikanLimiter
	}

	limiter.Wait()
	result, err := metadata.Search(providers, providerKey, "title", title, apiKey)
	if err != nil {
		log.Printf("[Metadata] manga search failed for %s: %v", title, err)
		return json.RawMessage(`{}`)
	}
	if result == nil {
		return json.RawMessage(`{}`)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(data)
}
