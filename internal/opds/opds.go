package opds

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/devourer/server/internal/db/queries"
)

const (
	opdsNS      = "http://www.w3.org/2005/Atom"
	opdsDCNS    = "http://purl.org/dc/terms/"
	opdsAcqNS   = "http://opds-spec.org/2010/catalog"
	contentType = "application/atom+xml"
)

var globalCache = NewCache()

func InvalidateLibrary(libraryID int64) {
	globalCache.Invalidate("catalog")
	globalCache.Invalidate(fmt.Sprintf("library_%d", libraryID))
}

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	XMLNS   string   `xml:"xmlns,attr"`
	DCNS    string   `xml:"xmlns:dc,attr,omitempty"`

	ID      string  `xml:"id"`
	Title   string  `xml:"title"`
	Updated string  `xml:"updated"`
	Links   []Link  `xml:"link"`
	Entries []Entry `xml:"entry"`
}

type Link struct {
	Rel   string `xml:"rel,attr,omitempty"`
	Href  string `xml:"href,attr"`
	Type  string `xml:"type,attr,omitempty"`
	Title string `xml:"title,attr,omitempty"`
}

type Entry struct {
	ID      string   `xml:"id"`
	Title   string   `xml:"title"`
	Updated string   `xml:"updated"`
	Author  *Author  `xml:"author,omitempty"`
	Content *Content `xml:"content,omitempty"`
	Links   []Link   `xml:"link"`
}

type Author struct {
	Name string `xml:"name"`
}

type Content struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func marshalFeed(f *Feed) ([]byte, error) {
	f.XMLNS = opdsNS
	out, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func CatalogFeed(d *sql.DB, baseURL string) ([]byte, error) {
	cacheKey := "catalog"
	if data, ok := globalCache.Get(cacheKey); ok {
		return data, nil
	}

	libs, err := queries.ListLibraries(d)
	if err != nil {
		return nil, err
	}

	feed := &Feed{
		ID:      baseURL + "/opds/v1.2/catalog",
		Title:   "Devourer Library",
		Updated: now(),
		Links: []Link{
			{Rel: "self", Href: baseURL + "/opds/v1.2/catalog", Type: contentType + ";profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: baseURL + "/opds/v1.2/catalog", Type: contentType + ";profile=opds-catalog;kind=navigation"},
		},
	}

	for _, lib := range libs {
		if lib.Type != "book" {
			continue
		}
		feed.Entries = append(feed.Entries, Entry{
			ID:      fmt.Sprintf("%s/opds/v1.2/library/%d", baseURL, lib.ID),
			Title:   lib.Name,
			Updated: now(),
			Content: &Content{Type: "text", Value: lib.Name + " — book library"},
			Links: []Link{
				{
					Rel:  "subsection",
					Href: fmt.Sprintf("%s/opds/v1.2/library/%d", baseURL, lib.ID),
					Type: contentType + ";profile=opds-catalog;kind=acquisition",
				},
			},
		})
	}

	data, err := marshalFeed(feed)
	if err != nil {
		return nil, err
	}
	globalCache.Set(cacheKey, data)
	return data, nil
}

func LibraryFeed(d *sql.DB, baseURL string, libraryID int64) ([]byte, error) {
	cacheKey := fmt.Sprintf("library_%d", libraryID)
	if data, ok := globalCache.Get(cacheKey); ok {
		return data, nil
	}

	lib, err := queries.GetLibraryByID(d, libraryID)
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	books, err := queries.ListBookFilesByLibrary(d, libraryID)
	if err != nil {
		return nil, err
	}

	sort.Slice(books, func(i, j int) bool {
		return naturalLess(books[i].Title, books[j].Title)
	})

	feed := &Feed{
		ID:      fmt.Sprintf("%s/opds/v1.2/library/%d", baseURL, libraryID),
		Title:   lib.Name,
		Updated: now(),
		Links: []Link{
			{Rel: "self", Href: fmt.Sprintf("%s/opds/v1.2/library/%d", baseURL, libraryID), Type: contentType + ";profile=opds-catalog;kind=acquisition"},
			{Rel: "start", Href: baseURL + "/opds/v1.2/catalog", Type: contentType + ";profile=opds-catalog;kind=navigation"},
			{
				Rel:   "search",
				Href:  fmt.Sprintf("%s/opds/v1.2/library/%d/search?q={searchTerms}", baseURL, libraryID),
				Type:  "application/opensearchdescription+xml",
				Title: "Search " + lib.Name,
			},
		},
	}

	for _, book := range books {
		entry := bookEntry(book.ID, book.Title, book.FileFormat, book.Metadata, baseURL, libraryID)
		feed.Entries = append(feed.Entries, entry)
	}

	data, err := marshalFeed(feed)
	if err != nil {
		return nil, err
	}
	globalCache.Set(cacheKey, data)
	return data, nil
}

func SearchFeed(d *sql.DB, baseURL string, libraryID int64, query string) ([]byte, error) {
	lib, err := queries.GetLibraryByID(d, libraryID)
	if err != nil {
		return nil, err
	}

	books, err := queries.SearchBookFiles(d, query)
	if err != nil {
		return nil, err
	}

	feed := &Feed{
		ID:      fmt.Sprintf("%s/opds/v1.2/library/%d/search", baseURL, libraryID),
		Title:   "Search: " + query + " — " + lib.Name,
		Updated: now(),
		Links: []Link{
			{Rel: "self", Href: fmt.Sprintf("%s/opds/v1.2/library/%d/search?q=%s", baseURL, libraryID, query), Type: contentType},
			{Rel: "start", Href: baseURL + "/opds/v1.2/catalog", Type: contentType + ";profile=opds-catalog;kind=navigation"},
		},
	}

	for _, book := range books {
		if book.LibraryID != libraryID {
			continue
		}
		feed.Entries = append(feed.Entries, bookEntry(book.ID, book.Title, book.FileFormat, book.Metadata, baseURL, libraryID))
	}

	return marshalFeed(feed)
}

func SingleBookFeed(d *sql.DB, baseURL string, libraryID, bookID int64) ([]byte, error) {
	book, err := queries.GetBookFileByID(d, bookID)
	if err != nil {
		return nil, err
	}

	feed := &Feed{
		ID:      fmt.Sprintf("%s/opds/v1.2/library/%d/book/%d", baseURL, libraryID, bookID),
		Title:   book.Title,
		Updated: now(),
		Links: []Link{
			{Rel: "self", Href: fmt.Sprintf("%s/opds/v1.2/library/%d/book/%d", baseURL, libraryID, bookID), Type: contentType},
			{Rel: "start", Href: baseURL + "/opds/v1.2/catalog", Type: contentType + ";profile=opds-catalog;kind=navigation"},
		},
		Entries: []Entry{bookEntry(book.ID, book.Title, book.FileFormat, book.Metadata, baseURL, libraryID)},
	}

	return marshalFeed(feed)
}

func bookEntry(id int64, title, format string, metaRaw []byte, baseURL string, libraryID int64) Entry {
	var meta map[string]any
	json.Unmarshal(metaRaw, &meta)

	entry := Entry{
		ID:      fmt.Sprintf("%s/opds/v1.2/library/%d/book/%d", baseURL, libraryID, id),
		Title:   title,
		Updated: now(),
		Links: []Link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: fmt.Sprintf("%s/stream/%d/%d", baseURL, libraryID, id),
				Type: mimeType(format),
			},
			{
				Rel:  "http://opds-spec.org/image",
				Href: fmt.Sprintf("%s/cover-image/%d/%d.jpg", baseURL, libraryID, id),
				Type: "image/jpeg",
			},
		},
	}

	if meta != nil {
		if authors, ok := meta["authors"].([]any); ok && len(authors) > 0 {
			if name, ok := authors[0].(string); ok {
				entry.Author = &Author{Name: name}
			}
		}
		if desc, ok := meta["description"].(string); ok && desc != "" {
			entry.Content = &Content{Type: "html", Value: desc}
		}
	}

	return entry
}

func mimeType(format string) string {
	switch strings.ToLower(format) {
	case "epub":
		return "application/epub+zip"
	case "pdf":
		return "application/pdf"
	case "mobi":
		return "application/x-mobipocket-ebook"
	default:
		return "application/octet-stream"
	}
}

func naturalLess(a, b string) bool {
	return strings.ToLower(a) < strings.ToLower(b)
}
