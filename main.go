package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	hunsuChess "hunsuChess/chess"

	"github.com/bwmarrin/discordgo"
	"github.com/notnil/chess"
)

var token string

type Game struct {
	game         *chess.Game
	whitePlayers map[string]*Player
	blackPlayers map[string]*Player
	turn         bool // false : white, true : black

	nextTime time.Time

	recentMove string
}

func (game *Game) VoteMove(id string, chat string) error {
	var players map[string]*Player

	if !game.turn {
		players = game.whitePlayers
	} else {
		players = game.blackPlayers
	}

	if _, ok := players[id]; !ok {
		return errors.New("Not joined game")
	}

	for _, move := range game.game.ValidMoves() {
		if chat == move.String() {
			player := players[id]

			player.move = chat

			return nil
		}
	}

	return errors.New("Invalid move")
}

func (game *Game) GetVotes() []string {
	var players map[string]*Player

	moves := []string{}

	if !game.turn {
		players = game.whitePlayers
	} else {
		players = game.blackPlayers
	}

	for _, player := range players {
		if player.move == "" {
			continue
		}
		moves = append(moves, player.move)
	}

	return moves
}

func (game *Game) Next() {
	var players map[string]*Player

	movesCount := make(map[string]int)

	var move string
	var maxCount int = 0

	if !game.turn {
		players = game.whitePlayers
	} else {
		players = game.blackPlayers
	}

	for _, player := range players {
		movesCount[player.move] += 1
		player.move = ""
	}

	for m, c := range movesCount {
		if maxCount < c {
			move = m
			maxCount = movesCount[m]
		}
	}

	if maxCount == 0 {
		validMoves := game.game.ValidMoves()
		game.recentMove = validMoves[rand.Intn(len(validMoves))].String()
	} else {
		game.recentMove = move
	}

	m, _ := chess.UCINotation{}.Decode(game.game.Position(), game.recentMove)

	game.game.Move(m)
}

type Player struct {
	move string
}

var game *Game = &Game{
	game:         chess.NewGame(),
	whitePlayers: map[string]*Player{},
	blackPlayers: map[string]*Player{},
}

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	session, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Printf("err by session create : %v", err)
		return
	}

	go DayCycle()

	session.AddHandler(messageCreate)

	err = session.Open()
	defer session.Close()

	if err != nil {
		fmt.Printf("err by session open : %v", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func DayCycle() {
	for {
		now := time.Now().AddDate(0, 0, 1)
		year, month, day := now.Date()

		game.nextTime = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		<-time.After(time.Until(game.nextTime))

		game.Next()
	}
}

// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!game" {
		file := hunsuChess.ChessImage(game.game.FEN(), game.GetVotes())

		_, err := s.ChannelFileSendWithMessage(m.ChannelID, "하루가 지나갈 때 턴이 넘어갑니다.", "chess.png", file)
		if err != nil {
			panic(err)
		}
	} else if m.Content == "!white" {
		if _, ok := game.whitePlayers[m.Author.ID]; ok {
			s.ChannelMessageSend(m.ChannelID, "already joined")
			return
		}
		if _, ok := game.blackPlayers[m.Author.ID]; ok {
			s.ChannelMessageSend(m.ChannelID, "already joined")
			return
		}

		game.whitePlayers[m.Author.ID] = &Player{
			move: "",
		}

		s.ChannelMessageSend(m.ChannelID, string(m.Author.Username)+" join white team")
	} else if m.Content == "!black" {
		if _, ok := game.whitePlayers[m.Author.ID]; ok {
			s.ChannelMessageSend(m.ChannelID, "already joined")
			return
		}
		if _, ok := game.blackPlayers[m.Author.ID]; ok {
			s.ChannelMessageSend(m.ChannelID, "already joined")
			return
		}

		game.blackPlayers[m.Author.ID] = &Player{
			move: "",
		}

		s.ChannelMessageSend(m.ChannelID, string(m.Author.Username)+" join black team")
	} else if strings.HasPrefix(m.Content, "!move") {
		arguments := strings.Split(m.Content, " ")
		notations := arguments[1:]

		err := game.VoteMove(m.Author.ID, strings.Join(notations, ""))

		if err != nil {
			fmt.Printf("%v\n", err)
		}

		file := hunsuChess.ChessImage(game.game.FEN(), game.GetVotes())

		_, err = s.ChannelFileSend(m.ChannelID, "test.png", file)
		if err != nil {
			panic(err)
		}
	}
}
