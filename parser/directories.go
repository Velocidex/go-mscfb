package parser

import (
	"time"
)

type DirectoryEntry struct {
	Name        string    `json:"Name"`
	Id          int64     `json:"Id"`
	Size        uint64    `json:"Size"`
	Mtime       time.Time `json:"Mtime"`
	Ctime       time.Time `json:"Ctime"`
	Attribute   string    `json:"Attribute"`
	IsDir       bool      `json:"IsDir"`
	FirstSector uint32    `json:"FirstSector"`
}
