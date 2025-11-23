package hwpv5

import (
	"fmt"
	"io"

	"github.com/hanpama/hwp/internal/document"
)

// ContentScanner implements document.ContentNodeScanner using a state machine approach.
// It converts flat record stream into hierarchical content nodes.
type ContentScanner struct {
	reader         *Reader
	currentSection int
	scanner        *RecScanner
	sectionCloser  io.Closer

	// Single-record lookahead buffer (needed for skipChildren and table-end detection)
	bufferedRec Rec
	hasBuffered bool

	// State machine fields
	currentPara  *paragraphBuilder
	currentTable *tableBuilder
	tableLevel   uint16 // Level at which table started
}

type paragraphBuilder struct {
	textParts []string
}

type tableBuilder struct {
	rows        int
	cols        int
	cells       []document.Cell
	currentCell *document.Cell
	tableLevel  uint16 // Level at which table started
}

// Open opens an HWP 5.0 file and returns a ContentNodeScanner
func Open(file io.ReaderAt) (document.ContentNodeScanner, error) {
	reader, err := OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open HWP reader: %w", err)
	}

	scanner := &ContentScanner{
		reader:         reader,
		currentSection: -1,
	}

	if err := scanner.advanceSection(); err != nil {
		return nil, err
	}

	return scanner, nil
}

func (s *ContentScanner) advanceSection() error {
	if s.sectionCloser != nil {
		s.sectionCloser.Close()
		s.sectionCloser = nil
		s.scanner = nil
	}

	s.currentSection++
	if s.currentSection >= s.reader.SectionCount() {
		return io.EOF
	}

	sectionReader, err := s.reader.OpenSection(s.currentSection)
	if err != nil {
		return fmt.Errorf("failed to open section %d: %w", s.currentSection, err)
	}

	s.sectionCloser = sectionReader
	s.scanner = NewRecScanner(sectionReader)
	return nil
}

// Next returns the next content node using state machine pattern
func (s *ContentScanner) Next() (document.ContentNode, error) {
	for {
		rec, err := s.nextRecord()
		if err != nil {
			// If EOF and we have a table in progress, return it first
			if s.currentTable != nil {
				table := s.finishTable()
				return table, nil
			}
			return nil, err
		}

		// Check if we're in a table and the level has dropped to or below table level
		// This means the table has ended
		if s.currentTable != nil && rec.Lvl() <= s.currentTable.tableLevel {
			table := s.finishTable()
			// Put this record back in buffer to process in next iteration
			s.putBack(rec)
			return table, nil
		}

		switch r := rec.(type) {
		case RecParaHeader:
			// Start new paragraph
			s.currentPara = &paragraphBuilder{
				textParts: make([]string, 0),
			}

		case RecParaText:
			// Add text to current paragraph
			if s.currentPara != nil {
				for _, el := range r.Els {
					switch elem := el.(type) {
					case ParaTextString:
						s.currentPara.textParts = append(s.currentPara.textParts, elem.Value)
					case ParaTextLineBreak:
						s.currentPara.textParts = append(s.currentPara.textParts, "\n")
					case ParaTextTab:
						s.currentPara.textParts = append(s.currentPara.textParts, "\t")
					}
				}
			}

		case RecParaCharShape, RecParaLineSeg:
			// Paragraph complete (these records mark end of paragraph)
			if s.currentPara != nil {
				text := joinTextParts(s.currentPara.textParts)
				s.currentPara = nil

				if s.currentTable != nil && s.currentTable.currentCell != nil {
					// Inside table: add to current cell
					if s.currentTable.currentCell.Text != "" {
						s.currentTable.currentCell.Text += "\n"
					}
					s.currentTable.currentCell.Text += text
				} else {
					// Regular paragraph: return it
					return &document.Paragraph{Text: text}, nil
				}
			}

		case RecCtrlHeader:
			switch r.CtrlID {
			case 0x74626c20: // MAKE_4CHID('t','b','l',' ') - TABLE
				// Mark that we're entering a table control
				s.tableLevel = r.Lvl()
				// Table will be created when we see RecTable

			case 0x67736f20: // MAKE_4CHID('g','s','o',' ') - Drawing Object
				// Skip drawing object children and return image placeholder
				s.skipChildren(r.Lvl())
				return &document.Image{}, nil

			default:
				// Unknown control, skip its children
				s.skipChildren(r.Lvl())
			}

		case RecTable:
			// Create table (must be inside a table control)
			if s.currentTable == nil {
				s.currentTable = &tableBuilder{
					rows:       int(r.RowCount),
					cols:       int(r.ColCount),
					cells:      make([]document.Cell, 0),
					tableLevel: s.tableLevel,
				}
			}

		case RecListHeader:
			// Start new cell in table
			if s.currentTable != nil && r.IsCell {
				// Update table dimensions if needed
				if int(r.RowIndex)+int(r.RowSpan) > s.currentTable.rows {
					s.currentTable.rows = int(r.RowIndex) + int(r.RowSpan)
				}
				if int(r.ColIndex)+int(r.ColSpan) > s.currentTable.cols {
					s.currentTable.cols = int(r.ColIndex) + int(r.ColSpan)
				}

				cell := document.Cell{
					Row:     int(r.RowIndex),
					Col:     int(r.ColIndex),
					RowSpan: int(r.RowSpan),
					ColSpan: int(r.ColSpan),
					Text:    "",
				}
				s.currentTable.cells = append(s.currentTable.cells, cell)
				s.currentTable.currentCell = &s.currentTable.cells[len(s.currentTable.cells)-1]
			}
		}
	}
}

// nextRecord returns the next record, automatically advancing sections
func (s *ContentScanner) nextRecord() (Rec, error) {
	// Return buffered record if available
	if s.hasBuffered {
		rec := s.bufferedRec
		s.hasBuffered = false
		s.bufferedRec = nil
		return rec, nil
	}

	for {
		if s.scanner == nil {
			return nil, io.EOF
		}

		rec, err := s.scanner.ScanNext()
		if err != nil {
			if err == io.EOF {
				if advErr := s.advanceSection(); advErr != nil {
					return nil, advErr
				}
				continue
			}
			return nil, err
		}
		return rec, nil
	}
}

// putBack puts a record back into the buffer to be read again
func (s *ContentScanner) putBack(rec Rec) {
	s.bufferedRec = rec
	s.hasBuffered = true
}

// finishTable completes the current table and returns it
func (s *ContentScanner) finishTable() *document.Table {
	if s.currentTable == nil {
		return nil
	}

	table := &document.Table{
		Rows:  s.currentTable.rows,
		Cols:  s.currentTable.cols,
		Cells: s.currentTable.cells,
	}
	s.currentTable = nil
	return table
}

// skipChildren skips all records that are children of the given parent level
func (s *ContentScanner) skipChildren(parentLevel uint16) error {
	for {
		rec, err := s.nextRecord()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if rec.Lvl() <= parentLevel {
			// This record is not a child, put it back
			s.putBack(rec)
			return nil
		}
	}
}

// joinTextParts joins text parts into a single string
func joinTextParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	totalLen := 0
	for _, p := range parts {
		totalLen += len(p)
	}
	result := make([]byte, 0, totalLen)
	for _, p := range parts {
		result = append(result, p...)
	}
	return string(result)
}
