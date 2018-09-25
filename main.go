package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

// 8922 expected raw resolution of 6000x4000, adobe rgb, depth 8bit, 300ppi

func main() {
	log.Println("Giraffe loves pictures 🦒")

	log.Println("inPath:", inPath)
	log.Println("outPath:", outPath)

	file, err := os.Open(inPath)
	if err != nil {
		log.Panicln(err)
	}

	rawHeader, err := ReadRawHeader(file)
	if err != nil {
		log.Panicln(err)
	}
	log.Println(rawHeader)

	jpgBytes := make([]byte, rawHeader.JpgLength)
	file.ReadAt(jpgBytes, int64(rawHeader.JpgOffset))

	if err := ioutil.WriteFile(outPath, jpgBytes, 0666); err != nil {
		log.Panicln(err)
	}

	jpgBytes2 := bytes.NewBuffer(jpgBytes)
	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(jpgBytes2)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v\n", x)

	focal, _ := x.Get(exif.FocalLength)
	numer, denom, _ := focal.Rat2(0) // retrieve first (only) rat. value
	fmt.Printf("%v\n", numer/denom)

	metaBytes := make([]byte, rawHeader.CfaHeaderLength)
	_, err = file.ReadAt(metaBytes, int64(rawHeader.CfaHeaderOffset))
	if err != nil {
		log.Panicln(err)
	}

	cfaHeader, err := ReadCFAHeader(bytes.NewBuffer(metaBytes))
	if err != nil {
		log.Panicln(err)
	}

	log.Println(cfaHeader)

	raw := RAWContainer{RAWHeader: rawHeader, CFAHeader: cfaHeader}

	_ = raw

}

func ReadRawHeader(rawFile *os.File) (RAWHeader, error) {
	var header RAWHeader

	if err := binary.Read(rawFile, binary.BigEndian, &header); err != nil {
		return header, err
	}

	return header, nil
}

type RAWContainer struct {
	RAWHeader

	CFAHeader
}

type RAWHeader struct {
	Magic [16]byte

	FormatVersion [4]byte

	CameraNumber [8]byte

	CameraName [32]byte

	Version [4]byte

	Unknown [20]byte

	JpgOffset uint32
	JpgLength uint32

	CfaHeaderOffset uint32
	CfaHeaderLength uint32

	CfaOffset uint32
	CfaLength uint32
}

type CFAHeader struct {
	NumRecords uint32

	Records []CFARecord
}

type CFARecord struct {
	TagID uint16
	Size  uint16

	Data interface{}
}

func (c CFARecord) String() string {
	return fmt.Sprintf(
		`
		TagID: %#x
		Size: %d
		Data: %v
		`,
		c.TagID,
		c.Size,
		c.Data,
	)
}

func ReadRAFData(r io.Reader) {
	data := make([]byte, 16)

	if err := binary.Read(r, binary.LittleEndian, &data); err != nil {
		log.Panicln(err)
	}

	log.Printf(`
	RawImageWidth %d
	RawImageWidth %d
	RawImageHeight %d
	RawImageWidth %d
	RawImageHeight %d
	RawImageHeight %d
	`,
		binary.LittleEndian.Uint32(data[:4]),
		binary.LittleEndian.Uint16(data[4:6]),
		binary.LittleEndian.Uint16(data[6:8]),
		binary.LittleEndian.Uint16(data[8:10]),
		binary.LittleEndian.Uint16(data[10:12]),
		binary.LittleEndian.Uint32(data[12:]),
	)
}

func ReadRAFSubdir(r io.Reader) {
	var numRecords uint32
	if err := binary.Read(r, binary.BigEndian, &numRecords); err != nil {
		log.Panicln(err)
	}

	for i := uint32(0); i < numRecords; i++ {
		var tagID uint16
		if err := binary.Read(r, binary.BigEndian, &tagID); err != nil {
			log.Panicln(err)
		}
		var recSize uint16
		if err := binary.Read(r, binary.BigEndian, recSize); err != nil {
			log.Panicln(err)
		}

		data := make([]byte, recSize)
		if err := binary.Read(r, binary.BigEndian, data); err != nil {
			log.Panicln(err)
		}

		log.Printf(`
		tag: %#x
		recSize: %d
		data: %v
		`,
			tagID,
			recSize,
			data)
	}
}

func SearchForValue(data []byte, test uint16) {
	for i := 0; i < len(data)-4; i++ {
		num := binary.BigEndian.Uint16(data[i : i+2])
		if num == test {
			log.Println("holy shit we found it", i)
		}
	}
}

func ReadCFAHeader(hReader io.Reader) (CFAHeader, error) {
	var header CFAHeader
	if err := binary.Read(hReader, binary.BigEndian, &header.NumRecords); err != nil {
		return header, err
	}

	for i := uint32(0); i < header.NumRecords; i++ {
		var rec CFARecord
		if err := binary.Read(hReader, binary.BigEndian, &rec.TagID); err != nil {
			return header, err
		}
		if err := binary.Read(hReader, binary.BigEndian, &rec.Size); err != nil {
			return header, err
		}

		data := make([]byte, rec.Size)

		if err := binary.Read(hReader, binary.BigEndian, data); err != nil {
			return header, err
		}

		switch rec.TagID {
		case RAFTagImgHeightWidth, RAFTagOutputHeightWidth, RAFTagImgTopLeft, RAFTagSensorDimension:
			h, w := binary.BigEndian.Uint16(data[:2]), binary.BigEndian.Uint16(data[2:])
			rec.Data = []uint16{h, w}
		case RAFTagRawInfo:
			rawProps := binary.BigEndian.Uint32(data)
			compressed := ((rawProps & 0xFF0000) >> 16) & 8
			rec.Data = compressed
		case 0xc000:
			SearchForValue(data, 4966)

			start, end := 0, 4
			e := binary.BigEndian

			n := e.Uint32(data[start:end])
			start, end = end, end+4
			if n > 10000 {
				n = e.Uint32(data[start:end])
				start, end = end, end+4
			}

			if n > 10000 {
				n = e.Uint32(data[start:end])
				start, end = end, end+4
			}

			log.Printf("Width! %d", n)
		default:
			log.Print("TagID", rec.TagID)
			continue
		}

		header.Records = append(header.Records, rec)
	}

	return header, nil
}

func (r RAWHeader) String() string {
	return fmt.Sprintf(
		`
		Magic: %s
		Format Version: %s
		Camera Number: %s
		Camera Name: %s
		Version: %s
		Unknown: %s
		JpgOffset: %d
		JpgLength: %d
		CfaHeaderOffset: %d
		CfaHeaderLength: %d
		CfaOffset: %d
		CfaLength: %d
		`,
		r.Magic,
		r.FormatVersion,
		r.CameraNumber,
		r.CameraName,
		r.Version,
		r.Unknown,
		r.JpgOffset,
		r.JpgLength,
		r.CfaHeaderOffset,
		r.CfaHeaderLength,
		r.CfaOffset,
		r.CfaLength,
	)
}

var inPath string
var outPath string

func init() {
	flag.StringVar(&outPath, "o", "", "Path of output file")

	flag.Parse()

	inPath = flag.Arg(0)

	if _, err := os.Stat(inPath); err != nil { // os.IsNotExist(err) {
		log.Panicln(err)
	}

	if outPath == "" {
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			log.Panicln(err)
		}
		outPath = filepath.Join(tempDir, "output.jpg")
	}
}

const (
	RAFTagSensorDimension   = 0x100
	RAFTagImgTopLeft        = 0x110 // Origin
	RAFTagImgHeightWidth    = 0x111 // Full Dimensions?
	RAFTagOutputHeightWidth = 0x121 // Cropped Dimension?
	RAFTagRawInfo           = 0x130
)
