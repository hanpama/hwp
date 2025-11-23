package document

// ContentNode is the interface for document content items
type ContentNode interface {
	IsContent()
}

// Paragraph represents a paragraph with text
type Paragraph struct {
	Text string
}

func (p *Paragraph) IsContent() {}

// Table represents a table with cells
type Table struct {
	Rows  int
	Cols  int
	Cells []Cell
}

func (t *Table) IsContent() {}

// Cell represents a table cell
type Cell struct {
	Row     int
	Col     int
	RowSpan int
	ColSpan int
	Text    string
}

// Image represents an image or drawing object
type Image struct {
	// TODO: Add metadata fields (size, caption, format) when image extraction is implemented
}

func (i *Image) IsContent() {}

type ContentNodeScanner interface {
	Next() (ContentNode, error)
}
