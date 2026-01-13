package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"hunsuChess/chess"
	"hunsuChess/game"

	"github.com/bwmarrin/discordgo"
	notnilchess "github.com/notnil/chess"
)

type InteractionHandler struct {
	Game *game.Game
}

func (h *InteractionHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		h.handleApplicationCommand(s, i)
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, ";")
		var User *discordgo.User

		if i.Member == nil {
			User = i.User
		} else {
			User = i.Member.User
		}

		if len(parts) == 2 {
			authorID := parts[1]
			if authorID != User.ID {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "이 버튼을 누를 수 없습니다.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			customID = parts[0]
		}

		if errMsg := CheckPlayerAndTurn(h.Game, User.ID); errMsg != "" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: errMsg,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		switch {
		case strings.HasPrefix(customID, chess.PrefixMovePage):
			h.handleMovePage(s, i, customID)
		case strings.HasPrefix(customID, chess.PrefixMoveSelect):
			h.handleMoveSelect(s, i, customID)
		case strings.HasPrefix(customID, chess.PrefixMoveVote):
			h.handleMoveVote(s, i, customID)
		case customID == chess.PrefixMoveCancel:
			h.handleMoveCancel(s, i)
		}
	}
}

func (h *InteractionHandler) handleApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdData := i.ApplicationCommandData()

	switch cmdData.Name {
	case "help":
		h.handleHelpCommand(s, i)
	case "game":
		h.handleGameCommand(s, i)
	case "join":
		h.handleJoinCommand(s, i)
	case "move":
		h.handleMoveCommand(s, i)
	// case "skip":
	// 	h.handleSkipCommand(s, i)
	// case "vote":
	// 	h.handleVoteCommand(s, i)
	default:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "알 수 없는 명령입니다.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func (h *InteractionHandler) handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	helpMessage := "니트로 유저들은 백, 외의 유저들은 흑으로 나뉘어 투표를 통해 다수결로 체스를 두는 봇입니다.\n" +
		"하루가 지나갈 때마다 턴이 넘어가며, 각 팀의 플레이어들은 자신의 턴에 투표를 할 수 있습니다.\n\n" +
		"**/join**: 게임에 참여합니다.\n" +
		"**/game**: 현재 게임 상태를 확인합니다.\n" +
		"**/move**: 두고 싶은 수에 투표합니다.\n\n" +
		"봇에 관련된 피드백 또는 버그 제보는 **@number_er**으로 연락해주시면 감사하겠습니다."

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *InteractionHandler) handleGameCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var message string

	if h.Game.IsGameOver() {
		outcome := h.Game.ChessGame.Outcome()
		method := h.Game.ChessGame.Method()
		var result string
		switch outcome {
		case notnilchess.WhiteWon:
			result = "백팀이 승리했습니다!"
		case notnilchess.BlackWon:
			result = "흑팀이 승리했습니다!"
		case notnilchess.Draw:
			result = "무승부입니다!"
		}

		message = fmt.Sprintf("게임 종료! %s (%s)\n`/game` 명령어로 새 게임을 시작할 수 있습니다.", result, method.String())
	} else {
		now := time.Now()
		duration := h.Game.NextTime.Sub(now)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60

		var turn string
		if h.Game.ChessGame.Position().Turn() == notnilchess.White {
			turn = "백"
		} else {
			turn = "흑"
		}

		message = fmt.Sprintf("%s팀 차례가 넘어갈 때까지 %d시간 %d분 %d초 남았습니다.\n\n%s", turn, hours, minutes, seconds, h.Game.GetTopNVotes(3))
	}

	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	if errMsg := CheckPlayer(h.Game, User.ID); errMsg != "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	team, _ := h.Game.GetPlayerTeam(User.ID)
	fen := h.Game.ChessGame.FEN()
	if team == "black" {
		parts := strings.Split(fen, " ")
		runes := []rune(parts[0])
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		parts[0] = string(runes)
		fen = strings.Join(parts, " ")
	}

	file := chess.ChessImage(fen, h.Game.GetVotes(), team)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Files: []*discordgo.File{
				{
					Name:        "chess.png",
					ContentType: "image/png",
					Reader:      file,
				},
			},
		},
	})
}

func (h *InteractionHandler) handleJoinCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	// As direct Nitro detection (PremiumType) is unreliable, we check for features that require Nitro,
	// such as having a user-specific avatar, banner, or accent color.
	member, err := s.GuildMember(i.GuildID, User.ID)
	isPremium := User.PremiumType > 0 || User.Banner != "" || User.AccentColor > 0

	if err == nil {
		isPremium = isPremium || (i.Member != nil && i.Member.PremiumSince != nil) || (member != nil && member.Avatar != "")
	}

	if isPremium {
		h.Game.AddWhitePlayer(User.ID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s님이 백팀에 참여했습니다.", User.Username),
			},
		})
	} else {
		h.Game.AddBlackPlayer(User.ID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s님이 흑팀에 참여했습니다.", User.Username),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func (h *InteractionHandler) handleMoveCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var moveUCI string
	for _, opt := range options {
		if opt.Name == "move_uci" {
			moveUCI = opt.StringValue()
			break
		}
	}

	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	if errMsg := CheckPlayerAndTurn(h.Game, User.ID); errMsg != "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if moveUCI != "" {
		// If move_uci is provided, attempt to vote for it
		err := h.Game.VoteMove(User.ID, moveUCI)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("잘못된 수: `%s`. `/move`를 사용하여 가능한 수를 확인하세요.", moveUCI),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s님이 **%s**에 투표했습니다.", User.Username, moveUCI),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	} else {
		// If no move_uci, display the initial move embed
		team, _ := h.Game.GetPlayerTeam(User.ID)
		messageToSend, err := chess.CreateInitialMoveEmbed(h.Game.ChessGame, User.ID, team, true)
		if err != nil {
			fmt.Printf("Error creating initial move embed: %v\n", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "초기 이동 임베드를 생성하는 중 오류가 발생했습니다.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		messageToSend.Flags = discordgo.MessageFlagsEphemeral // Ensure it's ephemeral
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    messageToSend.Content,
				Embeds:     messageToSend.Embeds,
				Components: messageToSend.Components,
				Files:      messageToSend.Files,
				Flags:      messageToSend.Flags,
			},
		})
	}
}

func (h *InteractionHandler) handleSkipCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if errMsg := CheckPlayerAndTurn(h.Game, i.Member.User.ID); errMsg != "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Defer the response first
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		fmt.Printf("Error deferring interaction response: %v\n", err)
		return
	}

	resultMsg := h.Game.Next()

	team, _ := h.Game.GetPlayerTeam(i.Member.User.ID)
	fen := h.Game.ChessGame.FEN()
	if team == "black" {
		parts := strings.Split(fen, " ")
		runes := []rune(parts[0])
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		parts[0] = string(runes)
		fen = strings.Join(parts, " ")
	}

	file := chess.ChessImage(fen, h.Game.GetVotes(), team)
	var message string
	if resultMsg != "" {
		message = resultMsg
	} else {
		message = fmt.Sprintf("턴이 스킵되었습니다. 다음 플레이어의 턴입니다.\n\n%s", h.Game.GetTopNVotes(3))
	}

	// Edit the deferred response with the actual content and file
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
		Files: []*discordgo.File{
			{
				Name:        "chess.png",
				ContentType: "image/png",
				Reader:      file,
			},
		},
	})
	if err != nil {
		fmt.Printf("Error editing interaction response: %v\n", err)
		return
	}
}

func (h *InteractionHandler) handleMovePage(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		fmt.Printf("Error responding to interaction: %v\n", err)
		return
	}

	pageStr := strings.TrimPrefix(customID, chess.PrefixMovePage)
	page, _ := strconv.Atoi(pageStr)
	team, _ := h.Game.GetPlayerTeam(User.ID)

	messageToEdit, err := chess.CreatePaginationMessageEdit(h.Game.ChessGame, page, h.Game.GetVotes(), User.ID, team)
	if err != nil {
		fmt.Printf("Error creating pagination embed: %v\n", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: StrPtr("이동 페이지를 생성하는 중 오류가 발생했습니다."),
		})
		return
	}
	s.InteractionResponseEdit(i.Interaction, MessageEditToWebhookEdit(messageToEdit))
}

func (h *InteractionHandler) handleMoveSelect(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		fmt.Printf("Error responding to interaction: %v\n", err)
		return
	}

	moveStr := strings.TrimPrefix(customID, chess.PrefixMoveSelect)
	team, _ := h.Game.GetPlayerTeam(User.ID)

	messageToEdit, err := chess.CreateMovePreviewEmbed(h.Game.ChessGame, moveStr, User.ID, team)
	if err != nil {
		fmt.Printf("Error creating move preview embed: %v\n", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: StrPtr("이동 미리보기를 생성하는 중 오류가 발생했습니다."),
		})
		return
	}
	s.InteractionResponseEdit(i.Interaction, MessageEditToWebhookEdit(messageToEdit))
}

func (h *InteractionHandler) handleMoveVote(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	moveStr := strings.TrimPrefix(customID, chess.PrefixMoveVote)
	err := h.Game.VoteMove(User.ID, moveStr)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "투표 중 오류가 발생했습니다.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	move, err := notnilchess.UCINotation{}.Decode(h.Game.ChessGame.Position(), moveStr)
	var san string
	if err != nil {
		san = moveStr
	} else {
		san = notnilchess.AlgebraicNotation{}.Encode(h.Game.ChessGame.Position(), move)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("**%s**에 대한 투표가 완료되었습니다!", san),
		},
	})
}

func (h *InteractionHandler) handleMoveCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var User *discordgo.User

	if i.Member == nil {
		User = i.User
	} else {
		User = i.Member.User
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		fmt.Printf("Error responding to interaction: %v\n", err)
		return
	}

	team, _ := h.Game.GetPlayerTeam(User.ID)

	messageToEdit, err := chess.CreatePaginationMessageEdit(h.Game.ChessGame, 0, h.Game.GetVotes(), User.ID, team)
	if err != nil {
		fmt.Printf("Error creating pagination embed: %v\n", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: StrPtr("이동 목록으로 돌아가는 중 오류가 발생했습니다."),
		})
		return
	}
	s.InteractionResponseEdit(i.Interaction, MessageEditToWebhookEdit(messageToEdit))
}
