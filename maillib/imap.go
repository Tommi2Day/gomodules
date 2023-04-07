package maillib

import (
	"fmt"
	"io"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	log "github.com/sirupsen/logrus"
)

// ImapType Server Config for Imap
type ImapType struct {
	Inbox        string
	ServerConfig *MailConfigType
	Client       *client.Client
}

// NewImapConfig prepares a new configuration object for sending mails
func NewImapConfig(server string, port int, username string, password string) *ImapType {
	log.Debug("imap:NewConfig entered..")
	var config ImapType
	config.ServerConfig = NewConfig(server, port, username, password)
	config.Inbox = "INBOX"
	return &config
}

// Connect to server
func (config *ImapType) Connect() error {
	log.Debug("imap:Connect entered..")
	var c *client.Client
	var err error
	url := ""
	config.Client = c
	mc := config.ServerConfig
	port := mc.Port
	if mc.SSL {
		if port == 0 {
			port = 993
		}
		url = fmt.Sprintf("%s:%d", mc.Server, port)
		c, err = client.DialTLS(url, mc.tlsConfig)
	} else {
		if port == 0 {
			port = 143
		}
		url = fmt.Sprintf("%s:%d", mc.Server, port)
		c, err = client.Dial(url)
		if err == nil && mc.StartTLS {
			// Start a TLS session
			tlsConfig := mc.tlsConfig
			if err = c.StartTLS(tlsConfig); err == nil {
				log.Debugf("imap:StartTLS on port %d activated", port)
			}
		}
	}

	if err != nil {
		log.Errorf("imap: connect to %s failed:%v", url, err)
		return err
	}
	log.Debugf("imap: Connected to %s", url)

	// Login
	if err := c.Login(config.ServerConfig.Username, config.ServerConfig.Password); err != nil {
		log.Errorf("imap: Login  User %s failed:%v", config.ServerConfig.Username, err)
		return err
	}
	config.Client = c
	log.Debugf("imap:Logged in as %s", config.ServerConfig.Username)
	log.Debug("imap:Connect leaved ..")
	return nil
}

// LogOut Don't forget to logout
func (config *ImapType) LogOut() {
	c := config.Client
	if c != nil {
		err := c.Logout()
		if err != nil {
			log.Warnf("imap: Logout failed:%s", err)
		}
		config.Client = nil
	}
}

// SelectInbox defines the used inbox and return actual status
func (config *ImapType) SelectInbox(inbox string) (all uint32, unseen uint32, flags []string, err error) {
	log.Debug("imap:InboxStatus entered..")
	// Select INBOX
	if inbox == "" {
		inbox = config.Inbox
	}
	c := config.Client
	mbox, err := c.Select(inbox, false)
	if err != nil {
		log.Errorf("imap:Select Inbox '%s' failed:%s", inbox, err)
		return
	}
	all = mbox.Messages
	unseen = mbox.Unseen
	flags = mbox.Flags
	log.Debug("imap:InboxStatus leaved..")
	return
}

// ListMailboxes retrieves existing mailboxes
func (config *ImapType) ListMailboxes() (mailboxes []string, err error) {
	log.Debug("imap:ListMailboxes entered..")
	// List mailboxes
	c := config.Client
	mboxChan := make(chan *imap.MailboxInfo, 10)

	if err = c.List("", "*", mboxChan); err != nil {
		log.Errorf("imap: ListMailboxes failed:%s", err)
		return
	}
	for mbox := range mboxChan {
		mailboxes = append(mailboxes, mbox.Name)
	}
	log.Debug("imap:ListMailboxes leaved..")
	return
}

// ReadMessages reads
func (config *ImapType) ReadMessages(ids []uint32) (msgList []imap.Literal, err error) {
	log.Debug("imap:ReadMessages entered..")
	c := config.Client
	// Get the whole message body
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message, 10)
	if err = c.Fetch(seqset,
		items,
		messages); err != nil {
		log.Errorf("imap: Fetch messages failed:%s", err)
		return
	}

	// read messages from chan
	for msg := range messages {
		r := msg.GetBody(&section)
		if r == nil {
			log.Warnf("imap: Server didn't returned a message body for uid %v", msg.Uid)
		}
		msgList = append(msgList, r)
	}
	log.Debug("imap:ReadMessages leaved..")
	return
}

// translate mail.address types in string
func convertAddresses(a []*mail.Address) (l []string) {
	for _, x := range a {
		l = append(l, x.String())
	}
	return l
}

// ParseMessageBody parses a given imap body for his parts
func (config *ImapType) ParseMessageBody(imapData imap.Literal) (mailContent MailType, err error) {
	log.Debug("imap:ParseMessageBody entered..")
	// Create a new mail reader
	mr, err := mail.CreateReader(imapData)
	if err != nil {
		log.Errorf("imap:ParseMessageBody reading failed:%s", err)
		return
	}

	// Print some info about the message
	header := mr.Header
	if date, err := header.Date(); err == nil {
		log.Debugf("imap: Date:%s", date.String())
		mailContent.Date = date
	}
	if from, err := header.AddressList("From"); err == nil {
		log.Debugf("imap: From:%v", from)
		mailContent.From = from[0].String()
	}
	if to, err := header.AddressList("To"); err == nil {
		log.Debugf("imap:SetTo:%s", to)
		mailContent.To = convertAddresses(to)
	}
	if cc, err := header.AddressList("Cc"); err == nil {
		log.Debugf("imap:SetCC:%s", cc)
		mailContent.CC = convertAddresses(cc)
	}
	if bcc, err := header.AddressList("Bcc"); err == nil {
		log.Debugf("imap:SetBCC:%s", bcc)
		mailContent.Bcc = convertAddresses(bcc)
	}
	if subject, err := header.Subject(); err == nil {
		log.Debugf("Subject:%s", subject)
		mailContent.Subject = subject
	}

	// Process each message's part
	var p *mail.Part
	for {
		p, err = mr.NextPart()
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			log.Errorf("imap:ParseMessageBody NextPart failed:%s", err)
			return
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's TextParts (can be plain-TextParts or HTML)
			b, _ := io.ReadAll(p.Body)
			mailContent.TextParts = append(mailContent.TextParts, string(b))
		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := h.Filename()
			log.Debugf("imap: Got attachment: %s", filename)
			mailContent.Attachments = append(mailContent.Attachments, filename)
		}
	}
	log.Debug("imap:ParseMessageBody leaved..")
	return
}

// GetUnseenMessages count unseen Messages
func (config *ImapType) GetUnseenMessages() (ids []uint32, err error) {
	log.Debug("imap:GetUnseenMessages entered..")
	c := config.Client
	// Set search criteria
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err = c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("imap: %d unseen messages found:", len(ids))
	log.Debug("imap:GetUnseenMessages leaved..")
	return ids, err
}
