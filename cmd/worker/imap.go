package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strconv"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"
)

// pollIMAP connects to an IMAP inbox, retrieves new messages and stores them.
func pollIMAP(ctx context.Context, c Config, db *pgxpool.Pool, mc *minio.Client) error {
	if c.MinIOBucket != "" && mc == nil {
		return fmt.Errorf("MinIO client is nil")
	}
	addr := fmt.Sprintf("%s:993", c.IMAPHost)
	cli, err := imapclient.DialTLS(addr, nil)
	if err != nil {
		return err
	}
	defer cli.Logout()

	if err := cli.Login(c.IMAPUser, c.IMAPPass); err != nil {
		return err
	}

	mbox, err := cli.Select(c.IMAPFolder, false)
	if err != nil {
		return err
	}
	if mbox.Messages == 0 {
		return nil
	}

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := cli.Search(criteria)
	if err != nil || len(uids) == 0 {
		return err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- cli.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}, messages)
	}()

	for msg := range messages {
		if msg == nil {
			continue
		}
		r := msg.GetBody(section)
		if r == nil {
			continue
		}
		raw, err := io.ReadAll(r)
		if err != nil {
			log.Error().Err(err).Msg("read body")
			continue
		}

		key := fmt.Sprintf("email/%s.eml", uuid.NewString())
		if c.MinIOBucket != "" {
			_, err = mc.PutObject(ctx, c.MinIOBucket, key, bytes.NewReader(raw), int64(len(raw)), minio.PutObjectOptions{})
			if err != nil {
				log.Error().Err(err).Msg("put object")
			}
		}
	}()

	for msg := range messages {
		if msg == nil {
			continue
		}
		r := msg.GetBody(section)
		if r == nil {
			continue
		}
		raw, err := io.ReadAll(r)
		if err != nil {
			log.Error().Err(err).Msg("read body")
			continue
		}

		key := fmt.Sprintf("email/%s.eml", uuid.NewString())
		if c.MinIOBucket != "" {
			_, err = mc.PutObject(ctx, c.MinIOBucket, key, bytes.NewReader(raw), int64(len(raw)), minio.PutObjectOptions{})
			if err != nil {
				log.Error().Err(err).Msg("put object")
			}
		}

>>>>>>> origin/codex/fix-comments-in-go.mod-file-8kt851:worker/cmd/worker/imap.go
		m, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			log.Error().Err(err).Msg("parse message")
			continue
		}
<<<<<<< HEAD:cmd/worker/imap.go
		subject := sanitizeEmailHeader(m.Header.Get("Subject"))
		from := sanitizeEmailHeader(m.Header.Get("From"))
		body, err := io.ReadAll(m.Body)
		if err != nil {
			log.Error().Err(err).Msg("read message body")
			continue
		}
		cleanBody := sanitizeEmailBody(body)

		var ticketID int64
		re := regexp.MustCompile(`\[TKT-(\d+)\]`)
		if match := re.FindStringSubmatch(subject); len(match) == 2 {
			if n, err := strconv.Atoi(match[1]); err == nil {
				if err := db.QueryRow(ctx, "select id from tickets where number=$1", n).Scan(&ticketID); err != nil {
					ticketID = 0
				}
			}
		}

		if ticketID == 0 {
			if err := db.QueryRow(ctx, "insert into tickets (title, description, status) values ($1,$2,'New') returning id", subject, cleanBody).Scan(&ticketID); err != nil {
				log.Error().Err(err).Msg("create ticket")
				continue
			}
		} else {
			if _, err := db.Exec(ctx, "insert into ticket_comments (ticket_id, body_md, is_internal) values ($1,$2,false)", ticketID, cleanBody); err != nil {
				log.Error().Err(err).Msg("insert comment")
			}
		}

		parsed := map[string]string{
			"subject": subject,
			"from":    from,
		}
		pj, err := json.Marshal(parsed)
		if err != nil {
			log.Error().Err(err).Msg("marshal parsed email")
			continue
		}
		if _, err := db.Exec(ctx, "insert into email_inbound (raw_store_key, parsed_json, status, ticket_id) values ($1,$2,'processed',$3)", key, pj, ticketID); err != nil {
			log.Error().Err(err).Msg("insert email_inbound")
		}

		seq := new(imap.SeqSet)
		seq.AddNum(msg.SeqNum)
		if err := cli.Store(seq, imap.AddFlags, []interface{}{imap.SeenFlag}, nil); err != nil {
			log.Error().Err(err).Msg("store flags")
		}
	}
	return <-done
}

=======
		subject := m.Header.Get("Subject")
		from := m.Header.Get("From")
		body, err := io.ReadAll(m.Body)
		if err != nil {
			log.Error().Err(err).Msg("read message body")
			continue
		}
		cleanBody := sanitizeEmailBody(body)

		var ticketID int64
		re := regexp.MustCompile(`\[TKT-(\d+)\]`)
		if match := re.FindStringSubmatch(subject); len(match) == 2 {
			if n, err := strconv.Atoi(match[1]); err == nil {
				if err := db.QueryRow(ctx, "select id from tickets where number=$1", n).Scan(&ticketID); err != nil {
					ticketID = 0
				}
			}
		}

		if ticketID == 0 {
			if err := db.QueryRow(ctx, "insert into tickets (title, description, status) values ($1,$2,'New') returning id", subject, cleanBody).Scan(&ticketID); err != nil {
				log.Error().Err(err).Msg("create ticket")
				continue
			}
		} else {
			if _, err := db.Exec(ctx, "insert into ticket_comments (ticket_id, body_md, is_internal) values ($1,$2,false)", ticketID, cleanBody); err != nil {
				log.Error().Err(err).Msg("insert comment")
			}
		}

		parsed := map[string]string{
			"subject": subject,
			"from":    from,
		}
		pj, err := json.Marshal(parsed)
		if err != nil {
			log.Error().Err(err).Msg("marshal parsed email")
			continue
		}
		if _, err := db.Exec(ctx, "insert into email_inbound (raw_store_key, parsed_json, status, ticket_id) values ($1,$2,'processed',$3)", key, pj, ticketID); err != nil {
			log.Error().Err(err).Msg("insert email_inbound")
		}

		seq := new(imap.SeqSet)
		seq.AddNum(msg.SeqNum)
		if err := cli.Store(seq, imap.AddFlags, []interface{}{imap.SeenFlag}, nil); err != nil {
			log.Error().Err(err).Msg("store flags")
		}
	}
	return <-done
}
>>>>>>> origin/codex/fix-comments-in-go.mod-file-8kt851:worker/cmd/worker/imap.go
