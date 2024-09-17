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

var games map[string]*chess.Game = make(map[string]*chess.Game)

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

/* Notation
 *
**/
func toChessNotation(game *chess.Game, chat string) []string {
	var notation_list []string
	// moves := game.ValidMoves()
	if err := game.MoveStr(chat); err != nil {
		fmt.Println("Error")
		return []string{}
	}

	return notation_list
}

// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!singleGame" {
		if games[m.ChannelID] != nil {
			s.ChannelMessage(m.ChannelID, "game is already exists")
			return
		}
		game := chess.NewGame()

		games[m.ChannelID] = game

		file := hunsuChess.ChessImage(game.FEN())

		_, err := s.ChannelFileSend(m.ChannelID, "test.png", file)
		if err != nil {
			panic(err)
		}
	} else if strings.HasPrefix(m.Content, "!") {
		if games[m.ChannelID] == nil {
			return
		}
		game := games[m.ChannelID]
		toChessNotation(game, m.Content[1:])

		file := hunsuChess.ChessImage(game.FEN())

		_, err := s.ChannelFileSend(m.ChannelID, "test.png", file)
		if err != nil {
			panic(err)
		}
	}
}
