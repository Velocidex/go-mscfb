// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-cfb

package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var (
	notFoundError = errors.New("Not found")
)

type OLEContext struct {
	Reader  io.ReaderAt
	Profile *OLEProfile
	Header  *CFHeader

	SectorSize       uint32
	MiniSectorCutoff uint64
	MiniSectorSize   int64

	FatSectors []uint32

	Fat         []uint32
	MiniFat     []uint32
	Directories []DirectoryEntry

	Ministream io.ReaderAt
}

// Opens a stream using the fat from the first sector.
func (self *OLEContext) OpenStream(sector uint32) (*StreamReader, error) {
	sectors, err := self.getChain(sector)
	if err != nil {
		return nil, err
	}

	res := &StreamReader{
		Reader:       self.Reader,
		Size:         int64(self.SectorSize) * int64(len(sectors)),
		ReaderOffset: int64(self.SectorSize),
		SectorSize:   int64(self.SectorSize),
		Sectors:      sectors,
	}

	return res, nil
}

// Opens a DirectoryEntry stream based on its index. Automatically
// reads from the minifat if needed.
func (self *OLEContext) OpenDirectoryStream(idx uint64) (io.ReaderAt, int64, error) {
	dir_entry, err := self.GetDirentry(idx)
	if err != nil {
		return nil, 0, err
	}

	// This is a small stream so it comes out of the minifat.
	if dir_entry.Size < self.MiniSectorCutoff {
		res := make([]byte, 0, dir_entry.Size)
		buff := make([]byte, self.MiniSectorSize)

		sectors, err := self.getMiniChain(dir_entry.FirstSector)
		if err != nil {
			return nil, 0, err
		}

		for _, s := range sectors {
			n, err := self.Ministream.ReadAt(buff, int64(s)*self.MiniSectorSize)
			if err != nil {
				break
			}
			res = append(res, buff[:n]...)
		}

		if dir_entry.Size < uint64(len(res)) {
			res = res[:dir_entry.Size]
		}

		return bytes.NewReader(res), int64(dir_entry.Size), nil
	}

	res, err := self.OpenStream(dir_entry.FirstSector)
	if err != nil {
		return nil, 0, err
	}

	if dir_entry.Size > 0 && dir_entry.Size < uint64(res.Size) {
		res.Size = int64(dir_entry.Size)
	}

	return NewReadAdapter(res), res.Size, nil
}

func (self *OLEContext) Open(name string) (io.ReaderAt, *DirectoryEntry, error) {
	for idx, d := range self.Directories {
		if name == d.Name {
			reader, _, err := self.OpenDirectoryStream(uint64(idx))
			return reader, &d, err
		}
	}
	return nil, nil, errors.New("Not found error")
}

func (self *OLEContext) Stat(name string) (*DirectoryEntry, error) {
	for _, d := range self.Directories {
		if name == d.Name {
			return &d, nil
		}
	}
	return nil, errors.New("Not found error")
}

func (self *OLEContext) GetDirentry(idx uint64) (*DirectoryEntry, error) {
	if idx > uint64(len(self.Directories)) {
		return nil, errors.New("Invalid directory index")
	}

	return &self.Directories[idx], nil
}

func (self *OLEContext) ReadSector(sector uint32) ([]byte, error) {
	buf := make([]byte, self.SectorSize)
	n, err := self.Reader.ReadAt(buf, int64(self.SectorSize*(sector+1)))
	return buf[:n], err
}

// Get the list of sector chains from the fat
func (self *OLEContext) getChain(start uint32) (res []uint32, err error) {
	seen := make(map[uint32]bool)
	for sector := start; sector != ENDOFCHAIN; {
		res = append(res, sector)
		if int(sector) > len(self.Fat) {
			return res, fmt.Errorf("Fat reference exceeded: %v", res)
		}
		next := self.Fat[sector]
		_, pres := seen[next]
		if pres {
			return res, fmt.Errorf("Fat cycle detected: %v", res)
		}

		// Do not allow extremely long streams, just truncate them.
		if len(res) > MAX_SECTORS {
			break
		}

		sector = next
	}
	return res, nil
}

func (self *OLEContext) getMiniChain(start uint32) (res []uint32, err error) {
	seen := make(map[uint32]bool)
	for sector := start; sector != ENDOFCHAIN; {
		res = append(res, sector)
		if int(sector) > len(self.MiniFat) {
			return res, fmt.Errorf("MiniFat reference exceeded: %v", res)
		}
		next := self.MiniFat[sector]
		_, pres := seen[next]
		if pres {
			return res, fmt.Errorf("MiniFat cycle detected: %v", res)
		}

		// Do not allow extremely long streams, just truncate them.
		if len(res) > MAX_SECTORS {
			break
		}

		sector = next
	}
	return res, nil
}

func (self *OLEContext) loadDIFSectors() error {
	// load any DIF sectors
	sector := self.Header.DIFATSectorLoc()
	seen := make(map[uint32]bool)
	for sector != FREESECT && sector != ENDOFCHAIN {
		sector_data, err := self.ReadSector(sector)
		if err != nil || len(sector_data) < int(self.SectorSize) {
			// Must be a full sector
			return err
		}
		dif_values := make([]uint32, self.SectorSize/4)
		err = binary.Read(bytes.NewReader(sector_data),
			binary.LittleEndian, dif_values)
		if err != nil {
			return err
		}

		// the last entry is actually a pointer to next DIF
		next := dif_values[len(dif_values)-1]
		for _, value := range dif_values[:len(dif_values)-2] {
			if value != FREESECT {
				self.FatSectors = append(self.FatSectors, value)
			}
		}

		_, pres := seen[next]
		if pres || len(seen) > MAX_SECTORS {
			return fmt.Errorf(
				"infinite loop detected at %v to %v starting at DIF",
				sector, next)
		}

		seen[next] = true
		sector = next
	}
	return nil
}

func (self *OLEContext) loadFat() error {
	for _, fat_sect := range self.FatSectors {
		sect_data, err := self.ReadSector(fat_sect)
		if err != nil {
			return err
		}

		sect_longs := make([]uint32, self.SectorSize/4)
		buffer := bytes.NewBuffer(sect_data)
		err = binary.Read(buffer, binary.LittleEndian, sect_longs)
		if err != nil {
			return err
		}
		self.Fat = append(self.Fat, sect_longs...)
	}
	return nil
}

func (self *OLEContext) loadMiniFat() (err error) {
	dir_entry, err := self.GetDirentry(0)
	if err != nil {
		return err
	}

	ministream, err := self.OpenStream(dir_entry.FirstSector)
	if err != nil {
		return err
	}

	if dir_entry.Size > 0 && dir_entry.Size < uint64(ministream.Size) {
		ministream.Size = int64(dir_entry.Size)
	}

	self.Ministream = ministream

	fat_stream, err := self.OpenStream(self.Header.MiniFATSectorLoc())
	if err != nil {
		return err
	}

	buffer := make([]byte, self.SectorSize)

	for offset := int64(0); offset < fat_stream.Size; offset += int64(self.SectorSize) {
		n, err := fat_stream.ReadAt(buffer, offset)
		if err != nil || n == 0 {
			break
		}

		fats := make([]uint32, n/4)
		err = binary.Read(bytes.NewBuffer(buffer[:n]),
			binary.LittleEndian, &fats)
		if err != nil {
			break
		}
		self.MiniFat = append(self.MiniFat, fats...)
	}
	return nil
}

func (self *OLEContext) loadDirectories() error {
	dir_stream, err := self.OpenStream(self.Header.DirectorySectorLoc())
	if err != nil {
		return err
	}

	self.Directories = nil
	count := 0
	for offset := int64(0); offset < dir_stream.Size; count++ {
		dir := self.Profile.DirectoryHeader(dir_stream, offset)
		name := strings.Split(dir.Name(), "\x00")[0]

		self.Directories = append(self.Directories, DirectoryEntry{
			Name:        name,
			Id:          int64(count),
			Size:        dir.StreamSize(),
			Ctime:       time.Unix(int64(dir.CreateTime())/10000000-11644473600, 0),
			Mtime:       time.Unix(int64(dir.ModifyTime())/10000000-11644473600, 0),
			IsDir:       dir.TypeInt() == 1 || dir.TypeInt() == 5,
			FirstSector: dir.SectorStart(),
		})
		offset += int64(dir.Size())
	}

	return nil
}

func (self *OLEContext) DebugString() string {
	result := self.Header.DebugString()
	result += "\n Directories: \n"
	for _, d := range self.Directories {
		result += Dump(d)
	}
	return result
}

// For now treat all directories as present at the top level.
func (self *OLEContext) ListDirectory(path string) ([]DirectoryEntry, error) {
	components := GetComponents(path)
	return self.ListDirectoryComponents(components)
}

func (self *OLEContext) ListDirectoryComponents(
	components []string) ([]DirectoryEntry, error) {

	if len(components) > 0 {
		return nil, nil
	}

	return self.Directories, nil
}

func GetOLEContext(reader io.ReaderAt) (*OLEContext, error) {
	profile := NewOLEProfile()
	self := &OLEContext{
		Reader:     reader,
		Profile:    profile,
		SectorSize: 0,
		Header:     profile.CFHeader(reader, 0),
	}

	if self.Header.Signature() != 0xe11ab1a1e011cfd0 {
		return nil, errors.New("Invalid file signature")
	}

	self.MiniSectorCutoff = uint64(self.Header.MiniSectorCutoff())
	self.MiniSectorSize = 1 << self.Header.MiniSectorShift()

	if self.MiniSectorSize != 64 {
		return nil, fmt.Errorf("Invalid MiniSectorSize %v", self.MiniSectorSize)
	}

	if self.MiniSectorCutoff != 4096 {
		return nil, fmt.Errorf("Invalid MiniSectorCutoff %v", self.MiniSectorCutoff)
	}

	switch self.Header.SectorSize() {
	case 0x9:
		self.SectorSize = 512
	case 0xc:
		self.SectorSize = 4096
	default:
		return nil, errors.New("Sector size invalid")
	}

	for _, e := range self.Header.InitialDIFATs() {
		if e != FREESECT {
			self.FatSectors = append(self.FatSectors, e)
		}
	}

	err := self.loadDIFSectors()
	if err != nil {
		return nil, err
	}

	err = self.loadFat()
	if err != nil {
		return nil, err
	}

	err = self.loadDirectories()
	if err != nil {
		return nil, err
	}

	err = self.loadMiniFat()
	if err != nil {
		return nil, err
	}

	return self, nil
}
