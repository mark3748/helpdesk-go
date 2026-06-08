package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	ws "github.com/mark3748/helpdesk-go/cmd/api/ws"
)

var dgSession atomic.Pointer[discordgo.Session]

const (
	discordLinkChallengeTTLMinutes = 15
	discordLinkChallengeLimit      = 3
	discordCreateTicketModalID     = "create_ticket_modal"
	discordTicketTitleInputID      = "ticket_title"
	discordTicketDescInputID       = "ticket_desc"
	discordTicketPriorityInputID   = "ticket_priority"
)

var (
	errDiscordLinkRateLimited = errors.New("too many verification requests; try again later")
	errDiscordLinkInvalid     = errors.New("invalid or expired verification token")
)

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
	}
	if discordEmailLinkEnabled(c) {
		commands = append(commands, &discordgo.ApplicationCommand{
			Name:        "link-email",
			Description: "Send an email verification token for account linking",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "email",
					Description: "Your email address",
					Required:    true,
				},
			},
		}, &discordgo.ApplicationCommand{
			Name:        "verify-email",
			Description: "Complete email verification and link your account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "token",
					Description: "Verification token sent to your email",
					Required:    true,
				},
			},
		})
	} else {
		log.Warn().Msg("Discord email linking disabled because SMTP_HOST or SMTP_FROM is not configured")
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

	cleanupTicker := time.NewTicker(time.Hour)
	defer cleanupTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			goto shutdown
		case <-cleanupTicker.C:
			if err := cleanupExpiredDiscordLinkChallenges(ctx, db); err != nil {
				log.Error().Err(err).Msg("cleanup expired Discord link challenges")
			}
		}
	}

shutdown:
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
					CustomID: discordCreateTicketModalID,
					Title:    "Create Support Ticket",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    discordTicketTitleInputID,
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
									CustomID:    discordTicketDescInputID,
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
									CustomID:    discordTicketPriorityInputID,
									Label:       "Priority (1=Critical, 2=High, 3=Medium, 4=Low)",
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
			if !discordEmailLinkEnabled(c) {
				respondInteractionError(s, i, "❌ Email linking is not configured.")
				return
			}
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
			err := beginDiscordEmailLink(ctx, discordUserID, emailOpt, db, rdb)

			var respMsg string
			if err != nil {
				if errors.Is(err, errDiscordLinkRateLimited) {
					respMsg = "❌ Too many verification requests. Please try again later."
				} else {
					respMsg = "❌ Unable to start email verification."
					log.Error().Err(err).Str("discord_user_id", discordUserID).Msg("start Discord email verification")
				}
			} else {
				respMsg = "✅ If that address can receive email, a verification token has been sent. Use `/verify-email` within 15 minutes."
			}

			respondInteractionError(s, i, respMsg)

		case "verify-email":
			if len(data.Options) == 0 {
				respondInteractionError(s, i, "❌ Missing required verification token.")
				return
			}
			if i.Member == nil || i.Member.User == nil {
				respondInteractionError(s, i, "❌ Unable to identify your Discord user.")
				return
			}

			discordUserID := i.Member.User.ID
			err := completeDiscordEmailLink(ctx, discordUserID, i.Member.User.Username, data.Options[0].StringValue(), db)
			if errors.Is(err, errDiscordLinkInvalid) {
				respondInteractionError(s, i, "❌ Invalid or expired verification token.")
				return
			}
			if err != nil {
				log.Error().Err(err).Str("discord_user_id", discordUserID).Msg("complete Discord email verification")
				respondInteractionError(s, i, "❌ Unable to complete email verification.")
				return
			}
			respondInteractionError(s, i, "✅ Your Discord account is now linked to the verified email address.")
		}

	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()
		if data.CustomID == discordCreateTicketModalID {
			if i.Member == nil || i.Member.User == nil {
				respondInteractionError(s, i, "❌ Unable to identify your Discord user.")
				return
			}
			inputs := discordModalTextInputValues(data)
			title := strings.TrimSpace(inputs[discordTicketTitleInputID])
			desc := strings.TrimSpace(inputs[discordTicketDescInputID])
			priorityStr := strings.TrimSpace(inputs[discordTicketPriorityInputID])
			if title == "" || desc == "" {
				log.Warn().
					Bool("missing_title", title == "").
					Bool("missing_description", desc == "").
					Msg("rejecting create-ticket modal with missing required fields")
				respondInteractionError(s, i, "❌ Ticket title and description are required.")
				return
			}
			if priorityStr == "" {
				priorityStr = "2"
			}

			parsedPriority, err := strconv.ParseInt(priorityStr, 10, 16)
			priority := int16(2)
			if err == nil && parsedPriority >= 1 && parsedPriority <= 4 {
				priority = int16(parsedPriority)
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

func discordModalTextInputValues(data discordgo.ModalSubmitInteractionData) map[string]string {
	values := make(map[string]string)
	for _, component := range data.Components {
		var children []discordgo.MessageComponent
		switch row := component.(type) {
		case *discordgo.ActionsRow:
			children = row.Components
		case discordgo.ActionsRow:
			children = row.Components
		default:
			continue
		}
		for _, component := range children {
			switch input := component.(type) {
			case *discordgo.TextInput:
				values[input.CustomID] = input.Value
			case discordgo.TextInput:
				values[input.CustomID] = input.Value
			}
		}
	}
	return values
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

func discordEmailLinkEnabled(c Config) bool {
	return strings.TrimSpace(c.SMTPHost) != "" && strings.TrimSpace(c.SMTPFrom) != ""
}

// beginDiscordEmailLink creates a short-lived challenge and emails the plaintext token.
func beginDiscordEmailLink(ctx context.Context, discordUserID, targetEmail string, db app.DB, rdb *redis.Client) error {
	email, err := sanitizeAndValidateEmail(strings.ToLower(strings.TrimSpace(targetEmail)))
	if err != nil {
		return err
	}
	if rdb == nil {
		return errors.New("Redis is required for Discord email verification")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256([]byte(token))

	var challengeID string
	err = db.QueryRow(ctx, `
		with allowed as (
			select count(*) < $4 as ok
			from discord_link_challenges
			where created_at > now() - interval '1 hour'
			  and (discord_user_id = $1 or email = $2)
		),
		invalidated as (
			update discord_link_challenges
			set consumed_at = now()
			where discord_user_id = $1
			  and consumed_at is null
			  and (select ok from allowed)
			returning id
		),
		inserted as (
			insert into discord_link_challenges (discord_user_id, email, token_hash, expires_at)
			select $1, $2, $3, now() + make_interval(mins => $5)
			where (select ok from allowed)
			returning id::text
		)
		select id from inserted
	`, discordUserID, email, tokenHash[:], discordLinkChallengeLimit, discordLinkChallengeTTLMinutes).Scan(&challengeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errDiscordLinkRateLimited
	}
	if err != nil {
		return fmt.Errorf("create Discord link challenge: %w", err)
	}

	jobData, err := json.Marshal(EmailJob{
		To:       email,
		Template: "discord_link_verification",
		Data: map[string]any{
			"Token":     token,
			"ExpiresIn": fmt.Sprintf("%d minutes", discordLinkChallengeTTLMinutes),
		},
	})
	if err != nil {
		return fmt.Errorf("marshal verification email: %w", err)
	}
	job, err := json.Marshal(Job{Type: "send_email", Data: jobData})
	if err != nil {
		return fmt.Errorf("marshal verification email job: %w", err)
	}
	if err := rdb.RPush(ctx, "jobs", job).Err(); err != nil {
		_, _ = db.Exec(ctx, "update discord_link_challenges set consumed_at = now() where id = $1", challengeID)
		return fmt.Errorf("enqueue verification email: %w", err)
	}
	return nil
}

// completeDiscordEmailLink atomically consumes a challenge and creates the verified mapping.
func completeDiscordEmailLink(ctx context.Context, discordUserID, username, token string, db app.DB) error {
	tokenHash := sha256.Sum256([]byte(strings.TrimSpace(token)))

	var requesterID string
	err := db.QueryRow(ctx, `
		with consumed as (
			update discord_link_challenges
			set consumed_at = now()
			where discord_user_id = $1
			  and token_hash = $2
			  and consumed_at is null
			  and expires_at > now()
			returning email
		),
		existing_requester as (
			select r.id
			from requesters r
			join consumed c on lower(r.email) = c.email
			limit 1
		),
		created_requester as (
			insert into requesters (email, name)
			select c.email, $3
			from consumed c
			where not exists (select 1 from existing_requester)
			on conflict (email) do update set email = excluded.email
			returning id
		),
		requester as (
			select id from existing_requester
			union all
			select id from created_requester
		),
		mapped as (
			insert into discord_user_mappings (discord_user_id, requester_id)
			select $1, id from requester
			on conflict (discord_user_id) do update set requester_id = excluded.requester_id
			returning requester_id::text
		)
		select requester_id from mapped
	`, discordUserID, tokenHash[:], username).Scan(&requesterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errDiscordLinkInvalid
	}
	if err != nil {
		return fmt.Errorf("complete Discord link challenge: %w", err)
	}
	return nil
}

func cleanupExpiredDiscordLinkChallenges(ctx context.Context, db app.DB) error {
	_, err := db.Exec(ctx, `
		delete from discord_link_challenges
		where expires_at < now() - interval '24 hours'
		   or consumed_at < now() - interval '24 hours'
	`)
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

	s := dgSession.Load()
	if s == nil {
		return errors.New("discord bot session is not available")
	}

	msg := fmt.Sprintf("💬 **New Comment:**\n%s", bodyMD)
	_, err = s.ChannelMessageSend(threadID, msg)
	return err
}
