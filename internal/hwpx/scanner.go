package hwpx

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/hanpama/hwp/internal/document"
)

// ContentScanner parses HWPX section XML and emits content nodes
type ContentScanner struct {
	decoder *xml.Decoder
	closer  io.Closer
}

// NewContentScanner creates a new ContentScanner from a section XML reader
func NewContentScanner(r io.ReadCloser) (*ContentScanner, error) {
	decoder := xml.NewDecoder(r)
	return &ContentScanner{
		decoder: decoder,
		closer:  r,
	}, nil
}

// Next returns the next content node from the document
func (s *ContentScanner) Next() (document.ContentNode, error) {
	for {
		token, err := s.decoder.Token()
		if err == io.EOF {
			return nil, io.EOF
		}
		if err != nil {
			return nil, fmt.Errorf("XML parse error: %w", err)
		}

		switch elem := token.(type) {
		case xml.StartElement:
			node, err := s.handleStartElement(elem)
			if err != nil {
				return nil, err
			}
			if node != nil {
				return node, nil
			}
		}
	}
}

func (s *ContentScanner) handleStartElement(elem xml.StartElement) (document.ContentNode, error) {
	localName := elem.Name.Local

	switch localName {
	case "p":
		return s.parseParagraph(elem)
	case "tbl":
		return s.parseTable(elem)
	}

	return nil, nil
}

// parseParagraph parses <hp:p> element into a Paragraph node or Table node
func (s *ContentScanner) parseParagraph(elem xml.StartElement) (document.ContentNode, error) {
	var para ParagraphElement
	if err := s.decoder.DecodeElement(&para, &elem); err != nil {
		return nil, fmt.Errorf("failed to decode paragraph: %w", err)
	}

	// Check if this paragraph contains a table
	for _, run := range para.Runs {
		if run.Table != nil {
			return s.parseTableElement(run.Table)
		}
	}

	text := para.extractText()
	if text == "" {
		return nil, nil
	}

	return &document.Paragraph{
		Text: text,
	}, nil
}

// parseTable parses <hp:tbl> element into a Table node
func (s *ContentScanner) parseTable(elem xml.StartElement) (document.ContentNode, error) {
	var tbl TableElement
	if err := s.decoder.DecodeElement(&tbl, &elem); err != nil {
		return nil, fmt.Errorf("failed to decode table: %w", err)
	}

	return s.parseTableElement(&tbl)
}

func (s *ContentScanner) parseTableElement(tbl *TableElement) (document.ContentNode, error) {
	rowCount := tbl.RowCnt
	colCount := tbl.ColCnt

	if rowCount == 0 || colCount == 0 {
		return nil, nil
	}

	table := &document.Table{
		Rows:  rowCount,
		Cols:  colCount,
		Cells: make([]document.Cell, 0),
	}

	for _, tr := range tbl.Rows {
		for _, tc := range tr.Cells {
			cell := s.parseCell(tc)
			if cell != nil {
				table.Cells = append(table.Cells, *cell)
			}
		}
	}

	return table, nil
}

func (s *ContentScanner) parseCell(tc TableCell) *document.Cell {
	row := tc.CellAddr.RowAddr
	col := tc.CellAddr.ColAddr
	rowSpan := tc.CellSpan.RowSpan
	colSpan := tc.CellSpan.ColSpan

	if rowSpan == 0 {
		rowSpan = 1
	}
	if colSpan == 0 {
		colSpan = 1
	}

	var textParts []string
	for _, p := range tc.SubList.Paragraphs {
		text := p.extractText()
		if text != "" {
			textParts = append(textParts, text)
		}
	}

	cellText := strings.Join(textParts, "\n")

	return &document.Cell{
		Row:     row,
		Col:     col,
		RowSpan: rowSpan,
		ColSpan: colSpan,
		Text:    cellText,
	}
}

// Close closes the underlying reader
func (s *ContentScanner) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// XML element structures with proper namespace handling

type ParagraphElement struct {
	XMLName xml.Name `xml:"p"`
	ID      string   `xml:"id,attr"`
	Runs    []Run    `xml:"run"`
}

func (p *ParagraphElement) extractText() string {
	var parts []string
	for _, run := range p.Runs {
		text := run.extractText()
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}

type Run struct {
	XMLName   xml.Name      `xml:"run"`
	TextNodes []TextNode    `xml:"t"`
	LineBreak *LineBreak    `xml:"lineBreak"`
	Table     *TableElement `xml:"tbl"`
}

func (r *Run) extractText() string {
	var parts []string
	for _, t := range r.TextNodes {
		parts = append(parts, t.Text)
	}
	if r.LineBreak != nil {
		parts = append(parts, "\n")
	}
	return strings.Join(parts, "")
}

type TextNode struct {
	XMLName xml.Name `xml:"t"`
	Text    string   `xml:",chardata"`
}

type LineBreak struct {
	XMLName xml.Name `xml:"lineBreak"`
}

type TableElement struct {
	XMLName xml.Name   `xml:"tbl"`
	ID      string     `xml:"id,attr"`
	RowCnt  int        `xml:"rowCnt,attr"`
	ColCnt  int        `xml:"colCnt,attr"`
	Rows    []TableRow `xml:"tr"`
}

type TableRow struct {
	XMLName xml.Name    `xml:"tr"`
	Cells   []TableCell `xml:"tc"`
}

type TableCell struct {
	XMLName  xml.Name `xml:"tc"`
	Name     string   `xml:"name,attr"`
	SubList  SubList  `xml:"subList"`
	CellAddr CellAddr `xml:"cellAddr"`
	CellSpan CellSpan `xml:"cellSpan"`
}

type SubList struct {
	XMLName    xml.Name           `xml:"subList"`
	Paragraphs []ParagraphElement `xml:"p"`
}

type CellAddr struct {
	XMLName xml.Name `xml:"cellAddr"`
	ColAddr int      `xml:"colAddr,attr"`
	RowAddr int      `xml:"rowAddr,attr"`
}

type CellSpan struct {
	XMLName xml.Name `xml:"cellSpan"`
	ColSpan int      `xml:"colSpan,attr"`
	RowSpan int      `xml:"rowSpan,attr"`
}
