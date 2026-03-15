package source

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type attachment struct {
	name string
	data []byte
}

func fetchNewFaxes(cfg Config) ([]FaxEntry, error) {
	addr := "imap.gmail.com:993"
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer c.Logout()

	if err := c.Login(cfg.Email, cfg.Password); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	if mbox.Messages == 0 {
		return nil, nil
	}

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	if len(uids) == 0 {
		return nil, nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}
	msgs := make(chan *imap.Message, len(uids))
	if err := c.Fetch(seqset, items, msgs); err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	os.MkdirAll(receivedDir(), 0755)

	var entries []FaxEntry
	var markRead []uint32

	for msg := range msgs {
		from := ""
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}

		if !senderAllowed(from, cfg.Senders) {
			log.Printf("ignored sender: %s", from)
			continue
		}

		body := msg.GetBody(section)
		if body == nil {
			continue
		}

		atts, err := extractAttachments(body, cfg)
		if err != nil {
			log.Printf("extract error from %s: %v", from, err)
			continue
		}

		for _, att := range atts {
			path := filepath.Join(receivedDir(), att.name)
			if err := os.WriteFile(path, att.data, 0644); err != nil {
				log.Printf("write error: %v", err)
				continue
			}

			status := "printed"
			if err := printFile(path, cfg); err != nil {
				log.Printf("print error: %v", err)
				status = "print_failed"
			}

			entries = append(entries, FaxEntry{
				Sender:   from,
				Time:     time.Now(),
				Filename: att.name,
				Status:   status,
			})
		}

		markRead = append(markRead, msg.SeqNum)
	}

	if len(markRead) > 0 {
		ss := new(imap.SeqSet)
		ss.AddNum(markRead...)
		flags := []interface{}{imap.SeenFlag}
		if err := c.Store(ss, imap.FormatFlagsOp(imap.AddFlags, true), flags, nil); err != nil {
			log.Printf("mark read error: %v", err)
		}
	}

	return entries, nil
}

func senderAllowed(addr string, allowed []string) bool {
	addr = strings.ToLower(addr)
	for _, a := range allowed {
		if strings.ToLower(a) == addr {
			return true
		}
	}
	return false
}

func extractAttachments(r io.Reader, cfg Config) ([]attachment, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}

	ct := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return nil, nil
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	var atts []attachment

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return atts, err
		}

		name := part.FileName()
		if name == "" {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		if !extAllowed(ext, cfg.Extensions) {
			log.Printf("skipped extension: %s", ext)
			continue
		}

		var reader io.Reader = part
		enc := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
		if enc == "base64" {
			reader = base64.NewDecoder(base64.StdEncoding, part)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return atts, err
		}

		maxBytes := int64(cfg.MaxMB) * 1024 * 1024
		if int64(len(data)) > maxBytes {
			log.Printf("skipped %s: too large (%d bytes)", name, len(data))
			continue
		}

		atts = append(atts, attachment{name: name, data: data})
	}

	return atts, nil
}

func extAllowed(ext string, allowed []string) bool {
	for _, a := range allowed {
		if strings.ToLower(a) == ext {
			return true
		}
	}
	return false
}
