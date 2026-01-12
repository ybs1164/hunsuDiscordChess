package handlers

import (
	"fmt"
	"strconv"
	"strings"

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
		if len(parts) == 2 {
			authorID := parts[1]
			if authorID != i.Member.User.ID {
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
	if h.Game.IsGameOver() {
		h.Game.Reset()
	}

	if errMsg := CheckPlayer(h.Game, i.Member.User.ID); errMsg != "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

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
	message := fmt.Sprintf("하루가 지나갈 때 턴이 넘어갑니다.\n\n%s", h.Game.GetTopNVotes(3))

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
	member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "멤버 정보를 가져오는 중 오류가 발생했습니다.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// As direct Nitro detection (PremiumType) is unreliable, we check for features that require Nitro,
	// such as server boosting, animated avatar, banner, or avatar decoration.
	isPremium := (member.PremiumSince != nil) ||
		(member.User != nil && (strings.HasPrefix(member.User.Avatar, "a_") || member.User.Banner != ""))

	if isPremium {
		h.Game.AddWhitePlayer(i.Member.User.ID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s님이 백팀에 참여했습니다.", i.Member.User.Username),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	} else {
		h.Game.AddBlackPlayer(i.Member.User.ID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("%s님이 흑팀에 참여했습니다.", i.Member.User.Username),
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

	if moveUCI != "" {
		// If move_uci is provided, attempt to vote for it
		err := h.Game.VoteMove(i.Member.User.ID, moveUCI)
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
				Content: fmt.Sprintf("%s님이 **%s**에 투표했습니다.", i.Member.User.Username, moveUCI),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	} else {
		// If no move_uci, display the initial move embed
		team, _ := h.Game.GetPlayerTeam(i.Member.User.ID)
		messageToSend, err := chess.CreateInitialMoveEmbed(h.Game.ChessGame, i.Member.User.ID, team, true)
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

func (h *InteractionHandler) handleVoteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var moveUCI string
	for _, opt := range options {
		if opt.Name == "move_uci" {
			moveUCI = opt.StringValue()
			break
		}
	}

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

	err := h.Game.VoteMove(i.Member.User.ID, moveUCI)
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

	move, err := notnilchess.UCINotation{}.Decode(h.Game.ChessGame.Position(), moveUCI)
	var san string
	if err != nil {
		san = moveUCI
	} else {
		san = notnilchess.AlgebraicNotation{}.Encode(h.Game.ChessGame.Position(), move)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s님이 **%s**에 투표했습니다.", i.Member.User.Username, san),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *InteractionHandler) handleMovePage(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		fmt.Printf("Error responding to interaction: %v\n", err)
		return
	}

	pageStr := strings.TrimPrefix(customID, chess.PrefixMovePage)
	page, _ := strconv.Atoi(pageStr)
	team, _ := h.Game.GetPlayerTeam(i.Member.User.ID)

	messageToEdit, err := chess.CreatePaginationMessageEdit(h.Game.ChessGame, page, h.Game.GetVotes(), i.Member.User.ID, team)
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
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		fmt.Printf("Error responding to interaction: %v\n", err)
		return
	}

	moveStr := strings.TrimPrefix(customID, chess.PrefixMoveSelect)
	team, _ := h.Game.GetPlayerTeam(i.Member.User.ID)

	messageToEdit, err := chess.CreateMovePreviewEmbed(h.Game.ChessGame, moveStr, i.Member.User.ID, team)
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
	moveStr := strings.TrimPrefix(customID, chess.PrefixMoveVote)
	err := h.Game.VoteMove(i.Member.User.ID, moveStr)
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
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		fmt.Printf("Error responding to interaction: %v\n", err)
		return
	}

	team, _ := h.Game.GetPlayerTeam(i.Member.User.ID)

	messageToEdit, err := chess.CreatePaginationMessageEdit(h.Game.ChessGame, 0, h.Game.GetVotes(), i.Member.User.ID, team)
	if err != nil {
		fmt.Printf("Error creating pagination embed: %v\n", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: StrPtr("이동 목록으로 돌아가는 중 오류가 발생했습니다."),
		})
		return
	}
	s.InteractionResponseEdit(i.Interaction, MessageEditToWebhookEdit(messageToEdit))
}
