package bidirpipe

import (
	"io"
)

type PipeReadWriteCloser struct {
	writer *io.PipeWriter
	reader *io.PipeReader
}

func (p *PipeReadWriteCloser) Close() error {
	p.writer.Close()
	p.reader.Close()

	return nil
}

func (p *PipeReadWriteCloser) Read(buf []byte) (int, error) {
	return p.reader.Read(buf)
}

func (p *PipeReadWriteCloser) Write(buf []byte) (int, error) {
	return p.writer.Write(buf)
}

func CreateBidirPipe() (io.ReadWriteCloser, io.ReadWriteCloser) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	part1 := &PipeReadWriteCloser{}
	part1.writer = w1
	part1.reader = r2

	part2 := &PipeReadWriteCloser{}
	part2.writer = w2
	part2.reader = r1

	return part1, part2
}
