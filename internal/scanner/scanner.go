package scanner

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	"github.com/devourer/server/internal/metadata"
	"github.com/devourer/server/internal/opds"
)

type ScanProgress struct {
	Series      string `json:"series"`
	LibraryType string `json:"libraryType"`
	Status      string `json:"status"`
	Progress    *struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progress,omitempty"`
	Error string `json:"error,omitempty"`
}

type ScanStatus struct {
	InProgress      bool           `json:"inProgress"`
	LibraryType     string         `json:"libraryType"`
	Series          []ScanProgress `json:"series"`
	CompletedSeries int            `json:"completedSeries"`
	TotalSeries     int            `json:"totalSeries"`
}

var (
	scanMu    sync.RWMutex
	scanState = make(map[int64]*ScanStatus)
)

func getScanStatus(id int64) *ScanStatus {
	scanMu.RLock()
	defer scanMu.RUnlock()
	return scanState[id]
}

func setScanStatus(id int64, s *ScanStatus) {
	scanMu.Lock()
	scanState[id] = s
	scanMu.Unlock()
}

func updateSeriesStatus(id int64, seriesName, status string) {
	scanMu.Lock()
	defer scanMu.Unlock()
	st, ok := scanState[id]
	if !ok {
		return
	}
	for i := range st.Series {
		if st.Series[i].Series == seriesName {
			st.Series[i].Status = status
			if status == "complete" {
				st.CompletedSeries++
			}
			return
		}
	}
}

func updateSeriesError(id int64, seriesName, errMsg string) {
	scanMu.Lock()
	defer scanMu.Unlock()
	st, ok := scanState[id]
	if !ok {
		return
	}
	for i := range st.Series {
		if st.Series[i].Series == seriesName {
			st.Series[i].Status = "error"
			st.Series[i].Error = errMsg
			return
		}
	}
}

type Config struct {
	DB          *sql.DB
	AssetsPath  string
	PluginsPath string
	Providers   map[string]*metadata.Provider
}

type ScanResult struct {
	Status    bool     `json:"status"`
	Message   string   `json:"message"`
	Remaining []string `json:"remaining,omitempty"`
}

func ScanLibrary(cfg *Config, libraryID int64) (*ScanResult, error) {
	lib, err := queries.GetLibraryByID(cfg.DB, libraryID)
	if err != nil {
		return nil, fmt.Errorf("library %d not found: %w", libraryID, err)
	}

	if st := getScanStatus(lib.ID); st != nil && st.InProgress {
		return &ScanResult{Status: false, Message: "Scan already in progress"}, nil
	}

	topLevel := listTopLevel(lib.Path)

	status := &ScanStatus{
		InProgress:  true,
		LibraryType: lib.Type,
		TotalSeries: len(topLevel),
	}
	for _, name := range topLevel {
		status.Series = append(status.Series, ScanProgress{
			Series:      name,
			LibraryType: lib.Type,
			Status:      "scanning",
		})
	}
	setScanStatus(lib.ID, status)

	go func() {
		if lib.Type == "book" {
			ScanBookLibrary(cfg, lib, topLevel)
		} else {
			ScanMangaLibrary(cfg, lib, topLevel)
		}
		scanMu.Lock()
		if st, ok := scanState[lib.ID]; ok {
			st.InProgress = false
		}
		scanMu.Unlock()
		opds.InvalidateLibrary(lib.ID)
	}()

	return &ScanResult{
		Status:    true,
		Message:   "Library scan started",
		Remaining: topLevel,
	}, nil
}

func GetScanStatus(libraryID int64) map[string]any {
	st := getScanStatus(libraryID)
	if st == nil {
		return map[string]any{
			"status":      false,
			"message":     "No scan in progress",
			"libraryType": "",
		}
	}
	remaining := []string{}
	for _, s := range st.Series {
		if s.Status == "scanning" {
			remaining = append(remaining, s.Series)
		}
	}
	return map[string]any{
		"status":      true,
		"inProgress":  st.InProgress,
		"libraryType": st.LibraryType,
		"progress": map[string]any{
			"completed": st.CompletedSeries,
			"total":     st.TotalSeries,
			"series":    st.Series,
		},
		"remaining": remaining,
	}
}

func DeleteBook(d *sql.DB, libraryPath, filePath string) error {
	book, err := queries.GetBookFileByPath(d, filePath)
	if err != nil {
		return nil
	}
	lib, err := queries.GetLibraryByPath(d, libraryPath)
	metaBase := filepath.Join(libraryPath, ".devourer")
	if err == nil {
		metaBase = lib.MetaBase()
	}
	queries.DeleteRecentlyReadByFileID(d, book.ID)
	queries.DeleteReadingStatusByFileID(d, book.ID)
	queries.DeleteUserRatingsByFileID(d, book.ID)
	queries.DeleteUserTagsByFileID(d, book.ID)
	queries.DeleteBookFile(d, book.ID)
	removeCoverDir(metaBase, "files", book.ID)
	return nil
}

func DeleteManga(d *sql.DB, libraryPath, filePath string) error {
	mf, err := queries.GetMangaFileByPath(d, filePath)
	if err != nil {
		return nil
	}

	queries.DeleteRecentlyReadByFileID(d, mf.ID)
	queries.DeleteReadingStatusByFileID(d, mf.ID)
	queries.DeleteUserRatingsByFileID(d, mf.ID)
	queries.DeleteUserTagsByFileID(d, mf.ID)
	queries.DeleteMangaFile(d, mf.ID)

	lib2, err2 := queries.GetLibraryByPath(d, libraryPath)
	metaBase2 := filepath.Join(libraryPath, ".devourer")
	if err2 == nil {
		metaBase2 = lib2.MetaBase()
	}

	remaining, _ := queries.ListMangaFilesBySeries(d, mf.SeriesID)
	if len(remaining) == 0 {
		queries.DeleteMangaSeries(d, mf.SeriesID)
		removeCoverDir(metaBase2, "series", mf.SeriesID)
	} else {
		removePreview(metaBase2, mf.SeriesID, mf.FileName+".jpg")
	}
	return nil
}

func UpdateRecentlyRead(d *sql.DB, lib *db.Library, fileID int64, page string, userID int64) {
	var seriesID int64
	var totalPages, volume, chapter int

	if lib.Type == "book" {
		if f, err := queries.GetBookFileByID(d, fileID); err == nil {
			totalPages = f.TotalPages
		}
	} else {
		if f, err := queries.GetMangaFileByID(d, fileID); err == nil {
			seriesID = f.SeriesID
			totalPages = f.TotalPages
			volume = f.Volume
			chapter = f.Chapter
		}
	}

	queries.DeleteRecentlyReadByLibraryAndFile(d, lib.ID, fileID, userID)
	rr := &db.RecentlyRead{
		IsLocal:     false,
		LibraryID:   lib.ID,
		SeriesID:    seriesID,
		FileID:      fileID,
		CurrentPage: page,
		TotalPages:  totalPages,
		Volume:      volume,
		Chapter:     chapter,
		UserID:      userID,
	}
	queries.CreateRecentlyRead(d, rr)
	queries.TrimRecentlyRead(d, userID, 10)
}
