package hwpv5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const signatureText = "HWP Document File"

// Version stores the four-part HWP version number (MM.nn.PP.rr).
type Version struct {
	Major byte
	Minor byte
	Patch byte
	Rev   byte
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Rev)
}

// FileProperties exposes a few frequently used flags from the FileHeader stream.
type FileProperties struct {
	Raw uint32
}

func (p FileProperties) Compressed() bool { return p.Raw&0x1 != 0 }
func (p FileProperties) Encrypted() bool  { return p.Raw&0x2 != 0 }

// FileHeader mirrors the 256-byte FileHeader stream.
type FileHeader struct {
	Signature       string
	Version         Version
	Properties      FileProperties
	SecondFlags     uint32
	EncryptVersion  uint32
	KoglLicenseCode byte
	Reserved        [207]byte
}

func readFileHeader(r io.Reader) (FileHeader, error) {
	var hdr FileHeader

	var sig [32]byte
	if _, err := io.ReadFull(r, sig[:]); err != nil {
		return hdr, fmt.Errorf("read signature: %w", err)
	}
	hdr.Signature = string(bytes.TrimRight(sig[:], "\x00"))
	if hdr.Signature != signatureText {
		return hdr, fmt.Errorf("unexpected signature %q", hdr.Signature)
	}

	var ver uint32
	if err := binary.Read(r, binary.LittleEndian, &ver); err != nil {
		return hdr, fmt.Errorf("read version: %w", err)
	}
	hdr.Version = Version{
		Major: byte(ver >> 24),
		Minor: byte(ver >> 16),
		Patch: byte(ver >> 8),
		Rev:   byte(ver),
	}

	if err := binary.Read(r, binary.LittleEndian, &hdr.Properties.Raw); err != nil {
		return hdr, fmt.Errorf("read properties: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &hdr.SecondFlags); err != nil {
		return hdr, fmt.Errorf("read second properties: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &hdr.EncryptVersion); err != nil {
		return hdr, fmt.Errorf("read encrypt version: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &hdr.KoglLicenseCode); err != nil {
		return hdr, fmt.Errorf("read kogl: %w", err)
	}
	if _, err := io.ReadFull(r, hdr.Reserved[:]); err != nil {
		return hdr, fmt.Errorf("read reserved: %w", err)
	}
	return hdr, nil
}
