package hunsuChess

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/fogleman/gg"
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

func ChessImage(fen string, arrows []string) io.Reader {
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

	// File and Rank
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

	png.Encode(file, AddArrowsInBoard(img, arrows))

	return file
}

func GetPosition(position string) (float64, float64) {
	return float64(position[0] - 'a'), float64(7 - (position[1] - '1'))
}

func AddArrowsInBoard(img image.Image, arrows []string) image.Image {
	board := gg.NewContextForImage(img)

	for _, arrow := range arrows {
		pre, post := arrow[:2], arrow[2:]
		preFile, preRank := GetPosition(pre)
		postFile, postRank := GetPosition(post)

		preLineX := preFile*45 + 22.5
		preLineY := preRank*45 + 22.5

		postLineX := postFile*45 + 22.5
		postLineY := postRank*45 + 22.5

		angle := math.Atan2(postLineY-preLineY, postLineX-preLineX)

		triAngleX := []float64{postLineX}
		triAngleY := []float64{postLineY}

		preLineX += math.Cos(angle) * 18.
		preLineY += math.Sin(angle) * 18.

		postLineX -= math.Cos(angle) * 18.
		postLineY -= math.Sin(angle) * 18.

		triAngleX = append(triAngleX, postLineX+math.Cos(angle+math.Pi/2)*14.)
		triAngleY = append(triAngleY, postLineY+math.Sin(angle+math.Pi/2)*14.)

		triAngleX = append(triAngleX, postLineX+math.Cos(angle+math.Pi/2)*6.)
		triAngleY = append(triAngleY, postLineY+math.Sin(angle+math.Pi/2)*6.)

		triAngleX = append(triAngleX, preLineX+math.Cos(angle+math.Pi/2)*6.)
		triAngleY = append(triAngleY, preLineY+math.Sin(angle+math.Pi/2)*6.)

		triAngleX = append(triAngleX, preLineX+math.Cos(angle-math.Pi/2)*6.)
		triAngleY = append(triAngleY, preLineY+math.Sin(angle-math.Pi/2)*6.)

		triAngleX = append(triAngleX, postLineX+math.Cos(angle-math.Pi/2)*6.)
		triAngleY = append(triAngleY, postLineY+math.Sin(angle-math.Pi/2)*6.)

		triAngleX = append(triAngleX, postLineX+math.Cos(angle-math.Pi/2)*14.)
		triAngleY = append(triAngleY, postLineY+math.Sin(angle-math.Pi/2)*14.)

		triAngleX = append(triAngleX, triAngleX[0])
		triAngleY = append(triAngleY, triAngleY[0])

		board.SetRGBA(1, 1, 0, 0.8)
		board.MoveTo(triAngleX[0], triAngleY[0])
		for i := 0; i < len(triAngleX)-1; i++ {
			board.LineTo(triAngleX[i+1], triAngleY[i+1])
		}
		board.Fill()
	}

	return board.Image()
}
