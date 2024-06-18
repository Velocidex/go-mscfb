package parser

import (
	"fmt"
	"unsafe"
)

type headerFields struct {
	signature           uint64
	_                   [16]byte    //CLSID - ignore, must be null
	minorVersion        uint16      //Version number for non-breaking changes. This field SHOULD be set to 0x003E if the major version field is either 0x0003 or 0x0004.
	majorVersion        uint16      //Version number for breaking changes. This field MUST be set to either 0x0003 (version 3) or 0x0004 (version 4).
	_                   [2]byte     //byte order - ignore, must be little endian
	sectorSize          uint16      //This field MUST be set to 0x0009, or 0x000c, depending on the Major Version field. This field specifies the sector size of the compound file as a power of 2. If Major Version is 3, then the Sector Shift MUST be 0x0009, specifying a sector size of 512 bytes. If Major Version is 4, then the Sector Shift MUST be 0x000C, specifying a sector size of 4096 bytes.
	_                   [2]byte     // ministream sector size - ignore, must be 64 bytes
	_                   [6]byte     // reserved - ignore, not used
	numDirectorySectors uint32      //This integer field contains the count of the number of directory sectors in the compound file. If Major Version is 3, then the Number of Directory Sectors MUST be zero. This field is not supported for version 3 compound files.
	numFatSectors       uint32      //This integer field contains the count of the number of FAT sectors in the compound file.
	directorySectorLoc  uint32      //This integer field contains the starting sector number for the directory stream.
	_                   [4]byte     // transaction - ignore, not used
	_                   [4]byte     // mini stream size cutooff - ignore, must be 4096 bytes
	miniFatSectorLoc    uint32      //This integer field contains the starting sector number for the mini FAT.
	numMiniFatSectors   uint32      //This integer field contains the count of the number of mini FAT sectors in the compound file.
	difatSectorLoc      uint32      //This integer field contains the starting sector number for the DIFAT.
	numDifatSectors     uint32      //This integer field contains the count of the number of DIFAT sectors in the compound file.
	initialDifats       [109]uint32 //The first 109 difat sectors are included in the header
}

type DirectoryHeader_ struct {
	AB          [32]uint16
	CB          uint16
	Mse         byte
	Flags       byte
	SidLeftSib  uint32
	SidRightSib uint32
	SidChild    uint32
	ClsId       [16]byte
	UserFlags   uint32
	CreateTime  uint64
	ModifyTime  uint64
	SectStart   uint32
	Size        uint32
	PropType    uint16
}

func debugStruct() {
	a := &DirectoryHeader_{}
	fmt.Printf("Offset %v\n", unsafe.Offsetof(a.CreateTime))
}
