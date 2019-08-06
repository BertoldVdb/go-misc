package gpio

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func (g *Chip) readChipInfo() error {
	type chipInfoRaw struct {
		Name  [32]byte
		Label [32]byte
		Lines uint32
	}
	var ci chipInfoRaw

	err := ioctlPtr(g.file, gpioGetChipinfoIoctl, unsafe.Pointer(&ci))
	if err != nil {
		return err
	}

	g.chipInfo.Name = bytesToString(ci.Name[:])
	g.chipInfo.Label = bytesToString(ci.Label[:])
	g.chipInfo.Lines = ci.Lines

	return nil
}

func (g *Chip) readLineNames() error {
	names := make(map[string](uint32))

	for i := uint32(0); i < g.chipInfo.Lines; i++ {
		line, err := g.GetLineInfo(i)
		if err != nil {
			return err
		}

		names[line.Name] = i
	}

	g.lineNames = names

	return nil
}

func OpenChip(chip int) (*Chip, error) {
	g := &Chip{}

	var err error
	g.file, err = os.OpenFile(fmt.Sprintf("/dev/gpiochip%d", chip), syscall.O_RDWR|syscall.O_NOCTTY, 0600)

	if err != nil {
		return nil, err
	}

	err = g.readChipInfo()
	if err != nil {
		return nil, err
	}

	err = g.readLineNames()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *Chip) Close() error {
	return g.file.Close()
}

func (g *Chip) GetChipInfo() ChipInfo {
	return g.chipInfo
}

func (g *Chip) GetLineInfo(line uint32) (LineInfo, error) {
	result := LineInfo{
		LineOffset: line,
	}

	if result.LineOffset >= g.chipInfo.Lines {
		return result, errors.New("Line out of range")
	}

	type lineInfoRaw struct {
		LineOffset uint32
		Flags      uint32
		Name       [32]byte
		Consumer   [32]byte
	}

	li := lineInfoRaw{
		LineOffset: result.LineOffset,
	}

	err := ioctlPtr(g.file, gpioGetLineinfoIoctl, unsafe.Pointer(&li))
	if err != nil {
		return result, err
	}

	result.Flags = LineFlag(li.Flags)
	result.Name = bytesToString(li.Name[:])
	result.Consumer = bytesToString(li.Consumer[:])

	return result, nil
}

func (g *Chip) findLineByName(name string) (uint32, error) {
	if index, found := g.lineNames[name]; found {
		return index, nil
	}
	return 0, errors.New("Name not found")
}

func (g *Chip) OpenLine(label string, flags RequestFlag, line LineRequest) (*Lines, error) {
	return g.OpenLines(label, flags, []LineRequest{line})
}

func (g *Chip) OpenLines(label string, flags RequestFlag, lines []LineRequest) (*Lines, error) {
	if len(lines) > 64 || len(lines) == 0 {
		return nil, errors.New("Invalid number of lines")
	}

	type handleRequestRaw struct {
		LineOffsets   [64]uint32
		Flags         uint32
		DefaultValues [64]uint8
		ConsumerLabel [32]byte
		Lines         uint32
		Fd            int
	}

	req := handleRequestRaw{
		Flags: uint32(flags),
		Lines: uint32(len(lines)),
	}
	stringToBytes(label, req.ConsumerLabel[:])

	for i, l := range lines {
		if len(l.Line.Name) != 0 {
			off, err := g.findLineByName(l.Line.Name)
			if err != nil {
				return nil, err
			}

			req.LineOffsets[i] = off
		} else {
			req.LineOffsets[i] = l.Line.Offset
		}

		if req.LineOffsets[i] >= g.chipInfo.Lines {
			return nil, errors.New("Line out of range")
		}

		req.DefaultValues[i] = l.DefaultValue
	}

	err := ioctlPtr(g.file, gpioGetLinehandleIoctl, unsafe.Pointer(&req))
	if err != nil {
		return nil, err
	}

	if req.Fd <= 0 {
		return nil, errors.New("Invalid file descriptor returned")
	}

	gl := &Lines{
		file:     os.NewFile(uintptr(req.Fd), label),
		numLines: req.Lines,
	}

	return gl, nil
}

func (g *Chip) WatchLine(label string, requestFlags RequestFlag, eventFlags EventFlag, line Line) (*Lines, error) {
	type eventRequestRaw struct {
		LineOffset    uint32
		HandleFlags   uint32
		EventFlags    uint32
		ConsumerLabel [32]byte
		Fd            int
	}

	req := eventRequestRaw{
		HandleFlags: uint32(requestFlags),
		EventFlags:  uint32(eventFlags),
		LineOffset:  line.Offset,
	}
	stringToBytes(label, req.ConsumerLabel[:])

	if len(line.Name) != 0 {
		off, err := g.findLineByName(line.Name)
		if err != nil {
			return nil, err
		}

		req.LineOffset = off
	}

	if req.LineOffset >= g.chipInfo.Lines {
		return nil, errors.New("Line out of range")
	}

	err := ioctlPtr(g.file, gpioGetLineeventIoctl, unsafe.Pointer(&req))
	if err != nil {
		return nil, err
	}

	//TODO: This did not work on my hardware. I will check it later.

	if req.Fd <= 0 {
		return nil, errors.New("Invalid file descriptor returned")
	}

	return nil, nil
}
