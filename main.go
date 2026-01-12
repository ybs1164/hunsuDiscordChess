package main

import (
	"flag"
	"time"

	"hunsuChess/bot"
	"hunsuChess/game"
)

var (
	token string
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	gameInstance := game.NewGame()

	go DayCycle(gameInstance)

	botInstance := bot.NewBot(gameInstance)
	botInstance.Start(token)
}

func DayCycle(gameInstance *game.Game) {
	for {
		now := time.Now().Add(24 * time.Hour)
		year, month, day := now.Date()
		gameInstance.NextTime = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		<-time.After(time.Until(gameInstance.NextTime))
		gameInstance.Next()
	}
}
