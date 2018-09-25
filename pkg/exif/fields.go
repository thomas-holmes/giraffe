package exif

import (
	"encoding/binary"
	"io"
	"log"
)

func ValFromTagID(r io.Reader, tagID uint16) (interface{}, bool) {
	switch tagID {
	case 0x100: // Image Width
		return readUint32(r), true
	case 0x101: // Image Height
		return readUint32(r), true
	case 0xc000: // Fuji RAF data https://www.sno.phy.queensu.ca/~phil/exiftool/TagNames/FujiFilm.html#RAFData
		return nil, false
	default:
		return nil, false
	}
}

func readUint32(r io.Reader) interface{} {
	dat := make([]byte, 4)

	_, err := r.Read(dat)
	if err != nil {
		log.Panicln(err)
	}

	return binary.BigEndian.Uint32(dat)
}
