package parser

import "io"

type StreamReader struct {
	Reader       io.ReaderAt
	Size         int64
	ReaderOffset int64

	Sectors    []uint32
	SectorSize int64
}

func (self *StreamReader) ReadAt(buff []byte, off int64) (int, error) {
	current_sector := int(off / self.SectorSize)

	// Read past the end of the runlist
	if current_sector >= len(self.Sectors) || off >= self.Size {
		return 0, io.EOF
	}

	current_sector_offset := off % self.SectorSize
	current_buff_offset := 0

	for current_buff_offset <= len(buff) && current_sector < len(self.Sectors) {
		available_in_file := self.Size - (off + int64(current_buff_offset))
		available_in_sector := self.SectorSize - current_sector_offset
		available_in_buffer := int64(len(buff) - current_buff_offset)
		to_read := available_in_sector
		if to_read > available_in_buffer {
			to_read = available_in_buffer
		}

		if to_read > available_in_file {
			to_read = available_in_file
		}

		if to_read == 0 {
			break
		}

		current_sector_to_read := self.Sectors[current_sector]
		n, err := self.Reader.ReadAt(
			buff[current_buff_offset:int(to_read)+current_buff_offset],
			self.ReaderOffset+self.SectorSize*int64(current_sector_to_read)+
				current_sector_offset)
		if err != nil {
			return 0, err
		}

		// Prepare for the next sector to read
		current_sector++
		current_buff_offset += n
		current_sector_offset = 0
	}

	if current_buff_offset == 0 {
		return 0, io.EOF
	}

	return current_buff_offset, nil

}
