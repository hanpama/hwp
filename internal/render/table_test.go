package render

import (
	"strings"
	"testing"
)

func TestBasicTable(t *testing.T) {
	table := &Table{
		Rows: 2,
		Cols: 3,
		Cells: []*Cell{
			{Row: 0, Col: 0, Text: "A", RowSpan: 1, ColSpan: 1},
			{Row: 0, Col: 1, Text: "B", RowSpan: 1, ColSpan: 1},
			{Row: 0, Col: 2, Text: "C", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 0, Text: "1", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 1, Text: "2", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 2, Text: "3", RowSpan: 1, ColSpan: 1},
		},
	}

	result := table.Render()
	t.Logf("\n%s", result)

	checkAllLinesEqualWidth(t, result)
}

func TestMultilineCell(t *testing.T) {
	table := &Table{
		Rows: 2,
		Cols: 2,
		Cells: []*Cell{
			{Row: 0, Col: 0, Text: "A", RowSpan: 1, ColSpan: 1},
			{Row: 0, Col: 1, Text: "첫째줄\n둘째줄\n셋째줄", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 0, Text: "B", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 1, Text: "단일줄", RowSpan: 1, ColSpan: 1},
		},
	}

	result := table.Render()
	t.Logf("\n%s", result)

	checkAllLinesEqualWidth(t, result)

	// top border + 3 display rows + middle border + 1 display row + bottom border + trailing newline = 8 lines
	lines := strings.Split(result, "\n")
	if len(lines) != 8 {
		t.Errorf("Expected 8 lines, got %d", len(lines))
	}
}

func TestMultilineWithColSpan(t *testing.T) {
	table := &Table{
		Rows: 2,
		Cols: 3,
		Cells: []*Cell{
			{Row: 0, Col: 0, Text: "Header\nLine2\nLine3", RowSpan: 1, ColSpan: 2},
			{Row: 0, Col: 2, Text: "C", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 0, Text: "A", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 1, Text: "B", RowSpan: 1, ColSpan: 1},
			{Row: 1, Col: 2, Text: "C", RowSpan: 1, ColSpan: 1},
		},
	}

	result := table.Render()
	t.Logf("\n%s", result)

	checkAllLinesEqualWidth(t, result)
}

func TestMultilineWithRowSpan(t *testing.T) {
	table := &Table{
		Rows: 3,
		Cols: 2,
		Cells: []*Cell{
			{Row: 0, Col: 0, Text: "A", RowSpan: 1, ColSpan: 1},
			{Row: 0, Col: 1, Text: "Merged\n3\nrows", RowSpan: 3, ColSpan: 1},
			{Row: 1, Col: 0, Text: "B", RowSpan: 1, ColSpan: 1},
			{Row: 2, Col: 0, Text: "C", RowSpan: 1, ColSpan: 1},
		},
	}

	result := table.Render()
	t.Logf("\n%s", result)

	checkAllLinesEqualWidth(t, result)
}

func TestKoreanMultiline(t *testing.T) {
	table := &Table{
		Rows: 1,
		Cols: 2,
		Cells: []*Cell{
			{Row: 0, Col: 0, Text: "제목", RowSpan: 1, ColSpan: 1},
			{Row: 0, Col: 1, Text: "첫째 줄\n둘째 줄\n셋째 줄", RowSpan: 1, ColSpan: 1},
		},
	}

	result := table.Render()
	t.Logf("\n%s", result)

	checkAllLinesEqualWidth(t, result)
}

func checkAllLinesEqualWidth(t *testing.T, result string) {
	lines := strings.Split(result, "\n")
	var firstLineWidth int
	for i, line := range lines {
		if line == "" {
			continue
		}
		width := displayWidth(line)
		if i == 0 || (firstLineWidth == 0 && line != "") {
			firstLineWidth = width
		}
		if width != firstLineWidth {
			t.Errorf("Line %d has different display width: expected %d, got %d\nLine: %s", i, firstLineWidth, width, line)
		}
	}
}
