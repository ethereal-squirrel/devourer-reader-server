package scanner

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"strings"
)

type EpubMetadata struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Publisher   string `json:"publisher"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Language    string `json:"language"`
	ISBN        string `json:"isbn"`
	CoverData   []byte `json:"cover"`
}

type opfPackage struct {
	XMLName  xml.Name    `xml:"package"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest []opfItem   `xml:"manifest>item"`
	Spine    opfSpine    `xml:"spine"`
}

type opfMetadata struct {
	Titles       []string        `xml:"title"`
	Creators     []string        `xml:"creator"`
	Publishers   []string        `xml:"publisher"`
	Dates        []string        `xml:"date"`
	Descriptions []string        `xml:"description"`
	Languages    []string        `xml:"language"`
	Identifiers  []opfIdentifier `xml:"identifier"`
	Meta         []opfMeta       `xml:"meta"`
}

type opfIdentifier struct {
	Scheme string `xml:"scheme,attr"`
	Value  string `xml:",chardata"`
}

type opfMeta struct {
	Name     string `xml:"name,attr"`
	Content  string `xml:"content,attr"`
	Property string `xml:"property,attr"`
	Value    string `xml:",chardata"`
}

type opfItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type opfSpine struct {
	Toc string `xml:"toc,attr"`
}

func ScanEpub(filePath string) (*EpubMetadata, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	index := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		normalized := strings.ReplaceAll(f.Name, "\\", "/")
		index[normalized] = f
		index[strings.ToLower(normalized)] = f
	}

	opfPath := findOPFPath(index)
	if opfPath == "" {
		for key := range index {
			if strings.HasSuffix(key, ".opf") {
				opfPath = index[key].Name
				break
			}
		}
	}
	if opfPath == "" {
		return &EpubMetadata{}, nil
	}

	pkg, err := parseOPF(index, opfPath)
	if err != nil {
		return &EpubMetadata{}, nil
	}

	meta := &EpubMetadata{}

	if len(pkg.Metadata.Titles) > 0 {
		meta.Title = pkg.Metadata.Titles[0]
	}
	if len(pkg.Metadata.Creators) > 0 {
		meta.Author = pkg.Metadata.Creators[0]
	}
	if len(pkg.Metadata.Publishers) > 0 {
		meta.Publisher = pkg.Metadata.Publishers[0]
	}
	if len(pkg.Metadata.Dates) > 0 {
		meta.Date = pkg.Metadata.Dates[0]
	}
	if len(pkg.Metadata.Descriptions) > 0 {
		meta.Description = pkg.Metadata.Descriptions[0]
	}
	if len(pkg.Metadata.Languages) > 0 {
		meta.Language = pkg.Metadata.Languages[0]
	}
	for _, id := range pkg.Metadata.Identifiers {
		s := strings.ToLower(id.Scheme)
		if s == "isbn" || strings.Contains(s, "isbn") {
			meta.ISBN = strings.TrimSpace(id.Value)
			break
		}
	}

	opfDir := ""
	if idx := strings.LastIndex(opfPath, "/"); idx >= 0 {
		opfDir = opfPath[:idx+1]
	}

	var coverHref string
	for _, item := range pkg.Manifest {
		if strings.HasPrefix(item.MediaType, "image/") {
			id := strings.ToLower(item.ID)
			if id == "cover-image" || id == "cover" || strings.Contains(id, "cover") {
				coverHref = opfDir + item.Href
				break
			}
		}
	}
	if coverHref == "" {
		for _, item := range pkg.Manifest {
			if strings.HasPrefix(item.MediaType, "image/") {
				coverHref = opfDir + item.Href
				break
			}
		}
	}

	if coverHref != "" {
		meta.CoverData = readZipEntry(index, coverHref)
	}

	return meta, nil
}

func findOPFPath(index map[string]*zip.File) string {
	type rootfile struct {
		FullPath string `xml:"full-path,attr"`
	}
	type container struct {
		Rootfiles []rootfile `xml:"rootfiles>rootfile"`
	}

	f, ok := index["meta-inf/container.xml"]
	if !ok {
		return ""
	}
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()
	var c container
	if err := xml.NewDecoder(rc).Decode(&c); err != nil {
		return ""
	}
	if len(c.Rootfiles) > 0 {
		return c.Rootfiles[0].FullPath
	}
	return ""
}

func parseOPF(index map[string]*zip.File, opfPath string) (*opfPackage, error) {
	normalized := strings.ReplaceAll(opfPath, "\\", "/")
	f, ok := index[normalized]
	if !ok {
		f, ok = index[strings.ToLower(normalized)]
	}
	if !ok {
		return nil, io.EOF
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var pkg opfPackage
	if err := xml.NewDecoder(rc).Decode(&pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func readZipEntry(index map[string]*zip.File, name string) []byte {
	normalized := strings.ReplaceAll(name, "\\", "/")
	f, ok := index[normalized]
	if !ok {
		f, ok = index[strings.ToLower(normalized)]
	}
	if !ok {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil
	}
	return data
}
