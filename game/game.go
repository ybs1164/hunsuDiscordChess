package game

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/notnil/chess"
)

type Game struct {
	ChessGame    *chess.Game
	WhitePlayers map[string]*Player
	BlackPlayers map[string]*Player
	Turn         bool // false : white, true : black
	GameOver     bool

	NextTime time.Time

	RecentMove string
}

type Player struct {
	Move string
}

type moveVote struct {
	move  string
	count int
}

func NewGame() *Game {
	rand.Seed(time.Now().UnixNano())
	return &Game{
		ChessGame:    chess.NewGame(),
		WhitePlayers: make(map[string]*Player),
		BlackPlayers: make(map[string]*Player),
		GameOver:     false,
	}
}

func (game *Game) Reset() {
	game.ChessGame = chess.NewGame()
	game.Turn = false
	game.RecentMove = ""
	game.GameOver = false
	for _, p := range game.WhitePlayers {
		p.Move = ""
	}
	for _, p := range game.BlackPlayers {
		p.Move = ""
	}
}

func (game *Game) IsGameOver() bool {
	return game.GameOver
}

func (game *Game) VoteMove(id string, chat string) error {
	var players map[string]*Player

	if game.GameOver {
		return errors.New("game is over")
	}

	if !game.Turn {
		players = game.WhitePlayers
	} else {
		players = game.BlackPlayers
	}

	if _, ok := players[id]; !ok {
		return errors.New("not joined game")
	}

	// Try to match SAN
	for _, move := range game.ChessGame.ValidMoves() {
		san := chess.AlgebraicNotation{}.Encode(game.ChessGame.Position(), move)
		if chat == san {
			player := players[id]
			player.Move = move.String() // Store as UCI
			return nil
		}
	}

	// Fallback to match UCI
	for _, move := range game.ChessGame.ValidMoves() {
		if chat == move.String() {
			player := players[id]
			player.Move = chat
			return nil
		}
	}

	return errors.New("invalid move")
}

func (game *Game) GetVotes() []string {
	var players map[string]*Player
	moves := []string{}

	if !game.Turn {
		players = game.WhitePlayers
	} else {
		players = game.BlackPlayers
	}

	for _, player := range players {
		if player.Move == "" {
			continue
		}
		moves = append(moves, player.Move)
	}

	return moves
}

func (game *Game) Next() string {
	var players map[string]*Player
	movesCount := make(map[string]int)
	var maxCount int = 0

	if !game.Turn {
		players = game.WhitePlayers
	} else {
		players = game.BlackPlayers
	}

	for _, player := range players {
		movesCount[player.Move] += 1
		player.Move = ""
	}

	for m, c := range movesCount {
		if m == "" {
			continue
		}
		if maxCount < c {
			maxCount = c
		}
	}

	var tiedMoves []string
	for m, c := range movesCount {
		if m != "" && c == maxCount {
			tiedMoves = append(tiedMoves, m)
		}
	}

	if len(tiedMoves) > 0 {
		game.RecentMove = tiedMoves[rand.Intn(len(tiedMoves))]
	} else {
		validMoves := game.ChessGame.ValidMoves()
		if len(validMoves) > 0 {
			game.RecentMove = validMoves[rand.Intn(len(validMoves))].String()
		} else {
			game.RecentMove = ""
		}
	}

	if game.RecentMove != "" {
		m, _ := chess.UCINotation{}.Decode(game.ChessGame.Position(), game.RecentMove)
		game.ChessGame.Move(m)
	}

	if outcome := game.ChessGame.Outcome(); outcome != chess.NoOutcome {
		var result string
		switch outcome {
		case chess.WhiteWon:
			result = "백팀이 승리했습니다!"
		case chess.BlackWon:
			result = "흑팀이 승리했습니다!"
		case chess.Draw:
			result = "무승부입니다!"
		}
		method := game.ChessGame.Method()
		msg := fmt.Sprintf("게임 종료! %s (%s)", result, method.String())
		game.GameOver = true
		return msg
	}

	game.Turn = !game.Turn
	return ""
}

func (game *Game) GetVoteCounts() map[string]int {
	var players map[string]*Player

	if !game.Turn {
		players = game.WhitePlayers
	} else {
		players = game.BlackPlayers
	}

	counts := make(map[string]int)
	for _, player := range players {
		if player.Move != "" {
			counts[player.Move]++
		}
	}
	return counts
}

func (game *Game) GetTopNVotes(n int) string {
	counts := game.GetVoteCounts()
	var totalVotes int
	for _, count := range counts {
		totalVotes += count
	}

	if totalVotes == 0 {
		return "아직 투표가 없습니다. /move 명령어로 투표에 참여해보세요!"
	}

	sortedVotes := make([]moveVote, 0, len(counts))
	for move, count := range counts {
		sortedVotes = append(sortedVotes, moveVote{move, count})
	}

	sort.Slice(sortedVotes, func(i, j int) bool {
		return sortedVotes[i].count > sortedVotes[j].count
	})

	var topVotes []string
	for i := 0; i < n && i < len(sortedVotes); i++ {
		move, err := chess.UCINotation{}.Decode(game.ChessGame.Position(), sortedVotes[i].move)
		var moveStr string
		if err != nil {
			moveStr = sortedVotes[i].move
		} else {
			moveStr = chess.AlgebraicNotation{}.Encode(game.ChessGame.Position(), move)
		}

		percentage := float64(sortedVotes[i].count) / float64(totalVotes) * 100
		topVotes = append(topVotes, fmt.Sprintf("%s: %.2f%% (%d표)", moveStr, percentage, sortedVotes[i].count))
	}

	if len(topVotes) == 0 {
		return "아직 투표가 없습니다."
	}

	return "현재 투표 현황:\n" + strings.Join(topVotes, "\n")
}

func (game *Game) GetPlayerTeam(id string) (string, bool) {
	if _, ok := game.WhitePlayers[id]; ok {
		return "white", true
	}
	if _, ok := game.BlackPlayers[id]; ok {
		return "black", true
	}
	return "", false
}

func (game *Game) AddWhitePlayer(id string) {
	if _, ok := game.BlackPlayers[id]; ok {
		delete(game.BlackPlayers, id)
	}
	game.WhitePlayers[id] = &Player{}
}

func (game *Game) AddBlackPlayer(id string) {
	if _, ok := game.WhitePlayers[id]; ok {
		delete(game.WhitePlayers, id)
	}
	game.BlackPlayers[id] = &Player{}
}
