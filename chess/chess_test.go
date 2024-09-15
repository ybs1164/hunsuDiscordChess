package hunsuChess

import (
	"fmt"
	"testing"
)

func TestMove(t *testing.T) {
	game := NewChessGame()

	fmt.Println(game.Move("e4"))
}
