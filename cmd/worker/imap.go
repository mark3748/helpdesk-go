package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	app "github.com/mark3748/helpdesk-go/cmd/api/app"
	ws "github.com/mark3748/helpdesk-go/cmd/api/ws"
)

type imapClient interface {
	Login(username, password string) error
	Select(mailbox string, readOnly bool) (*imap.MailboxStatus, error)
	Search(criteria *imap.SearchCriteria) ([]uint32, error)
	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	Store(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error
	Logout() error
}

var dialIMAP = func(addr string) (imapClient, error) {
	return imapclient.DialTLS(addr, nil)
}

var processIMAP = processIMAPMessage

// pollIMAP connects to an IMAP inbox, retrieves new messages and stores them.
func pollIMAP(ctx context.Context, c Config, db app.DB, store app.ObjectStore, rdb *redis.Client) error {
	if c.MinIOBucket != "" && store == nil {
		return fmt.Errorf("object store is nil")
	}
	addr := fmt.Sprintf("%s:993", c.IMAPHost)
	cli, err := dialIMAP(addr)
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

		if err := processIMAP(ctx, c, db, store, rdb, raw); err != nil {
			log.Error().Err(err).Msg("process message")
		}

		seq := new(imap.SeqSet)
		seq.AddNum(msg.SeqNum)
		if err := cli.Store(seq, imap.AddFlags, []interface{}{imap.SeenFlag}, nil); err != nil {
			log.Error().Err(err).Msg("store flags")
		}
	}
	return <-done
}

// processIMAPMessage parses and stores a single email message.
func processIMAPMessage(ctx context.Context, c Config, db app.DB, store app.ObjectStore, rdb *redis.Client, raw []byte) error {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	msgID := strings.TrimSpace(mr.Header.Get("Message-Id"))
	if msgID != "" {
		var existing int64
		if err := db.QueryRow(ctx, "select ticket_id from email_inbound where message_id=$1", msgID).Scan(&existing); err == nil {
			return nil
		}
	}

	subject := sanitizeEmailHeader(mr.Header.Get("Subject"))
	from := sanitizeEmailHeader(mr.Header.Get("From"))

	type att struct {
		name string
		mime string
		data []byte
	}
	var atts []att
	var body string

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			b, _ := io.ReadAll(part.Body)
			if body == "" && strings.HasPrefix(ct, "text/") {
				body = sanitizeEmailBody(b)
			}
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			filename = sanitizeAttachmentName(filename)
			if filename == "" {
				log.Error().Msg("attachment filename invalid")
				continue
			}
			ct, _, _ := h.ContentType()
			b, _ := io.ReadAll(part.Body)
			atts = append(atts, att{name: filename, mime: ct, data: b})
		}
	}

	var ticketID int64
	re := regexp.MustCompile(`\[TKT-(\d+)\]`)
	if match := re.FindStringSubmatch(subject); len(match) == 2 {
		if n, err := strconv.Atoi(match[1]); err == nil {
			if err := db.QueryRow(ctx, "select id from tickets where number=$1", n).Scan(&ticketID); err != nil {
				ticketID = 0
			}
		}
	}

	created := false
	if ticketID == 0 {
		if err := db.QueryRow(ctx, "insert into tickets (title, description, status) values ($1,$2,'New') returning id", subject, body).Scan(&ticketID); err != nil {
			return err
		}
		created = true
	} else {
		if _, err := db.Exec(ctx, "insert into ticket_comments (ticket_id, body_md, is_internal) values ($1,$2,false)", ticketID, body); err != nil {
			log.Error().Err(err).Msg("insert comment")
		}
	}
	if created {
		if rdb != nil {
			ej := EmailJob{To: from, Template: "ticket_created", Data: map[string]any{"Number": ticketID}}
			b, _ := json.Marshal(ej)
			nb, _ := json.Marshal(Job{Type: "send_email", Data: b})
			_ = rdb.RPush(ctx, "jobs", nb).Err()
		}
		ws.PublishEvent(ctx, rdb, ws.Event{Type: "ticket_created", Data: map[string]interface{}{"id": ticketID}})
	} else {
		ws.PublishEvent(ctx, rdb, ws.Event{Type: "ticket_updated", Data: map[string]interface{}{"id": ticketID}})
	}

	var attMeta []map[string]interface{}
	for _, a := range atts {
		if store != nil && c.MinIOBucket != "" {
			fname := sanitizeAttachmentName(a.name)
			if fname == "" {
				log.Error().Msg("attachment filename invalid")
				continue
			}
			key := fmt.Sprintf("attachments/%s/%s", uuid.NewString(), fname)
			if _, err := store.PutObject(ctx, c.MinIOBucket, key, bytes.NewReader(a.data), int64(len(a.data)), minio.PutObjectOptions{ContentType: a.mime}); err != nil {
				log.Error().Err(err).Msg("put attachment")
			} else {
				if _, err := db.Exec(ctx, "insert into attachments (ticket_id, uploader_id, object_key, filename, bytes, mime) values ($1,$2,$3,$4,$5,$6)", ticketID, uuid.Nil, key, fname, len(a.data), a.mime); err != nil {
					log.Error().Err(err).Msg("insert attachment")
				}
				attMeta = append(attMeta, map[string]interface{}{"filename": fname, "object_key": key})
			}
		}
	}

	parsed := map[string]interface{}{
		"subject":     subject,
		"from":        from,
		"attachments": attMeta,
	}
	pj, err := json.Marshal(parsed)
	if err != nil {
		return err
	}
	rawKey := fmt.Sprintf("email/%s.eml", uuid.NewString())
	if store != nil && c.MinIOBucket != "" {
		if _, err := store.PutObject(ctx, c.MinIOBucket, rawKey, bytes.NewReader(raw), int64(len(raw)), minio.PutObjectOptions{}); err != nil {
			log.Error().Err(err).Msg("put object")
		}
	}
	if _, err := db.Exec(ctx, "insert into email_inbound (raw_store_key, parsed_json, message_id, status, ticket_id) values ($1,$2,$3,'processed',$4)", rawKey, pj, msgID, ticketID); err != nil {
		log.Error().Err(err).Msg("insert email_inbound")
	}
	return nil
}
