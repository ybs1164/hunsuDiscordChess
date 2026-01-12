package handlers

import (
	"hunsuChess/game"

	"github.com/bwmarrin/discordgo"
)

func MessageEditToWebhookEdit(msg *discordgo.MessageEdit) *discordgo.WebhookEdit {
	if msg == nil {
		return nil
	}
	return &discordgo.WebhookEdit{
		Content:     msg.Content,
		Embeds:      msg.Embeds,
		Components:  msg.Components,
		Attachments: msg.Attachments,
		Files:       msg.Files,
	}
}

func StrPtr(s string) *string {
	return &s
}

// CheckPlayerAndTurn checks if a player is on a team and if it is their turn.
// It returns an error message if a check fails, or an empty string if all checks pass.
func CheckPlayerAndTurn(g *game.Game, userID string) string {
	if g.IsGameOver() {
		return "게임이 종료되었습니다. `/game` 명령어로 새 게임을 시작하세요."
	}
	team, ok := g.GetPlayerTeam(userID)
	if !ok {
		return "팀에 소속되어야 합니다. `/join`을 사용해 팀에 참여하세요."
	}
	if (team == "white" && g.Turn) || (team == "black" && !g.Turn) {
		return "당신의 턴이 아닙니다."
	}
	return ""
}

// CheckPlayer checks if a player is on a team.
func CheckPlayer(g *game.Game, userID string) string {
	_, ok := g.GetPlayerTeam(userID)
	if !ok {
		return "게임에 참여해야 합니다. `/join`을 사용해 팀에 참여하세요."
	}
	return ""
}
