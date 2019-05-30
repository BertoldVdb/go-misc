package gpio

import "os"

type Chip struct {
	file      *os.File
	chipInfo  ChipInfo
	lineNames map[string](uint32)
}

type ChipInfo struct {
	Name  string
	Label string
	Lines uint32
}

type Lines struct {
	file     *os.File
	numLines uint32
}

type LineInfo struct {
	LineOffset uint32
	Flags      LineFlag
	Name       string
	Consumer   string
}

type Line struct {
	Offset uint32
	Name   string
}

type LineRequest struct {
	Line         Line
	DefaultValue uint8
}
