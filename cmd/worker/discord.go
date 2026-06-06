package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	ws "github.com/mark3748/helpdesk-go/cmd/api/ws"
)

var dgSession atomic.Pointer[discordgo.Session]

// runDiscordBot connects to the Discord gateway, registers slash commands, and processes events.
func runDiscordBot(ctx context.Context, c Config, db app.DB, store app.ObjectStore, rdb *redis.Client) error {
	if strings.TrimSpace(c.DiscordGuildID) == "" {
		return errors.New("DISCORD_GUILD_ID is required when Discord bot is enabled")
	}
	if strings.TrimSpace(c.DiscordChannelID) == "" {
		return errors.New("DISCORD_CHANNEL_ID is required when Discord bot is enabled")
	}

	s, err := discordgo.New("Bot " + c.DiscordBotToken)
	if err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent | discordgo.IntentsGuilds

	// Register message create handler
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handleMessageCreate(ctx, s, m, db, rdb)
	})

	// Register interaction handler
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		handleInteractionCreate(ctx, s, i, c, db, rdb)
	})

	err = s.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}
	appID := ""
	if s.State != nil && s.State.Application != nil {
		appID = s.State.Application.ID
	}
	if appID == "" {
		application, appErr := s.Application("@me")
		if appErr != nil || application == nil || application.ID == "" {
			_ = s.Close()
			if appErr != nil {
				return fmt.Errorf("resolve bot application id: %w", appErr)
			}
			return errors.New("resolve bot application id: missing application id")
		}
		appID = application.ID
	}
	dgSession.Store(s)
	defer func() {
		dgSession.Store(nil)
		_ = s.Close()
	}()

	log.Info().Msg("Discord bot connected successfully")

	// Register commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "create-ticket",
			Description: "Create a new support ticket",
		},
		{
			Name:        "link-email",
			Description: "Link your Discord account to a Helpdesk email address",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "email",
					Description: "Your email address",
					Required:    true,
				},
			},
		},
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for idx, cmd := range commands {
		reg, err := s.ApplicationCommandCreate(appID, c.DiscordGuildID, cmd)
		if err != nil {
			log.Error().Err(err).Str("command", cmd.Name).Msg("cannot create command")
		} else {
			registeredCommands[idx] = reg
		}
	}

	// Wait for context to end
	<-ctx.Done()

	// Cleanup registered commands
	for _, cmd := range registeredCommands {
		if cmd != nil {
			_ = s.ApplicationCommandDelete(appID, c.DiscordGuildID, cmd.ID)
		}
	}

	log.Info().Msg("Discord bot shutting down")
	return nil
}

// handleInteractionCreate dispatches commands and modal submissions.
func handleInteractionCreate(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, c Config, db app.DB, rdb *redis.Client) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		switch data.Name {
		case "create-ticket":
			// Show interactive creation modal
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "create_ticket_modal",
					Title:    "Create Support Ticket",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "ticket_title",
									Label:       "Title",
									Style:       discordgo.TextInputShort,
									Placeholder: "Brief summary of your request",
									Required:    true,
									MinLength:   3,
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "ticket_desc",
									Label:       "Description",
									Style:       discordgo.TextInputParagraph,
									Placeholder: "Provide details of the issue",
									Required:    true,
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "ticket_priority",
									Label:       "Priority (1=Low, 2=Medium, 3=High, 4=Urgent)",
									Style:       discordgo.TextInputShort,
									Placeholder: "2",
									Required:    true,
									MaxLength:   1,
								},
							},
						},
					},
				},
			})
			if err != nil {
				log.Error().Err(err).Msg("error responding with create-ticket modal")
			}

		case "link-email":
			if len(data.Options) == 0 {
				respondInteractionError(s, i, "❌ Missing required email option.")
				return
			}
			if i.Member == nil || i.Member.User == nil {
				respondInteractionError(s, i, "❌ Unable to identify your Discord user.")
				return
			}
			emailOpt := data.Options[0].StringValue()
			discordUserID := i.Member.User.ID
			username := i.Member.User.Username

			err := handleLinkEmail(ctx, discordUserID, username, emailOpt, db)

			var respMsg string
			if err != nil {
				respMsg = fmt.Sprintf("❌ Failed to link email: %v", err)
			} else {
				respMsg = fmt.Sprintf("✅ Successfully linked your Discord account to **%s**!", emailOpt)
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: respMsg,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}

	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()
		if data.CustomID == "create_ticket_modal" {
			if i.Member == nil || i.Member.User == nil {
				respondInteractionError(s, i, "❌ Unable to identify your Discord user.")
				return
			}
			title := ""
			desc := ""
			priorityStr := "2"

			for _, row := range data.Components {
				actionsRow, ok := row.(discordgo.ActionsRow)
				if !ok {
					continue
				}
				for _, comp := range actionsRow.Components {
					input, ok := comp.(discordgo.TextInput)
					if !ok {
						continue
					}
					switch input.CustomID {
					case "ticket_title":
						title = input.Value
					case "ticket_desc":
						desc = input.Value
					case "ticket_priority":
						priorityStr = input.Value
					}
				}
			}

			parsedPriority, err := strconv.ParseInt(priorityStr, 10, 16)
			priority := int16(2)
			if err == nil && parsedPriority >= 1 && parsedPriority <= 4 {
				priority = int16(parsedPriority)
			}

			if i.Member == nil || i.Member.User == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "❌ Unable to resolve user for this interaction.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			discordUserID := i.Member.User.ID
			username := i.Member.User.Username
			displayName := i.Member.Nick
			if displayName == "" {
				displayName = i.Member.User.GlobalName
			}
			if displayName == "" {
				displayName = username
			}

			ticketNum, _, threadID, err := handleCreateTicketFromDiscord(ctx, s, c, discordUserID, displayName, username, title, desc, priority, db)

			var responseData *discordgo.InteractionResponseData
			if err != nil {
				responseData = &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Failed to create ticket: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				}
			} else {
				responseData = &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("✅ Ticket **%s** has been created! Discuss it here: <#%s>", ticketNum, threadID),
					Flags:   discordgo.MessageFlagsEphemeral,
				}
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: responseData,
			})
		}
	}
}

func respondInteractionError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Error().Err(err).Msg("error sending interaction error response")
	}
}

// handleLinkEmail maps/updates Discord account to requester email.
func handleLinkEmail(ctx context.Context, discordUserID, username, targetEmail string, db app.DB) error {
	targetEmail = strings.ToLower(strings.TrimSpace(targetEmail))
	if targetEmail == "" {
		return errors.New("email cannot be empty")
	}

	// 1. Check if a requester with this email already exists
	var existingReqID string
	err := db.QueryRow(ctx, "select id::text from requesters where lower(email) = $1", targetEmail).Scan(&existingReqID)
	if err == nil {
		// Linked to existing requester. Upsert mapping.
		_, err = db.Exec(ctx, `
			insert into discord_user_mappings (discord_user_id, requester_id)
			values ($1, $2)
			on conflict (discord_user_id) do update set requester_id = excluded.requester_id
		`, discordUserID, existingReqID)
		return err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup requester by email: %w", err)
	}

	// 2. If not found in requesters, check if we have a mapping already
	var curReqID string
	err = db.QueryRow(ctx, "select requester_id::text from discord_user_mappings where discord_user_id=$1", discordUserID).Scan(&curReqID)
	if err == nil {
		// Update their current auto-created requester's email
		_, err = db.Exec(ctx, "update requesters set email = $1 where id=$2", targetEmail, curReqID)
		return err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup discord mapping: %w", err)
	}

	// 3. If no mapping exists, create new requester first
	newID := uuid.NewString()
	_, err = db.Exec(ctx, "insert into requesters (id, name, email) values ($1, $2, $3)", newID, username, targetEmail)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, "insert into discord_user_mappings (discord_user_id, requester_id) values ($1, $2)", discordUserID, newID)
	return err
}

// handleCreateTicketFromDiscord maps the user, inserts the ticket, and initializes the Discord thread.
func handleCreateTicketFromDiscord(ctx context.Context, s *discordgo.Session, c Config, discordUserID, displayName, username, title, desc string, priority int16, db app.DB) (string, string, string, error) {
	// 1. Resolve requester ID
	var requesterID string
	err := db.QueryRow(ctx, "select requester_id::text from discord_user_mappings where discord_user_id=$1", discordUserID).Scan(&requesterID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", fmt.Errorf("lookup discord user mapping: %w", err)
		}

		// Create guest requester
		newID := uuid.NewString()
		placeholderEmail := fmt.Sprintf("%s@discord.user", discordUserID)
		err = db.QueryRow(ctx, `
			insert into requesters (id, name, email)
			values ($1, $2, $3)
			on conflict (email) do update set name = coalesce(excluded.name, requesters.name)
			returning id::text
		`, newID, displayName, placeholderEmail).Scan(&requesterID)
		if err != nil {
			return "", "", "", fmt.Errorf("create guest requester: %w", err)
		}

		// Insert mapping
		_, err = db.Exec(ctx, `
			insert into discord_user_mappings (discord_user_id, requester_id)
			values ($1, $2)
			on conflict (discord_user_id) do update set requester_id = excluded.requester_id
		`, discordUserID, requesterID)
		if err != nil {
			return "", "", "", fmt.Errorf("create discord user mapping: %w", err)
		}
	}

	// 2. Reserve ticket number and create Discord thread before persisting ticket
	var seqNum int64
	err = db.QueryRow(ctx, "select nextval('ticket_seq')").Scan(&seqNum)
	if err != nil {
		return "", "", "", fmt.Errorf("reserve ticket number: %w", err)
	}
	ticketNum := fmt.Sprintf("HD-%d", seqNum)

	thread, err := s.ThreadStartComplex(c.DiscordChannelID, &discordgo.ThreadStart{
		Name:                fmt.Sprintf("%s: %s", ticketNum, title),
		AutoArchiveDuration: 1440,
		Type:                discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("start discord thread: %w", err)
	}

	// 3. Insert ticket
	var ticketID string
	const q = `
		insert into tickets (number, title, description, requester_id, priority, status, source)
		values ($1, $2, $3, $4, $5, 'New', 'discord')
		returning id::text`
	err = db.QueryRow(ctx, q, ticketNum, title, desc, requesterID, priority).Scan(&ticketID)
	if err != nil {
		if _, delErr := s.ChannelDelete(thread.ID); delErr != nil {
			log.Error().Err(delErr).Str("thread_id", thread.ID).Msg("failed to cleanup discord thread after ticket insert failure")
		}
		return "", "", "", fmt.Errorf("insert ticket: %w", err)
	}

	// Post initial ticket details in thread
	_, err = s.ChannelMessageSend(thread.ID, fmt.Sprintf("🎫 **Ticket Created: %s**\n**Title:** %s\n**Priority:** %d\n\n*Reply to this thread to add comments to this ticket.*", ticketNum, title, priority))
	if err != nil {
		log.Error().Err(err).Msg("error posting welcome message in discord thread")
	}

	// 4. Insert thread mapping
	_, err = db.Exec(ctx, `
		insert into discord_thread_mappings (discord_thread_id, ticket_id, channel_id)
		values ($1, $2, $3)
	`, thread.ID, ticketID, c.DiscordChannelID)
	if err != nil {
		if _, delErr := db.Exec(ctx, "delete from tickets where id=$1", ticketID); delErr != nil {
			log.Error().Err(delErr).Str("ticket_id", ticketID).Msg("failed to cleanup ticket after mapping insert failure")
		}
		if _, delErr := s.ChannelDelete(thread.ID); delErr != nil {
			log.Error().Err(delErr).Str("thread_id", thread.ID).Msg("failed to cleanup discord thread after mapping insert failure")
		}
		return "", "", "", fmt.Errorf("insert thread mapping: %w", err)
	}

	return ticketNum, ticketID, thread.ID, nil
}

// handleMessageCreate intercepts replies in Discord ticket threads and pushes them as comments.
func handleMessageCreate(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, db app.DB, rdb *redis.Client) {
	// Skip messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Query mapping to see if this message is inside a mapped thread
	var ticketID string
	err := db.QueryRow(ctx, "select ticket_id::text from discord_thread_mappings where discord_thread_id=$1", m.ChannelID).Scan(&ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not a mapped ticket thread, ignore
			return
		}
		log.Error().Err(err).Msg("failed to resolve discord thread mapping")
		return
	}

	// Resolve sender requester ID
	var requesterID string
	err = db.QueryRow(ctx, "select requester_id::text from discord_user_mappings where discord_user_id=$1", m.Author.ID).Scan(&requesterID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Error().Err(err).Msg("failed to resolve discord user mapping")
			return
		}

		// Create guest requester
		newID := uuid.NewString()
		placeholderEmail := fmt.Sprintf("%s@discord.user", m.Author.ID)
		displayName := m.Author.Username
		if m.Member != nil && m.Member.Nick != "" {
			displayName = m.Member.Nick
		} else if m.Author.GlobalName != "" {
			displayName = m.Author.GlobalName
		}
		err = db.QueryRow(ctx, `
			insert into requesters (id, name, email)
			values ($1, $2, $3)
			on conflict (email) do update set name = coalesce(excluded.name, requesters.name)
			returning id::text
		`, newID, displayName, placeholderEmail).Scan(&requesterID)
		if err != nil {
			log.Error().Err(err).Msg("failed to create guest requester")
			return
		}
		_, err = db.Exec(ctx, `
			insert into discord_user_mappings (discord_user_id, requester_id)
			values ($1, $2)
			on conflict (discord_user_id) do update set requester_id = excluded.requester_id
		`, m.Author.ID, requesterID)
		if err != nil {
			log.Error().Err(err).Msg("failed to upsert discord user mapping")
			return
		}
	}

	// Insert comment into ticket_comments
	_, err = db.Exec(ctx, `
		insert into ticket_comments (ticket_id, author_id, author_requester_id, body_md)
		values ($1, NULL, $2, $3)
	`, ticketID, requesterID, m.Content)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert comment from discord message")
		return
	}

	// Publish websocket event so UI updates instantly
	ws.PublishEvent(ctx, rdb, ws.Event{Type: "ticket_updated", Data: map[string]interface{}{"id": ticketID}})
}

// sendCommentToDiscord posts outbound comments from the Helpdesk Web UI to the Discord thread.
func sendCommentToDiscord(ctx context.Context, db app.DB, ticketID string, bodyMD string) error {
	s := dgSession.Load()
	if s == nil {
		return errors.New("discord bot session is not available")
	}

	// Find the mapped Discord thread ID
	var threadID string
	err := db.QueryRow(ctx, "select discord_thread_id from discord_thread_mappings where ticket_id=$1", ticketID).Scan(&threadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No Discord thread mapped for this ticket, ignore
			return nil
		}
		return fmt.Errorf("lookup discord thread mapping: %w", err)
	}

	msg := fmt.Sprintf("💬 **New Comment:**\n%s", bodyMD)
	_, err = s.ChannelMessageSend(threadID, msg)
	return err
}
