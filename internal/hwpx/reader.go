package hwpx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/hanpama/hwp/internal/document"
)

// Reader provides access to HWPX document content
type Reader struct {
	zipReader *zip.Reader
	version   Version
	sections  []*Section
}

// Version represents the HWPX format version
type Version struct {
	Major       int
	Minor       int
	Micro       int
	BuildNumber int
	XMLVersion  string
}

// Section represents a section XML file in the HWPX document
type Section struct {
	name   string
	reader io.ReadCloser
}

// Open opens an HWPX file and returns a Reader
func Open(r io.ReaderAt, size int64) (*Reader, error) {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("failed to open HWPX as ZIP: %w", err)
	}

	reader := &Reader{
		zipReader: zipReader,
	}

	if err := reader.validateMimetype(); err != nil {
		return nil, err
	}

	if err := reader.parseVersion(); err != nil {
		return nil, err
	}

	if err := reader.loadSections(); err != nil {
		return nil, err
	}

	return reader, nil
}

func (r *Reader) validateMimetype() error {
	file, err := r.zipReader.Open("mimetype")
	if err != nil {
		return fmt.Errorf("mimetype file not found: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read mimetype: %w", err)
	}

	mimetype := string(data)
	if mimetype != "application/hwp+zip" {
		return fmt.Errorf("invalid mimetype: expected 'application/hwp+zip', got '%s'", mimetype)
	}

	return nil
}

func (r *Reader) parseVersion() error {
	file, err := r.zipReader.Open("version.xml")
	if err != nil {
		return fmt.Errorf("version.xml not found: %w", err)
	}
	defer file.Close()

	var versionDoc struct {
		XMLName     xml.Name `xml:"HCFVersion"`
		Major       int      `xml:"major,attr"`
		Minor       int      `xml:"minor,attr"`
		Micro       int      `xml:"micro,attr"`
		BuildNumber int      `xml:"buildNumber,attr"`
		XMLVersion  string   `xml:"xmlVersion,attr"`
	}

	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&versionDoc); err != nil {
		return fmt.Errorf("failed to parse version.xml: %w", err)
	}

	r.version = Version{
		Major:       versionDoc.Major,
		Minor:       versionDoc.Minor,
		Micro:       versionDoc.Micro,
		BuildNumber: versionDoc.BuildNumber,
		XMLVersion:  versionDoc.XMLVersion,
	}

	return nil
}

func (r *Reader) loadSections() error {
	r.sections = make([]*Section, 0)

	for _, file := range r.zipReader.File {
		if strings.HasPrefix(file.Name, "Contents/section") && strings.HasSuffix(file.Name, ".xml") {
			r.sections = append(r.sections, &Section{
				name: file.Name,
			})
		}
	}

	if len(r.sections) == 0 {
		return fmt.Errorf("no section files found in Contents/")
	}

	return nil
}

// NewContentScanner creates a ContentNodeScanner for the HWPX document
func (r *Reader) NewContentScanner() (document.ContentNodeScanner, error) {
	if len(r.sections) == 0 {
		return nil, fmt.Errorf("no sections available")
	}

	// Open the first section file
	file, err := r.zipReader.Open(r.sections[0].name)
	if err != nil {
		return nil, fmt.Errorf("failed to open section file: %w", err)
	}

	return NewContentScanner(file)
}
