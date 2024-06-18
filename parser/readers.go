package parser

import (
	"io"
	"os"
)

type ReadAdapter struct {
	io.ReaderAt
	offset int64
	size   int64
}

func (self *ReadAdapter) Read(buf []byte) (int, error) {
	if self.offset < 0 {
		return 0, io.EOF
	}

	n, err := self.ReadAt(buf, self.offset)
	self.offset += int64(n)
	return n, err
}

func (self *ReadAdapter) Seek(offset int64, whence int) (int64, error) {
	if whence == os.SEEK_SET {
		self.offset = offset
	}

	return self.offset, nil
}

func NewReadAdapter(reader io.ReaderAt) *ReadAdapter {
	return &ReadAdapter{ReaderAt: reader}
}
