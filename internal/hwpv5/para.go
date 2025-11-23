package hwpv5

import (
	"encoding/binary"
	"io"
)

const (
	// Unusable range (0)
	paraTextCodeUnusable uint16 = 0
	// Reserved (1)
	paraTextCodeReserved1 uint16 = 1
	// Extended Controls (require additional records like CTRL_HEADER)
	paraTextCodeSectionColDef uint16 = 2 // 구역 정의/단 정의
	paraTextCodeFieldStart    uint16 = 3 // 필드 시작 (누름틀, 하이퍼링크 등)
	paraTextCodeFieldEnd      uint16 = 4 // 필드 끝 (Inline)
	// Reserved (5-7)
	paraTextCodeReserved5 uint16 = 5
	paraTextCodeReserved6 uint16 = 6
	paraTextCodeReserved7 uint16 = 7
	// Inline Controls
	paraTextCodeTitleMark uint16 = 8 // Title mark
	paraTextCodeTab       uint16 = 9 // 탭 (Tab)
	// Char Controls
	paraTextCodeLineBreak uint16 = 10 // 한 줄 끝 (Line break)
	// Extended Controls (Most common for Objects)
	paraTextCodeGsoTable uint16 = 11 // 그리기 개체/표 (Drawing Object/Table)
	// Reserved (12)
	paraTextCodeReserved12 uint16 = 12 //
	// Char Controls
	paraTextCodeParaBreak uint16 = 13 // 문단 끝 (Para break) - 보통 문단 끝에는 저장되지 않음
	// Extended Controls
	paraTextCodeReserved14      uint16 = 14 // 예약
	paraTextCodeHiddenComment   uint16 = 15 // 숨은 설명
	paraTextCodeHeaderFooter    uint16 = 16 // 머리말/꼬리말
	paraTextCodeFootnoteEndnote uint16 = 17 // 각주/미주
	paraTextCodeAutoNumber      uint16 = 18 // 자동번호 (각주, 표 등)
	// Reserved Inline Controls (19-20)
	paraTextCodeReserved19 uint16 = 19
	paraTextCodeReserved20 uint16 = 20
	// Extended Controls
	paraTextCodePageControl    uint16 = 21 // 페이지 컨트롤 (감추기, 새 번호로 시작 등)
	paraTextCodeBookmarkIndex  uint16 = 22 // 책갈피/찾아보기 표식
	paraTextCodeAddTextOverlap uint16 = 23 // 덧말/글자 겹침
	// Char Controls
	paraTextCodeHyphen uint16 = 24 // 하이픈
	// Reserved Char Controls (25-29)
	paraTextCodeReserved25 uint16 = 25
	paraTextCodeReserved26 uint16 = 26
	paraTextCodeReserved27 uint16 = 27
	paraTextCodeReserved28 uint16 = 28
	paraTextCodeReserved29 uint16 = 29
	// Char Controls
	paraTextCodeBundleSpace uint16 = 30 // 묶음 빈칸
	paraTextCodeFixedSpace  uint16 = 31 // 고정폭 빈칸
)

type ParaTextElement interface {
	isParaTextElement()
}

type paraTextBase struct {
	Code uint16
}

func (p paraTextBase) isParaTextElement() {}

type (
	ParaTextString struct {
		paraTextBase
		Value string
	}
	ParaTextSectionColDef   struct{ paraTextBase }
	ParaTextFieldStart      struct{ paraTextBase }
	ParaTextFieldEnd        struct{ paraTextBase }
	ParaTextTitleMark       struct{ paraTextBase }
	ParaTextTab             struct{ paraTextBase }
	ParaTextLineBreak       struct{ paraTextBase }
	ParaTextGsoTable        struct{ paraTextBase }
	ParaTextParaBreak       struct{ paraTextBase }
	ParaTextHiddenComment   struct{ paraTextBase }
	ParaTextHeaderFooter    struct{ paraTextBase }
	ParaTextFootnoteEndnote struct{ paraTextBase }
	ParaTextAutoNumber      struct{ paraTextBase }
	ParaTextPageControl     struct{ paraTextBase }
	ParaTextBookmarkIndex   struct{ paraTextBase }
	ParaTextAddTextOverlap  struct{ paraTextBase }
	ParaTextHyphen          struct{ paraTextBase }
	ParaTextBundleSpace     struct{ paraTextBase }
	ParaTextFixedSpace      struct{ paraTextBase }
)

type paraTextDecoder struct {
	data io.Reader
}

func (d *paraTextDecoder) decodeParaTextElements() []ParaTextElement {
	var elements []ParaTextElement
	var stringBuffer []rune

	flushString := func() {
		if len(stringBuffer) > 0 {
			elements = append(elements, ParaTextString{
				paraTextBase: paraTextBase{Code: 0},
				Value:        string(stringBuffer),
			})
			stringBuffer = stringBuffer[:0]
		}
	}

	for {
		var code uint16
		if err := binary.Read(d.data, binary.LittleEndian, &code); err != nil {
			break
		}

		if code >= 32 {
			stringBuffer = append(stringBuffer, rune(code))
			continue
		}

		flushString()

		switch code {
		case paraTextCodeUnusable:
		case paraTextCodeReserved1:

		// === Extended Controls (8 WCHAR = 16 bytes) ===
		case paraTextCodeSectionColDef:
			d.skipBytes(14)
			elements = append(elements, ParaTextSectionColDef{paraTextBase{code}})

		case paraTextCodeFieldStart:
			d.skipBytes(14)
			elements = append(elements, ParaTextFieldStart{paraTextBase{code}})

		// === Inline Controls (8 WCHAR = 16 bytes) ===
		case paraTextCodeFieldEnd:
			d.skipBytes(14)
			elements = append(elements, ParaTextFieldEnd{paraTextBase{code}})

		case paraTextCodeReserved5, paraTextCodeReserved6, paraTextCodeReserved7:
			d.skipBytes(14)

		case paraTextCodeTitleMark:
			d.skipBytes(14)
			elements = append(elements, ParaTextTitleMark{paraTextBase{code}})

		case paraTextCodeTab:
			d.skipBytes(14)
			elements = append(elements, ParaTextTab{paraTextBase{code}})

		// === Char Controls (1 WCHAR = 2 bytes) ===
		case paraTextCodeLineBreak:
			elements = append(elements, ParaTextLineBreak{paraTextBase{code}})

		// === Extended Controls ===
		case paraTextCodeGsoTable:
			d.skipBytes(14)
			elements = append(elements, ParaTextGsoTable{paraTextBase{code}})

		case paraTextCodeReserved12:
			d.skipBytes(14)

		case paraTextCodeParaBreak:
			elements = append(elements, ParaTextParaBreak{paraTextBase{code}})

		case paraTextCodeReserved14:
			d.skipBytes(14)

		case paraTextCodeHiddenComment:
			d.skipBytes(14)
			elements = append(elements, ParaTextHiddenComment{paraTextBase{code}})

		case paraTextCodeHeaderFooter:
			d.skipBytes(14)
			elements = append(elements, ParaTextHeaderFooter{paraTextBase{code}})

		case paraTextCodeFootnoteEndnote:
			d.skipBytes(14)
			elements = append(elements, ParaTextFootnoteEndnote{paraTextBase{code}})

		case paraTextCodeAutoNumber:
			d.skipBytes(14)
			elements = append(elements, ParaTextAutoNumber{paraTextBase{code}})

		case paraTextCodeReserved19, paraTextCodeReserved20:
			d.skipBytes(14)

		case paraTextCodePageControl:
			d.skipBytes(14)
			elements = append(elements, ParaTextPageControl{paraTextBase{code}})

		case paraTextCodeBookmarkIndex:
			d.skipBytes(14)
			elements = append(elements, ParaTextBookmarkIndex{paraTextBase{code}})

		case paraTextCodeAddTextOverlap:
			d.skipBytes(14)
			elements = append(elements, ParaTextAddTextOverlap{paraTextBase{code}})

		case paraTextCodeHyphen:
			elements = append(elements, ParaTextHyphen{paraTextBase{code}})

		case paraTextCodeReserved25, paraTextCodeReserved26, paraTextCodeReserved27,
			paraTextCodeReserved28, paraTextCodeReserved29:

		case paraTextCodeBundleSpace:
			elements = append(elements, ParaTextBundleSpace{paraTextBase{code}})

		case paraTextCodeFixedSpace:
			elements = append(elements, ParaTextFixedSpace{paraTextBase{code}})
		}
	}

	flushString()
	return elements
}

func (d *paraTextDecoder) skipBytes(n int) {
	io.CopyN(io.Discard, d.data, int64(n))
}
