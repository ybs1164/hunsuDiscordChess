package hunsuChess

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var light = color.RGBA{235, 209, 166, 255}
var dark = color.RGBA{165, 117, 81, 255}

var board image.Image
var pieces = map[rune]image.Image{
	'b': nil,
	'k': nil,
	'n': nil,
	'p': nil,
	'q': nil,
	'r': nil,
	'B': nil,
	'K': nil,
	'N': nil,
	'P': nil,
	'Q': nil,
	'R': nil,
}

var piece_to_file_name = map[rune]string{
	'b': "bB",
	'k': "bK",
	'n': "bN",
	'p': "bP",
	'q': "bQ",
	'r': "bR",
	'B': "wB",
	'K': "wK",
	'N': "wN",
	'P': "wP",
	'Q': "wQ",
	'R': "wR",
}

func init() {
	file_board, err := os.Open("images/board.png")
	if err != nil {
		panic(err)
	}
	defer file_board.Close()

	data, _, err := image.Decode(file_board)
	if err != nil {
		panic(err)
	}
	board = data

	for piece_type := range pieces {
		file, err := os.Open("images/" + piece_to_file_name[piece_type] + ".png")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		data, _, err := image.Decode(file)
		if err != nil {
			panic(err)
		}
		pieces[piece_type] = data
	}
}

func addLabel(img *image.NRGBA, x, y int, label string, c color.Color) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func ChessImage(fen string) io.Reader {
	width, height := 360, 360

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), board, image.Point{}, draw.Over)

	// Set Pieces
	datas := strings.Split(fen, " ")
	x, y := 0, 0

	for _, char := range datas[0] {
		if char == '/' {
			y++
			x = 0
		} else if char >= '1' && char <= '8' {
			x += int(char - '0')
		} else {
			draw.Draw(img, image.Rect(x*45, y*45, x*45+45, y*45+45), pieces[char], image.Point{}, draw.Over)
			x++
		}
	}

	for i := 1; i <= 8; i++ {
		var c color.RGBA
		if i%2 == 0 {
			c = dark
		} else {
			c = light
		}
		addLabel(img, 2, (9-i)*45-33, strconv.Itoa(i), c)
		addLabel(img, i*45-8, 358, string(rune(i+'a'-1)), c)
	}

	file := bytes.NewBuffer([]byte{})

	png.Encode(file, img)

	return file
}
