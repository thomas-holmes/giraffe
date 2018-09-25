package exif

// type Exif struct {
// 	Tiff *tiff.Tiff
// 	main map[FieldName]*tiff.Tag
// 	Raw  []byte
// }

// func Decode(r io.Reader) (*Exif, error) {
// 	header := make([]byte, 4)
// 	n, err := io.ReadFull(r, header)
// 	if err != nil {
// 		return nil, fmt.Errorf("exif: error reading 4 byte header, got %d, %v", n, err)
// 	}

// 	var (
// 		isTiff     bool
// 		isRawExif  bool
// 		assumeJpeg bool
// 	)

// 	switch string(header) {
// 	case "II*\x00":
// 		// TIFF - Little Endian (Intel)
// 		isTiff = true
// 	case "MM\x00*":
// 		// TIFF - Big Endian (Motorola)
// 		isTiff = true
// 	case "Exif":
// 		isRawExif = true
// 	default:
// 		// Not TIFF, assume JPEG
// 		assumeJpeg = true
// 	}

// 	r = io.MultiReader(bytes.NewReader(header), r)
// 	var (
// 		er  *bytes.Reader
// 		tif *tiff.Tiff
// 		sec *appSec
// 	)
// }
