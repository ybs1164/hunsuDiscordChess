package bot

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hunsuChess/game"
	"hunsuChess/handlers"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	game               *game.Game
	interactionHandler *handlers.InteractionHandler
}

func NewBot(game *game.Game) *Bot {
	return &Bot{
		game:               game,
		interactionHandler: &handlers.InteractionHandler{Game: game},
	}
}

func (bot *Bot) Start(token string) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Printf("err by session create : %v\n", err)
		return
	}

	session.AddHandler(bot.interactionHandler.Handle)

	err = session.Open()
	if err != nil {
		fmt.Printf("err by session open : %v\n", err)
		return
	}

	bot.addSlashCommands(session)

	defer session.Close()
	defer bot.CleanUp(session)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "봇 사용법을 안내합니다.",
		},
		{
			Name:        "game",
			Description: "체스 게임을 시작하거나 현재 게임 상태를 표시합니다.",
		},
		{
			Name:        "join",
			Description: "니트로 유저는 백, 무료 유저는 흑으로 팀이 자동 배정됩니다.",
		},
		{
			Name:        "move",
			Description: "가능한 체스 수를 확인하거나 직접 수를 입력합니다.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "move_uci",
					Description: "이동할 수 (UCI 형식)",
					Required:    false,
				},
			},
		},
	}
	commandIDs = make(map[string]string)
)

func (bot *Bot) addSlashCommands(s *discordgo.Session) {
	fmt.Println("Adding commands...")
	for _, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			fmt.Printf("Cannot create command %v: %v\n", v.Name, err)
			continue
		}
		commandIDs[cmd.ID] = cmd.Name
	}
	fmt.Println("Commands added!")
}

func (bot *Bot) CleanUp(s *discordgo.Session) {
	fmt.Println("Removing commands...")
	for id, name := range commandIDs {
		err := s.ApplicationCommandDelete(s.State.User.ID, "", id)
		if err != nil {
			fmt.Printf("Cannot delete command %s (%s): %v\n", name, id, err)
		}
	}
	fmt.Println("Commands removed!")
}
