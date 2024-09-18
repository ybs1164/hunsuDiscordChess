package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	hunsuChess "hunsuChess/chess"

	"github.com/bwmarrin/discordgo"
	"github.com/notnil/chess"
)

var token string

type Game struct {
	game         *chess.Game
	whitePlayers []Player
	blackPlayers []Player
}

func (game *Game) VoteMove(id string, chat []string) {

}

type Player struct {
	id   string
	move string
}

var games map[string]*Game = make(map[string]*Game)

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

func MoveChess(game *chess.Game, chat []string) {
	moves := game.ValidMoves()
	if len(chat) == 1 {
		// TODO : read PGN
		for _, move := range moves {
			if move.String() != chat[0] {
				continue
			}
			game.MoveStr(chat[0])
		}
	} else if len(chat) == 2 {
		// TODO : vote move
		for _, move := range moves {
			if move.S1().String() == chat[0] && move.S2().String() == chat[1] {
				game.Move(move)
			}
		}
	}
}

// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!gameSet" {
		if games[m.ChannelID] != nil {
			s.ChannelMessage(m.ChannelID, "game is already exists")
			return
		}
		game := chess.NewGame()

		games[m.ChannelID] = &Game{
			game: game,
		}

		file := hunsuChess.ChessImage(game.FEN(), []string{"e2e4", "d2d4", "b1c3"})

		_, err := s.ChannelFileSend(m.ChannelID, "test.png", file)
		if err != nil {
			panic(err)
		}
	} else if strings.HasPrefix(m.Content, "!move") {
		if games[m.ChannelID] == nil {
			return
		}
		game := games[m.ChannelID]
		arguments := strings.Split(m.Content, " ")
		notations := arguments[1:]

		MoveChess(game.game, notations)

		file := hunsuChess.ChessImage(game.game.FEN(), []string{})

		_, err := s.ChannelFileSend(m.ChannelID, "test.png", file)
		if err != nil {
			panic(err)
		}
	}
}
