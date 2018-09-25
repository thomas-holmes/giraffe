package raw

import "fmt"

type Color int

const (
	// Green Represents a Green pixel
	Green Color = 0
	// Blue Represents a Blue pixel
	Blue Color = 1
	// Red Represents a Red pixel
	Red Color = 2
)

// TileSize is ???
const TileSize = 512

// XTransPattern represents the pattern of the color filter array
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

func interpolate(passes, width, height int) {
	var (
		c    int
		d    int
		f    int
		g    int
		h    int
		v    int
		ng   int
		row  int
		col  int
		top  int
		left int
		mrow int
		mcol int

		val  int
		ndir int
		pass int

		hm    [8]int
		avg   [4]int
		color [3][8]int

		min   int
		max   int
		sgrow int
		sgcol int

		allhex [3][3][2][8]int
	)

	orth := [12]int{
		1, 0, 0, 1,
		-1, 0, 0, -1,
		1, 0, 0, 1,
	}

	pattern := [2][16]int{
		{0, 1, 0, -1, 2, 0, -1, 0, 1, 1, 1, -1, 0, 0, 0, 0},
		{0, 1, 0, -2, 1, 0, -2, 0, 1, 1, -2, -2, 1, -1, -1, 1},
	}

	direction := [4]int{1, TileSize, TileSize + 1, TileSize - 1}

	fmt.Printf("%d-pass X-Trans interpolation\n", passes)

	// Map a green hexagon around each non-green pixel and vice-versa

	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			for ng, d := 0, 0; d < 10; d += 2 {

				// if the filter in this location has a green pixel, use pattern 0
				// else use pattern 1
				if filterColor(row, col) == Green {
					g = 0
				} else {
					g = 1
				}

				// Calculate "new green?" based on the orthogonal????
				if filterColor(row+orth[d], col+orth[d+2]) == Green {
					ng = 0
				} else {
					ng++
				}

				// ????
				if ng == 4 {
					sgrow, sgcol = row, col
				}

				/// ?????
				if ng == g+1 {
					v = orth[d]*pattern[g][c*2] + orth[d+1]*pattern[g][c*2+1]
					h = orth[d+2]*pattern[g][c*2] + orth[d+3]*pattern[g][c*2+1]

					allhex[row][col][0][c^(g*2&d)] = h + v*width
					allhex[row][col][1][c^(g*2&d)] = h + v*TileSize
				}
			}
		}
	}

	// Set green1 and green3 to the minimum and maximum allowed values
	// what those values are, no idea
	for row := 2; row < height-2; row++ {
		for min, max, col := 65535, 0, 2; col < width-2; col++ {

			// if the pixel is already green and min is the negation of max?
			if filterColor(row, col) == Green && min == 65535 {
				continue
			}

			pix := image + row*width + col
			hex := allhex[row%3][col%3][0]

			if max > 0 {
				for c := 0; c < 6; c++ {
					val = pix[hex[c]][1]
					if min > val {
						min = val
					}

					if max < val {
						max = val
					}
				}
			}
		}
	}
}
