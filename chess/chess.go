package hunsuChess

import (
	"errors"
	"fmt"
	"strings"
)

var piece_list map[string]uint8 = map[string]uint8{
	"":  1,
	"N": 2,
	"B": 3,
	"R": 4,
	"Q": 5,
	"K": 6,
}

type ChessGame struct {
	board    [8][8]uint8 // 1: pawn, 2: knight, 3: bishop, 4: rook, 5: queen, 6: king => black + 8
	notation []string
	turn     bool // false: white, true: black
}

func NewChessGame() *ChessGame {
	return &ChessGame{
		board: [8][8]uint8{
			{12, 10, 11, 13, 14, 11, 10, 12},
			{9, 9, 9, 9, 9, 9, 9, 9},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{4, 2, 3, 5, 6, 3, 2, 4},
		},
		notation: make([]string, 0),
		turn:     false,
	}
}

// TODO
func (game *ChessGame) Move(notation string) (*ChessGame, error) {
	game.notation = append(game.notation, notation)

	// Castling moves
	if notation == "O-O" {
		return game, nil
	} else if notation == "O-O-O" {
		return game, nil
	}

	// TODO : promotion

	// Remove Check and Checkmate notation
	notation = strings.TrimSuffix(notation, "+")
	notation = strings.TrimSuffix(notation, "#")

	position_notation := notation[len(notation)-2:]
	notation = notation[:len(notation)-2]

	// Remove Catch notation
	notation = strings.TrimSuffix(notation, "x")

	// TODO : Get position of moving piece

	piece := piece_list[notation]
	// if black, change piece team
	if game.turn {
		piece += 8
	}

	x := int(position_notation[0] - 'a')
	y := int(8 - (position_notation[1] - '0'))

	if piece == 1 { // white pawn
		fmt.Printf("%v %v\n", x, y)
		if game.board[y+1][x] == piece {
			game.board[y+1][x] = 0
			game.board[y][x] = piece
		} else if game.board[y+2][x] == piece {
			game.board[y+2][x] = 0
			game.board[y][x] = piece
		} else {
			return game, errors.New("pawn do not can move that position")
		}
	} else if piece == 2 { // white knight
		xx := []int{1, 2, 2, 1, -1, -2, -2, -1}
		yy := []int{2, 1, -1, -2, -2, -1, 1, 2}

		for i := 0; i < 8; i++ {
			if x+xx[i] < 0 || x+xx[i] > 7 {
				continue
			}
			if y+yy[i] < 0 || y+yy[i] > 7 {
				continue
			}
			// TODO
		}
	} else if piece == 3 {

	} else if piece == 4 {

	} else if piece == 5 {

	} else if piece == 6 {

	} else if piece == 9 { // black piece

	} else if piece == 10 {

	} else if piece == 11 {

	} else if piece == 12 {

	} else if piece == 13 {

	} else if piece == 14 {

	}

	game.turn = !game.turn

	return game, nil
}
