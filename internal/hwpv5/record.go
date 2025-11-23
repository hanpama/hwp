package hwpv5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	recTagBegin                   = 0x10
	recTagParaHeader              = recTagBegin + 50
	recTagParaText                = recTagBegin + 51
	recTagParaCharShape           = recTagBegin + 52
	recTagParaLineSeg             = recTagBegin + 53
	recTagParaRangeTag            = recTagBegin + 54
	recTagCtrlHeader              = recTagBegin + 55
	recTagListHeader              = recTagBegin + 56
	recTagPageDef                 = recTagBegin + 57
	recTagFootnoteShape           = recTagBegin + 58
	recTagPageBorderFill          = recTagBegin + 59
	recTagShapeComponent          = recTagBegin + 60
	recTagTable                   = recTagBegin + 61
	recTagShapeComponentLine      = recTagBegin + 62
	recTagShapeComponentRectangle = recTagBegin + 63
	recTagShapeComponentEllipse   = recTagBegin + 64
	recTagShapeComponentArc       = recTagBegin + 65
	recTagShapeComponentPolygon   = recTagBegin + 66
	recTagShapeComponentCurve     = recTagBegin + 67
	recTagShapeComponentOLE       = recTagBegin + 68
	recTagShapeComponentPicture   = recTagBegin + 69
	recTagShapeComponentContainer = recTagBegin + 70
	recTagCtrlData                = recTagBegin + 71
	recTagEqEdit                  = recTagBegin + 72
	recTagShapeComponentTextArt   = recTagBegin + 74
	recTagFormObject              = recTagBegin + 75
	recTagMemoShape               = recTagBegin + 76
	recTagMemoList                = recTagBegin + 77
	recTagChartData               = recTagBegin + 79
	recTagVideoData               = recTagBegin + 82
	recTagShapeComponentUnknown   = recTagBegin + 99
)

// recHeader holds the common metadata shared by all concrete record nodes.
type recHeader struct {
	TagID uint16
	Level uint16
	Size  uint32
}

// Rec represents a typed record.
type Rec interface {
	Tag() uint16
	Lvl() uint16
	Len() uint32
}

func (b recHeader) Tag() uint16 { return b.TagID }
func (b recHeader) Lvl() uint16 { return b.Level }
func (b recHeader) Len() uint32 { return b.Size }

// Body record concrete types (payloads are intentionally empty scaffolds).
type (
	RecParaHeader struct{ recHeader }
	RecParaText   struct {
		recHeader
		Els []ParaTextElement
	}
	RecParaCharShape struct{ recHeader }
	RecParaLineSeg   struct{ recHeader }
	RecParaRangeTag  struct{ recHeader }
	RecCtrlHeader    struct {
		recHeader
		CtrlID uint32
		Data   []byte
	}
	RecListHeader struct {
		recHeader
		IsCell    bool
		ParaCount int16
		Property  uint32
		ColIndex  uint16
		RowIndex  uint16
		ColSpan   uint16
		RowSpan   uint16
	}
	RecPageDef        struct{ recHeader }
	RecFootnoteShape  struct{ recHeader }
	RecPageBorderFill struct{ recHeader }
	RecShapeComponent struct{ recHeader }
	RecTable          struct {
		recHeader
		Data     []byte
		RowCount uint16
		ColCount uint16
	}
	RecShapeComponentLine      struct{ recHeader }
	RecShapeComponentRectangle struct{ recHeader }
	RecShapeComponentEllipse   struct{ recHeader }
	RecShapeComponentArc       struct{ recHeader }
	RecShapeComponentPolygon   struct{ recHeader }
	RecShapeComponentCurve     struct{ recHeader }
	RecShapeComponentOLE       struct{ recHeader }
	RecShapeComponentPicture   struct{ recHeader }
	RecShapeComponentContainer struct{ recHeader }
	RecCtrlData                struct{ recHeader }
	RecEqEdit                  struct{ recHeader }
	RecShapeComponentTextArt   struct{ recHeader }
	RecFormObject              struct{ recHeader }
	RecMemoShape               struct{ recHeader }
	RecMemoList                struct{ recHeader }
	RecChartData               struct{ recHeader }
	RecVideoData               struct{ recHeader }
	RecShapeComponentUnknown   struct{ recHeader }

	// RecUnknown keeps the raw payload when no concrete type is defined.
	RecUnknown struct {
		recHeader
		Data []byte
	}
)

// RecScanner consumes a stream of records and yields them sequentially.
type RecScanner struct {
	r io.Reader
}

func NewRecScanner(r io.Reader) *RecScanner {
	return &RecScanner{r: r}
}

func (s *RecScanner) ScanNext() (Rec, error) {
	var headerRaw uint32
	if err := binary.Read(s.r, binary.LittleEndian, &headerRaw); err != nil {
		return nil, err
	}

	base := recHeader{
		TagID: uint16(headerRaw & 0x3ff),
		Level: uint16((headerRaw >> 10) & 0x3ff),
		Size:  uint32((headerRaw >> 20) & 0xfff),
	}
	if base.Size == 0xfff {
		if err := binary.Read(s.r, binary.LittleEndian, &base.Size); err != nil {
			return nil, fmt.Errorf("read extended size: %w", err)
		}
	}

	data := make([]byte, base.Size)
	if _, err := io.ReadFull(s.r, data); err != nil {
		return nil, fmt.Errorf("read record data: %w", err)
	}

	switch base.TagID {
	case recTagParaHeader:
		return s.decodeParaHeaderRecord(base, data)
	case recTagParaText:
		return s.decodeParaTextRecord(base, data)
	case recTagParaCharShape:
		return s.decodeParaCharShapeRecord(base, data)
	case recTagParaLineSeg:
		return s.decodeParaLineSegRecord(base, data)
	case recTagParaRangeTag:
		return s.decodeParaRangeTagRecord(base, data)
	case recTagCtrlHeader:
		return s.decodeCtrlHeaderRecord(base, data)
	case recTagListHeader:
		return s.decodeListHeaderRecord(base, data)
	case recTagPageDef:
		return s.decodePageDefRecord(base, data)
	case recTagFootnoteShape:
		return s.decodeFootnoteShapeRecord(base, data)
	case recTagPageBorderFill:
		return s.decodePageBorderFillRecord(base, data)
	case recTagShapeComponent:
		return s.decodeShapeComponentRecord(base, data)
	case recTagTable:
		return s.decodeTableRecord(base, data)
	case recTagShapeComponentLine:
		return s.decodeShapeComponentLineRecord(base, data)
	case recTagShapeComponentRectangle:
		return s.decodeShapeComponentRectangleRecord(base, data)
	case recTagShapeComponentEllipse:
		return s.decodeShapeComponentEllipseRecord(base, data)
	case recTagShapeComponentArc:
		return s.decodeShapeComponentArcRecord(base, data)
	case recTagShapeComponentPolygon:
		return s.decodeShapeComponentPolygonRecord(base, data)
	case recTagShapeComponentCurve:
		return s.decodeShapeComponentCurveRecord(base, data)
	case recTagShapeComponentOLE:
		return s.decodeShapeComponentOLERecord(base, data)
	case recTagShapeComponentPicture:
		return s.decodeShapeComponentPictureRecord(base, data)
	case recTagShapeComponentContainer:
		return s.decodeShapeComponentContainerRecord(base, data)
	case recTagCtrlData:
		return s.decodeCtrlDataRecord(base, data)
	case recTagEqEdit:
		return s.decodeEqEditRecord(base, data)
	case recTagShapeComponentTextArt:
		return s.decodeShapeComponentTextArtRecord(base, data)
	case recTagFormObject:
		return s.decodeFormObjectRecord(base, data)
	case recTagMemoShape:
		return s.decodeMemoShapeRecord(base, data)
	case recTagMemoList:
		return s.decodeMemoListRecord(base, data)
	case recTagChartData:
		return s.decodeChartDataRecord(base, data)
	case recTagVideoData:
		return s.decodeVideoDataRecord(base, data)
	case recTagShapeComponentUnknown:
		return s.decodeShapeComponentUnknownRecord(base, data)
	default:
		return RecUnknown{recHeader: base, Data: data}, nil
	}
}

func (s *RecScanner) decodeParaHeaderRecord(b recHeader, _ []byte) (Rec, error) {
	return RecParaHeader{b}, nil
}

func (s *RecScanner) decodeParaTextRecord(b recHeader, data []byte) (Rec, error) {
	d := &paraTextDecoder{data: bytes.NewReader(data)}
	return RecParaText{recHeader: b, Els: d.decodeParaTextElements()}, nil
}

func (s *RecScanner) decodeParaCharShapeRecord(b recHeader, _ []byte) (Rec, error) {
	return RecParaCharShape{b}, nil
}

func (s *RecScanner) decodeParaLineSegRecord(b recHeader, _ []byte) (Rec, error) {
	return RecParaLineSeg{b}, nil
}

func (s *RecScanner) decodeParaRangeTagRecord(b recHeader, _ []byte) (Rec, error) {
	return RecParaRangeTag{b}, nil
}

func (s *RecScanner) decodeCtrlHeaderRecord(b recHeader, data []byte) (Rec, error) {
	rec := RecCtrlHeader{recHeader: b, Data: data}
	if len(data) >= 4 {
		rec.CtrlID = binary.LittleEndian.Uint32(data[:4])
	}
	return rec, nil
}

func (s *RecScanner) decodeListHeaderRecord(b recHeader, data []byte) (Rec, error) {
	rec := RecListHeader{recHeader: b}
	if len(data) >= 6 {
		rec.ParaCount = int16(binary.LittleEndian.Uint16(data[0:]))
		rec.Property = binary.LittleEndian.Uint32(data[2:])
	}
	// Cell list = LIST_HEADER (6 bytes) + Cell properties (26 bytes) = 32 bytes total
	// But in practice we need 33 bytes based on old hwp3 code
	if len(data) >= 33 {
		rec.IsCell = true

		// Cell data format (from old hwp3 implementation):
		// Uses byte offsets at positions 7+1, 7+3, 7+5, 7+7
		cellData := data[7:33]
		if len(cellData) >= 8 {
			rec.ColIndex = uint16(cellData[1])
			rec.RowIndex = uint16(cellData[3])
			rec.ColSpan = uint16(cellData[5])
			rec.RowSpan = uint16(cellData[7])

			if rec.ColSpan == 0 {
				rec.ColSpan = 1
			}
			if rec.RowSpan == 0 {
				rec.RowSpan = 1
			}
		}
	}
	return rec, nil
}

func (s *RecScanner) decodePageDefRecord(b recHeader, _ []byte) (Rec, error) {
	return RecPageDef{b}, nil
}

func (s *RecScanner) decodeFootnoteShapeRecord(b recHeader, _ []byte) (Rec, error) {
	return RecFootnoteShape{b}, nil
}

func (s *RecScanner) decodePageBorderFillRecord(b recHeader, _ []byte) (Rec, error) {
	return RecPageBorderFill{b}, nil
}

func (s *RecScanner) decodeShapeComponentRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponent{b}, nil
}

func (s *RecScanner) decodeTableRecord(b recHeader, data []byte) (Rec, error) {
	rec := RecTable{recHeader: b, Data: data}
	if len(data) >= 8 {
		rec.RowCount = binary.LittleEndian.Uint16(data[4:])
		rec.ColCount = binary.LittleEndian.Uint16(data[6:])
	}
	return rec, nil
}

func (s *RecScanner) decodeShapeComponentLineRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentLine{b}, nil
}

func (s *RecScanner) decodeShapeComponentRectangleRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentRectangle{b}, nil
}

func (s *RecScanner) decodeShapeComponentEllipseRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentEllipse{b}, nil
}

func (s *RecScanner) decodeShapeComponentArcRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentArc{b}, nil
}

func (s *RecScanner) decodeShapeComponentPolygonRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentPolygon{b}, nil
}

func (s *RecScanner) decodeShapeComponentCurveRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentCurve{b}, nil
}

func (s *RecScanner) decodeShapeComponentOLERecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentOLE{b}, nil
}

func (s *RecScanner) decodeShapeComponentPictureRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentPicture{b}, nil
}

func (s *RecScanner) decodeShapeComponentContainerRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentContainer{b}, nil
}

func (s *RecScanner) decodeCtrlDataRecord(b recHeader, _ []byte) (Rec, error) {
	return RecCtrlData{b}, nil
}

func (s *RecScanner) decodeEqEditRecord(b recHeader, _ []byte) (Rec, error) {
	return RecEqEdit{b}, nil
}

func (s *RecScanner) decodeShapeComponentTextArtRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentTextArt{b}, nil
}

func (s *RecScanner) decodeFormObjectRecord(b recHeader, _ []byte) (Rec, error) {
	return RecFormObject{b}, nil
}

func (s *RecScanner) decodeMemoShapeRecord(b recHeader, _ []byte) (Rec, error) {
	return RecMemoShape{b}, nil
}

func (s *RecScanner) decodeMemoListRecord(b recHeader, _ []byte) (Rec, error) {
	return RecMemoList{b}, nil
}

func (s *RecScanner) decodeChartDataRecord(b recHeader, _ []byte) (Rec, error) {
	return RecChartData{b}, nil
}

func (s *RecScanner) decodeVideoDataRecord(b recHeader, _ []byte) (Rec, error) {
	return RecVideoData{b}, nil
}

func (s *RecScanner) decodeShapeComponentUnknownRecord(b recHeader, _ []byte) (Rec, error) {
	return RecShapeComponentUnknown{b}, nil
}
