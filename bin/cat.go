package main

import (
	"io"
	"os"

	"github.com/Velocidex/go-mscfb/parser"
	kingpin "github.com/alecthomas/kingpin/v2"
	ntfs_parser "www.velocidex.com/golang/go-ntfs/parser"
)

var (
	cat_command = app.Command(
		"cat", "Dump stream.")

	cat_command_file_arg = cat_command.Arg(
		"file", "The image file to inspect",
	).Required().OpenFile(os.O_RDONLY, os.FileMode(0666))

	cat_command_first_sector = cat_command.Flag(
		"sector", "If specified we treat the Id as first sector",
	).Bool()

	cat_command_arg = cat_command.Arg(
		"id", "The stream id to dump",
	).Default("0").Uint64()

	cat_command_image_offset = cat_command.Flag(
		"image_offset", "An offset into the file.",
	).Default("0").Int64()
)

func doCat() {
	reader, _ := ntfs_parser.NewPagedReader(
		getReader(*cat_command_file_arg), 1024, 10000)

	ole, err := parser.GetOLEContext(reader)
	kingpin.FatalIfError(err, "Can not open filesystem")

	if *cat_command_first_sector {
		first_sector := uint32(*cat_command_arg)
		stream, err := ole.OpenStream(first_sector)
		kingpin.FatalIfError(err, "Can not open stream")

		io.CopyN(os.Stdout, parser.NewReadAdapter(stream), stream.Size)
		return
	}

	stream, size, err := ole.OpenDirectoryStream(uint64(*cat_command_arg))
	kingpin.FatalIfError(err, "Can not open stream")

	io.CopyN(os.Stdout, parser.NewReadAdapter(stream), size)
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case "cat":
			doCat()
		default:
			return false
		}
		return true
	})
}
