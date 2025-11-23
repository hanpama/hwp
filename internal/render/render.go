package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/hanpama/hwp/internal/document"
)

// RenderText renders a ContentNodeScanner to plain text with ASCII tables.
func RenderText(scanner document.ContentNodeScanner, w io.Writer) error {
	for {
		node, err := scanner.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading content: %w", err)
		}

		switch n := node.(type) {
		case *document.Paragraph:
			if err := renderParagraph(n, w); err != nil {
				return err
			}
		case *document.Table:
			if err := renderTable(n, w); err != nil {
				return err
			}
			fmt.Fprintln(w)
		case *document.Image:
			if err := renderImage(n, w); err != nil {
				return err
			}
		}
	}
}

func renderParagraph(para *document.Paragraph, w io.Writer) error {
	text := strings.TrimRight(para.Text, "\n")
	if text != "" {
		_, err := fmt.Fprintln(w, text)
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

func renderTable(docTable *document.Table, w io.Writer) error {
	if len(docTable.Cells) == 0 {
		return nil
	}

	t := &Table{
		Rows:  docTable.Rows,
		Cols:  docTable.Cols,
		Cells: make([]*Cell, 0, len(docTable.Cells)),
	}

	for _, docCell := range docTable.Cells {
		text := strings.TrimSpace(docCell.Text)
		t.Cells = append(t.Cells, &Cell{
			Row:     docCell.Row,
			Col:     docCell.Col,
			Text:    text,
			RowSpan: docCell.RowSpan,
			ColSpan: docCell.ColSpan,
		})
	}

	_, err := fmt.Fprint(w, t.Render())
	return err
}

func renderImage(_ *document.Image, w io.Writer) error {
	_, err := fmt.Fprintln(w, "[IMAGE]")
	return err
}
