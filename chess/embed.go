package chess

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/notnil/chess"
)

const (
	// Defines how many move buttons to show per page. Discord limits to 5x5=25 components.
	// We use 4 rows for moves and 1 for navigation, so 20 moves per page.
	movesPerPage = 10

	// CustomID prefixes for button interactions
	PrefixMovePage   = "move_page_"
	PrefixMoveSelect = "move_select_"
	PrefixMoveVote   = "move_vote_"
	PrefixMoveCancel = "move_cancel_"
)

// Generates the initial, paginated embed listing available moves as buttons.
func CreateInitialMoveEmbed(g *chess.Game, userID string, team string, ephemeral bool) (*discordgo.MessageSend, error) {
	return createMoveListPage(g, 0, userID, team, ephemeral)
}

// Generates an embed that shows a preview of a board state after a specific move.
func CreateMovePreviewEmbed(g *chess.Game, moveStr string, userID string, team string) (*discordgo.MessageEdit, error) {
	// Create a copy of the game to apply the move without changing the main game state.
	gameCopy := g.Clone()
	m, _ := chess.UCINotation{}.Decode(g.Position(), moveStr)
	err := gameCopy.Move(m)
	if err != nil {
		return nil, fmt.Errorf("invalid move for preview: %w", err)
	}

	fen := gameCopy.FEN()
	if team == "black" {
		parts := strings.Split(fen, " ")
		runes := []rune(parts[0])
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		parts[0] = string(runes)
		fen = strings.Join(parts, " ")
	}

	// Generate image for the new state.
	imageReader := ChessImage(fen, []string{}, team) // No votes shown in preview
	imageName := fmt.Sprintf("chess-%d.png", time.Now().UnixNano())

	san := chess.AlgebraicNotation{}.Encode(g.Position(), m)

	embed := &discordgo.MessageEmbed{
		Title:       "Move Preview",
		Description: fmt.Sprintf("This is the board if the move **%s** is played.", san),
		Color:       0x0000ff, // Blue for preview
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://" + imageName,
		},
	}

	// Create "Vote" and "Back" buttons
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Vote for this Move",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("%s%s;%s", PrefixMoveVote, moveStr, userID),
				},
				discordgo.Button{
					Label:    "Back to List",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("%s;%s", PrefixMoveCancel, userID),
				},
			},
		},
	}

	// We need to read the image into a buffer to send it with MessageEdit.
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, imageReader); err != nil {
		return nil, fmt.Errorf("failed to read image buffer: %w", err)
	}

	msgEdit := &discordgo.MessageEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
		Files: []*discordgo.File{
			{
				Name:        imageName,
				ContentType: "image/png",
				Reader:      buf,
			},
		},
		Attachments: &[]*discordgo.MessageAttachment{},
	}
	return msgEdit, nil
}

// Helper function that creates a specific page of the move list.
func createMoveListPage(g *chess.Game, page int, userID string, team string, ephemeral bool) (*discordgo.MessageSend, error) {
	validMoves := g.ValidMoves()
	if len(validMoves) == 0 {
		return &discordgo.MessageSend{Content: "No valid moves available."}, nil
	}

	// Pagination
	numPages := int(math.Ceil(float64(len(validMoves)) / float64(movesPerPage)))
	start := page * movesPerPage
	end := start + movesPerPage
	if end > len(validMoves) {
		end = len(validMoves)
	}
	pagedMoves := validMoves[start:end]

	fen := g.FEN()
	if team == "black" {
		parts := strings.Split(fen, " ")
		runes := []rune(parts[0])
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		parts[0] = string(runes)
		fen = strings.Join(parts, " ")
	}

	// Generate the current board image with votes
	imageReader := ChessImage(fen, getVoteStrings(g), team)

	embed := &discordgo.MessageEmbed{
		Title:       "Available Moves",
		Description: "Select a move to see a preview, then vote.",
		Color:       0x00ff00, // Green
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://chess.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d of %d", page+1, numPages),
		},
	}

	// Build Button Components
	var components []discordgo.MessageComponent
	actionRows := buildMoveButtonRows(g, pagedMoves, userID)
	components = append(components, actionRows...)

	// Navigation buttons
	navRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("%s%d;%s", PrefixMovePage, page-1, userID),
				Disabled: page == 0,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("%s%d;%s", PrefixMovePage, page+1, userID),
				Disabled: page >= numPages-1,
			},
		},
	}
	components = append(components, navRow)

	msg := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
		Files: []*discordgo.File{
			{
				Name:        "chess.png",
				ContentType: "image/png",
				Reader:      imageReader,
			},
		},
	}

	if ephemeral {
		msg.Flags = discordgo.MessageFlagsEphemeral
	}

	return msg, nil
}

// This function is not exported, it's a helper for createMoveListPage
func buildMoveButtonRows(g *chess.Game, moves []*chess.Move, userID string) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent
	var currentRow discordgo.ActionsRow

	for i, move := range moves {
		if i%5 == 0 {
			if len(currentRow.Components) > 0 {
				rows = append(rows, currentRow)
			}
			currentRow = discordgo.ActionsRow{}
		}
		san := chess.AlgebraicNotation{}.Encode(g.Position(), move)
		currentRow.Components = append(currentRow.Components, discordgo.Button{
			Label:    san,
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("%s%s;%s", PrefixMoveSelect, move.String(), userID),
		})
	}
	if len(currentRow.Components) > 0 {
		rows = append(rows, currentRow)
	}
	return rows
}

// A helper to get vote strings from the game, needed for the image.
// This is duplicated from main.go to avoid circular dependencies.
// A better design would be to move the Game struct to the chess package.
func getVoteStrings(g *chess.Game) []string {
	// This is a placeholder. In a real scenario, you'd need access to the vote map.
	// For now, it returns an empty list. The interaction handler in main.go
	// will need to supply the real votes.
	return []string{}
}

// CreatePaginationMessageEdit is used to update the message for page navigation
func CreatePaginationMessageEdit(g *chess.Game, page int, votes []string, userID string, team string) (*discordgo.MessageEdit, error) {
	validMoves := g.ValidMoves()
	if len(validMoves) == 0 {
		return &discordgo.MessageEdit{Content: strPtr("No valid moves available.")}, nil
	}

	numPages := int(math.Ceil(float64(len(validMoves)) / float64(movesPerPage)))
	start := page * movesPerPage
	end := start + movesPerPage
	if end > len(validMoves) {
		end = len(validMoves)
	}
	pagedMoves := validMoves[start:end]

	fen := g.FEN()
	if team == "black" {
		parts := strings.Split(fen, " ")
		runes := []rune(parts[0])
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		parts[0] = string(runes)
		fen = strings.Join(parts, " ")
	}

	imageReader := ChessImage(fen, votes, team)
	imageName := fmt.Sprintf("chess-%d.png", time.Now().UnixNano())

	embed := &discordgo.MessageEmbed{
		Title:       "Available Moves",
		Description: "Select a move to see a preview, then vote.",
		Color:       0x00ff00, // Green
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://" + imageName,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d of %d", page+1, numPages),
		},
	}

	var components []discordgo.MessageComponent
	actionRows := buildMoveButtonRows(g, pagedMoves, userID)
	components = append(components, actionRows...)

	// Navigation buttons
	navRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("%s%d;%s", PrefixMovePage, page-1, userID),
				Disabled: page == 0,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("%s%d;%s", PrefixMovePage, page+1, userID),
				Disabled: page >= numPages-1,
			},
		},
	}
	components = append(components, navRow)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, imageReader); err != nil {
		return nil, fmt.Errorf("failed to read image buffer: %w", err)
	}

	msgEdit := &discordgo.MessageEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
		Files: []*discordgo.File{
			{
				Name:        imageName,
				ContentType: "image/png",
				Reader:      buf,
			},
		},
		Attachments: &[]*discordgo.MessageAttachment{},
	}
	return msgEdit, nil
}

func strPtr(s string) *string {
	return &s
}
