package render

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

type Cell struct {
	Row     int
	Col     int
	Text    string
	RowSpan int
	ColSpan int
}

type Table struct {
	Rows  int
	Cols  int
	Cells []*Cell
}

// Layout represents the computed layout of the table.
// Separates layout computation from rendering to manage complexity.
type Layout struct {
	table *Table

	cellOwner  [][]*Cell          // cellOwner[row][col] = the original Cell that owns this grid cell
	colWidths  []int              // content width for each column
	rowHeights []int              // display row count for each table row (accounting for multiline text)
	cellLines  map[*Cell][]string // cell text split by newlines
}

// Render renders the table to ASCII string
func (t *Table) Render() string {
	layout := t.buildLayout()
	return layout.render()
}

func (t *Table) buildLayout() *Layout {
	layout := &Layout{
		table:      t,
		cellOwner:  make([][]*Cell, t.Rows),
		colWidths:  make([]int, t.Cols),
		rowHeights: make([]int, t.Rows),
		cellLines:  make(map[*Cell][]string),
	}

	for i := range layout.cellOwner {
		layout.cellOwner[i] = make([]*Cell, t.Cols)
	}

	for _, cell := range t.Cells {
		for r := 0; r < cell.RowSpan && cell.Row+r < t.Rows; r++ {
			for c := 0; c < cell.ColSpan && cell.Col+c < t.Cols; c++ {
				layout.cellOwner[cell.Row+r][cell.Col+c] = cell
			}
		}
	}

	for _, cell := range t.Cells {
		lines := strings.Split(cell.Text, "\n")
		layout.cellLines[cell] = lines
	}

	layout.computeColWidths()
	layout.computeRowHeights()

	return layout
}

func (l *Layout) computeColWidths() {
	for i := range l.colWidths {
		l.colWidths[i] = 1
	}

	// Single-column cells establish initial widths
	for _, cell := range l.table.Cells {
		if cell.ColSpan == 1 {
			lines := l.cellLines[cell]
			maxWidth := 0
			for _, line := range lines {
				width := displayWidth(line)
				if width > maxWidth {
					maxWidth = width
				}
			}
			if maxWidth > l.colWidths[cell.Col] {
				l.colWidths[cell.Col] = maxWidth
			}
		}
	}

	// Distribute extra width needed for multi-column cells
	for _, cell := range l.table.Cells {
		if cell.ColSpan > 1 {
			lines := l.cellLines[cell]
			maxWidth := 0
			for _, line := range lines {
				width := displayWidth(line)
				if width > maxWidth {
					maxWidth = width
				}
			}

			totalWidth := 0
			for c := 0; c < cell.ColSpan; c++ {
				totalWidth += l.colWidths[cell.Col+c]
			}

			if maxWidth > totalWidth {
				extra := maxWidth - totalWidth
				perCol := extra / cell.ColSpan
				remainder := extra % cell.ColSpan

				for c := 0; c < cell.ColSpan; c++ {
					l.colWidths[cell.Col+c] += perCol
					if c < remainder {
						l.colWidths[cell.Col+c]++
					}
				}
			}
		}
	}
}

func (l *Layout) computeRowHeights() {
	for row := 0; row < l.table.Rows; row++ {
		maxLines := 1

		for _, cell := range l.table.Cells {
			if cell.Row == row {
				lineCount := len(l.cellLines[cell])
				if lineCount > maxLines {
					maxLines = lineCount
				}
			}
		}

		l.rowHeights[row] = maxLines
	}
}

func (l *Layout) render() string {
	var sb strings.Builder

	sb.WriteString(l.renderBorderLine(-1))
	sb.WriteString("\n")

	for rowIdx := 0; rowIdx < l.table.Rows; rowIdx++ {
		displayRows := l.rowHeights[rowIdx]

		for displayRowIdx := 0; displayRowIdx < displayRows; displayRowIdx++ {
			sb.WriteString(l.renderContentLine(rowIdx, displayRowIdx))
			sb.WriteString("\n")
		}

		sb.WriteString(l.renderBorderLine(rowIdx))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderBorderLine renders a horizontal border line.
// rowIdx: -1 for top border, 0..Rows-1 for border after each row.
func (l *Layout) renderBorderLine(rowIdx int) string {
	var sb strings.Builder

	sb.WriteString("+")

	for colIdx := 0; colIdx < l.table.Cols; colIdx++ {
		needsHorizontal := l.needsHorizontalLine(rowIdx, colIdx)
		if needsHorizontal {
			sb.WriteString(strings.Repeat("-", l.colWidths[colIdx]+2))
		} else {
			sb.WriteString(strings.Repeat(" ", l.colWidths[colIdx]+2))
		}

		if colIdx < l.table.Cols-1 {
			needsVertical := l.needsVerticalLine(rowIdx, colIdx)
			if needsVertical {
				sb.WriteString("+")
			} else {
				sb.WriteString("-")
			}
		}
	}

	sb.WriteString("+")

	return sb.String()
}

func (l *Layout) needsHorizontalLine(rowIdx int, colIdx int) bool {
	if rowIdx == -1 {
		return true
	}

	if rowIdx == l.table.Rows-1 {
		return true
	}

	cellAbove := l.cellOwner[rowIdx][colIdx]
	cellBelow := l.cellOwner[rowIdx+1][colIdx]

	return cellAbove != cellBelow
}

func (l *Layout) needsVerticalLine(rowIdx int, colIdx int) bool {
	if rowIdx == -1 {
		return true
	}

	if rowIdx == l.table.Rows-1 {
		return true
	}

	cellAboveLeft := l.cellOwner[rowIdx][colIdx]
	cellAboveRight := l.cellOwner[rowIdx][colIdx+1]
	cellBelowLeft := l.cellOwner[rowIdx+1][colIdx]
	cellBelowRight := l.cellOwner[rowIdx+1][colIdx+1]

	return cellAboveLeft != cellAboveRight || cellBelowLeft != cellBelowRight
}

// renderContentLine renders a single display row of content.
// rowIdx: table row index
// displayRowIdx: display row index within this table row (0-based)
func (l *Layout) renderContentLine(rowIdx int, displayRowIdx int) string {
	var sb strings.Builder

	sb.WriteString("|")

	colIdx := 0
	for colIdx < l.table.Cols {
		owner := l.cellOwner[rowIdx][colIdx]

		isStartOfColumn := owner != nil && owner.Col == colIdx

		if !isStartOfColumn {
			colIdx++
			continue
		}

		colspan := owner.ColSpan
		totalContentWidth := 0
		for c := 0; c < colspan; c++ {
			totalContentWidth += l.colWidths[colIdx+c]
		}
		if colspan > 1 {
			totalContentWidth += (colspan - 1) * 3
		}

		lines := l.cellLines[owner]
		var text string
		if owner.Row == rowIdx {
			if displayRowIdx < len(lines) {
				text = lines[displayRowIdx]
			} else {
				text = ""
			}
		} else {
			// Rowspan cells only show text in their starting row
			text = ""
		}

		sb.WriteString(" ")
		width := displayWidth(text)
		padding := totalContentWidth - width
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(text)
		sb.WriteString(strings.Repeat(" ", padding))
		sb.WriteString(" ")

		nextColIdx := colIdx + colspan
		if nextColIdx < l.table.Cols {
			sb.WriteString("|")
		}

		colIdx = nextColIdx
	}

	sb.WriteString("|")

	return sb.String()
}

// displayWidth calculates the display width of a string using go-runewidth.
// Correctly handles East Asian Width properties including:
// - CJK characters (width 2)
// - Control characters (width 0)
// - Combining marks (width 0)
// - Emoji and variation selectors
// - Zero-Width Joiner (ZWJ) sequences
// - Ambiguous characters (width 1 or 2 depending on context)
func displayWidth(s string) int {
	return runewidth.StringWidth(s)
}
