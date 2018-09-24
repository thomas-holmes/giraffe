package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.Println("Giraffe loves pictures")

	log.Println("inPath:", inPath)
	log.Println("outPath:", outPath)

}

func ReadRaw(rawFile *os.File) (RAWContainer, error) {

	return RAWContainer{}, nil

}

type RAWContainer struct{}

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
