package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

var (
	outPath  string
	inPath   string
	demosaic string
	verbose  bool
)

func init() {
	flag.StringVar(&outPath, "out", "", "Path of output file")
	flag.StringVar(&inPath, "in", "", "Path of input file")
	flag.StringVar(&demosaic, "demosaic", "", "Demosaic method, valid options are: none, nearest_neighbor")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")

	flag.Parse()

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

	if verbose {
		fmt.Printf("inPath: %+v\n", inPath)
		fmt.Printf("outPath: %+v\n", outPath)
		fmt.Printf("verbose: %+v\n", verbose)
	}
}

// 8922 expected raw resolution of 6000x4000, adobe rgb, depth 8bit, 300ppi

func main() {
	log.Println("Giraffe loves pictures 🦒")

	flag.Parse()

	if inPath == "" {
		log.Fatal("requires an -in=image.raf")
	}

	file, err := os.Open(inPath)
	if err != nil {
		log.Panicln(err)
	}

	switch flag.Arg(0) {
	case "exif":
		displayExifData(file)
	case "render":
		renderImage(file)
	default:
		log.Fatal("invalid command")
	}
}

func displayExifData(file *os.File) {
	if verbose {
		log.Println("inPath:", inPath)
	}

	rawHeader, err := ReadRawHeader(file)
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		fmt.Printf("rawHeader: %+v\n", rawHeader)
	}

	jpgBytes := make([]byte, rawHeader.JpgLength)
	file.ReadAt(jpgBytes, int64(rawHeader.JpgOffset))

	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(bytes.NewBuffer(jpgBytes))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v\n", x)
}

func renderImage(file *os.File) {
	if outPath == "" {
		log.Fatal("requires -out=image.jpg")
	}

	if verbose {
		log.Println("inPath:", inPath)
		log.Println("outPath:", outPath)
	}

	rawHeader, err := ReadRawHeader(file)
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		fmt.Printf("rawHeader: %+v\n", rawHeader)
	}

	metaBytes := make([]byte, rawHeader.CfaHeaderLength)
	_, err = file.ReadAt(metaBytes, int64(rawHeader.CfaHeaderOffset))
	if err != nil {
		log.Panicln(err)
	}

	cfaHeader, err := ReadCFAHeader(bytes.NewBuffer(metaBytes))
	if err != nil {
		log.Panicln(err)
	}

	if verbose {
		log.Println(cfaHeader)
	}

	cfaBytes := make([]byte, rawHeader.CfaLength)
	_, err = file.ReadAt(cfaBytes, int64(rawHeader.CfaOffset))
	if err != nil {
		log.Panicln(err)
	}

	DoStuffWithCFABytes(cfaBytes)

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
			d := bytes.NewBuffer(data)
			w, h, err := getRAFWidthHeight(d)
			if err != nil {
				log.Panicln(err)
			}
			rec.Data = struct{ Width, Height uint32 }{w, h}
		default:
			continue
		}

		header.Records = append(header.Records, rec)
	}

	return header, nil
}

type CFAData struct {
	width, height int
	data          []uint16

	grayscale []color.Gray
	rgb       []color.RGBA
}

type Color int

const (
	// Green Represents a Green pixel
	Green Color = 0
	// Blue Represents a Blue pixel
	Blue Color = 1
	// Red Represents a Red pixel
	Red Color = 2
)

var XTransPattern = [][]Color{
	[]Color{Green, Blue, Green, Green, Red, Green},
	[]Color{Red, Green, Red, Blue, Green, Blue},
	[]Color{Green, Blue, Green, Green, Red, Green},
	[]Color{Green, Red, Green, Green, Blue, Green},
	[]Color{Blue, Green, Blue, Red, Green, Red},
	[]Color{Green, Red, Green, Green, Blue, Green},
}

// given a row and a column, return the color of this pixel
func filterColor(row, col int) Color {
	return XTransPattern[(row+6)%6][(col+6)%6]
}

// The FujiFilm X-H1 has a color depth of 14 bits
const BitDepth = 1 << 14

func (c CFAData) nearestNeighbor(x int, y int, targetColor Color) uint8 {
	for dX := -1; dX <= 1; dX++ {
		if x+dX >= c.width ||
			x+dX < 0 {
			continue
		}
		for dY := -1; dY <= 1; dY++ {
			if y+dY >= c.height ||
				y+dY < 0 {
				continue
			}

			if filterColor(x+dX, y+dY) == targetColor {
				var intensity uint8
				// Do something

				intensity = uint8((float64(c.data[(y+dY)*c.width+((x+dX)%c.width)]) / float64(BitDepth)) * 255)

				return intensity
			}
		}
	}

	log.Println("Couldn't find values for (", x, ",", y, ")")
	return 0
}

func (c CFAData) intensityAt(x, y int) uint8 {
	return uint8((float64(c.data[y*c.width+(x%c.width)]) / float64(BitDepth)) * 255)
}

func (c CFAData) bilinearInterp(x, y int, targetColor Color) uint8 {
	var intensityAcc uint64
	var intensityCount uint64

	for dX := -1; dX <= 1; dX++ {
		if x+dX >= c.width ||
			x+dX < 0 {
			continue
		}
		for dY := -1; dY <= 1; dY++ {
			if y+dY >= c.height ||
				y+dY < 0 {
				continue
			}

			if filterColor(x+dX, y+dY) == targetColor {
				intensityAcc += uint64(c.intensityAt(x+dX, y+dY))
				intensityCount++
			}
		}
	}

	return uint8(intensityAcc / intensityCount)
}

func (c CFAData) grayAt(x, y int) *color.Gray {
	return &c.grayscale[y*c.width+x]
}

func (c CFAData) rgbAt(x, y int) *color.RGBA {
	return &c.rgb[y*c.width+x]
}

func (c CFAData) demosaicUsingColorHue() {
	// Setup our initial intermediate color array values
	for i := range c.rgb {
		x := i % c.width
		y := i / c.width

		switch filterColor(x, y) {
		case Red:
			c.rgb[i].R = c.intensityAt(x, y)
		case Green:
			c.rgb[i].G = c.intensityAt(x, y)
		case Blue:
			c.rgb[i].B = c.intensityAt(x, y)
		}
		c.rgb[i].A = 255
	}

	// Interp green into non-green pixels
	for i := range c.rgb {
		x := i % c.width
		y := i / c.width

		if filterColor(x, y) != Green {
			c.rgb[i].G = c.bilinearInterp(x, y, Green)
		}
	}

	// interp non greens onto neighboring pixels
	for i := range c.rgb {
		x := i % c.width
		y := i / c.width

		switch filterColor(x, y) {
		case Blue:
			c.rgbAt(x, y).R = c.colorRatioInterp(x, y, Red)
		case Red:
			c.rgbAt(x, y).B = c.colorRatioInterp(x, y, Blue)
		case Green:
			c.rgbAt(x, y).R = c.colorRatioInterp(x, y, Red)
			c.rgbAt(x, y).B = c.colorRatioInterp(x, y, Blue)
		}
	}
}

func (c CFAData) grayscaleImage() {
	for i := range c.grayscale {
		x := i % c.width
		y := i / c.width

		c.grayAt(x, y).Y = c.intensityAt(x, y)
	}
}

func (c CFAData) Demosaic(method string) {
	switch method {
	case "color_hue":
		c.demosaicUsingColorHue()
	default:
		c.grayscaleImage()
	}
}

func (c CFAData) colorRatioInterp(x, y int, target Color) uint8 {
	var intensityAcc uint64
	var intensityCount uint64

	for dX := -1; dX <= 1; dX++ {
		if x+dX >= c.width ||
			x+dX < 0 {
			continue
		}
		for dY := -1; dY <= 1; dY++ {
			if y+dY >= c.height ||
				y+dY < 0 ||
				dY == 0 && dX == 0 {
				continue
			}

			if filterColor(x+dX, y+dY) == target {
				pix := c.rgbAt(x+dX, y+dY)
				switch target {
				case Red:
					if pix.G != 0 {
						intensityAcc += uint64(pix.R / pix.G)
					}
					intensityCount++
				case Blue:
					if pix.G != 0 {
						intensityAcc += uint64(pix.B / pix.G)
					}
					intensityCount++
				}
			}
		}
	}

	return uint8(float64(intensityAcc) / float64(intensityCount) * float64(c.rgbAt(x, y).G))

}

func (c CFAData) At(x int, y int) color.Color {
	if demosaic == "" {
		return c.grayscale[y*c.width+x]
	}
	return c.rgb[y*c.width+x]
}

/*
func (c CFAData) At(x int, y int) color.Color {
	pixel := color.RGBA{A: 255}

	// intensity := (float64(c.data[y*6160+(x%6160)]) / 65535) * 255
	intensity := c.intensityAt(x, y)

	switch filterColor(x, y) {
	case Red:
		// pixel.R = intensity
		// pixel.G = c.nearestNeighbor(x, y, Green)
		// pixel.B = c.nearestNeighbor(x, y, Blue)

		pixel.G = c.bilinearInterp(x, y, Green)
	case Green:
		// pixel.R = c.nearestNeighbor(x, y, Red)
		// pixel.G = intensity
		// pixel.B = c.nearestNeighbor(x, y, Blue)

		pixel.G = intensity
	case Blue:
		// pixel.R = c.nearestNeighbor(x, y, Red)
		// pixel.G = c.nearestNeighbor(x, y, Green)
		// pixel.B = intensity

		pixel.G = c.bilinearInterp(x, y, Green)
	}

	return pixel
}
*/

func (c CFAData) ColorModel() color.Model {
	if demosaic == "" {
		return color.GrayModel
	}
	return color.RGBAModel
}

func (c CFAData) Bounds() image.Rectangle {
	return image.Rect(0, 0, 6160, 4032)
}

func DoStuffWithCFABytes(data []byte) {
	_ = data[:2048] // unused header?
	rawData := data[2048:]
	log.Printf("CFA Bytes len %d % #x", len(rawData), rawData[:32])

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Panicln(err)
	}

	f, err := os.Create(filepath.Join(dir, "rawimg.jpg"))
	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()

	cfaData := CFAData{
		data:      make([]uint16, 6160*4032),
		width:     6160,
		height:    4032,
		grayscale: make([]color.Gray, 6160*4032),
		rgb:       make([]color.RGBA, 6160*4032)}

	binary.Read(bytes.NewBuffer(rawData), binary.LittleEndian, cfaData.data)

	cfaData.Demosaic(demosaic)

	jpeg.Encode(f, cfaData, &jpeg.Options{Quality: 100})
	log.Printf("Raw image at: %s", f.Name())
}

func getRAFWidthHeight(r io.Reader) (uint32, uint32, error) {
	var width uint32

	for {
		var err error
		width, err = get4(r)
		if err != nil {
			return 0, 0, err
		}

		if width < 10000 {
			break
		}
	}

	height, err := get4(r)
	if err != nil {
		return 0, 0, err
	}

	log.Printf("Width: %d, Height: %d", width, height)

	return width, height, nil
}

func get4(r io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		log.Panicln(err)
	}

	num := binary.LittleEndian.Uint32(buf)

	return num, nil
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

const (
	RAFTagSensorDimension   = 0x100
	RAFTagImgTopLeft        = 0x110 // Origin
	RAFTagImgHeightWidth    = 0x111 // Full Dimensions?
	RAFTagOutputHeightWidth = 0x121 // Cropped Dimension?
	RAFTagRawInfo           = 0x130
)
