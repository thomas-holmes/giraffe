package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.Println("Giraffe loves pictures ��")

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
