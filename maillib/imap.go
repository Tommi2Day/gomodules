package maillib

import (
	"fmt"
	"io"
	"os"
	"path"

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
	DownloadDir  string
}

// ImapMsg hold a received message
type ImapMsg struct {
	Content imap.Literal
	UID     uint32
}

// NewImapConfig prepares a new configuration object for sending mails
func NewImapConfig(server string, port int, username string, password string) *ImapType {
	log.Debug("imap:NewConfig entered..")
	var config ImapType
	config.ServerConfig = NewConfig(server, port, username, password)
	config.Inbox = "INBOX"
	config.DownloadDir = "."
	return &config
}

// SetDownloadDir target dir to save Attachments
func (it *ImapType) SetDownloadDir(dir string) {
	it.DownloadDir = dir
}

// Connect to server
func (it *ImapType) Connect() error {
	log.Debug("imap:Connect entered..")
	var c *client.Client
	var err error
	url := ""
	it.Client = c
	mc := it.ServerConfig
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
	if err := c.Login(it.ServerConfig.Username, it.ServerConfig.Password); err != nil {
		log.Errorf("imap: Login  User %s failed:%v", it.ServerConfig.Username, err)
		return err
	}
	it.Client = c
	log.Debugf("imap:Logged in as %s", it.ServerConfig.Username)
	log.Debug("imap:Connect leaved ..")
	return nil
}

// LogOut Don't forget to logout
func (it *ImapType) LogOut() {
	c := it.Client
	if c != nil {
		err := c.Logout()
		if err != nil {
			log.Warnf("imap: Logout failed:%s", err)
		}
		it.Client = nil
	}
}

// MBoxStatus defines the used inbox and return actual status
func (it *ImapType) MBoxStatus(inbox string) (all uint32, unseen uint32, flags []string, err error) {
	log.Debug("imap: MBoxStatus entered..")
	// Select INBOX
	mbox, err := it.SelectMailbox(inbox)
	if err != nil {
		err = fmt.Errorf("MBoxStatus: Select failed:%s", err)
		return
	}
	if mbox == nil {
		err = fmt.Errorf("MBoxStatus: mbox not available")
		return
	}
	all = mbox.Messages
	unseen = mbox.Unseen
	flags = mbox.Flags
	log.Debug("imap: MBoxStatus leaved..")
	return
}

// ListMailboxes retrieves existing mailboxes
func (it *ImapType) ListMailboxes() (mailboxes []string, err error) {
	log.Debug("imap:ListMailboxes entered..")
	// List mailboxes
	c := it.Client
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

// PurgeMessages deletes given ids finally from mailbox
func (it *ImapType) PurgeMessages(ids []uint32) (err error) {
	log.Debug("imap: Purge entered..")
	c := it.Client
	// checks
	if c == nil {
		err = fmt.Errorf("imap Purge: no Conn available")
		return
	}
	if len(ids) == 0 {
		err = fmt.Errorf("imap Purge: no ids given")
		return
	}
	_, err = it.SelectMailbox("")
	if err != nil {
		return
	}

	// build seqset with given ids
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	// First mark the message as deleted
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err = c.Store(seqset, item, flags, nil); err != nil {
		err = fmt.Errorf("imap: delete messages failed:%s", err)
		log.Error(err)
		return
	}

	// Then delete it
	if err = c.Expunge(nil); err != nil {
		err = fmt.Errorf("imap: Expunge messages failed:%s", err)
		log.Error(err)
		return
	}
	log.Debugf("imap Purge: %d messages deleted", len(ids))
	return
}

// ReadMessages reads
func (it *ImapType) ReadMessages(ids []uint32) (msgList []ImapMsg, err error) {
	log.Debug("imap:ReadMessages entered..")
	c := it.Client
	// checks
	if c == nil {
		err = fmt.Errorf("imap Search: no Conn available")
		return
	}
	_, err = it.SelectMailbox("")
	if err != nil {
		return
	}

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
		newMsg := ImapMsg{}
		newMsg.UID = msg.Uid
		newMsg.Content = r
		msgList = append(msgList, newMsg)
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

func parseHeader(header mail.Header) (mailContent MailType) {
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
	return
}

// ParseMessage parses a given imap body for his parts
func (it *ImapType) ParseMessage(imapData ImapMsg, saveAttachments bool) (mailContent MailType, err error) {
	log.Debug("imap:ParseMessage entered..")
	// Create a new mail reader
	mr, err := mail.CreateReader(imapData.Content)
	if err != nil {
		log.Errorf("imap:ParseMessage reading failed:%s", err)
		return
	}
	dir := it.DownloadDir

	// get info about the message
	header := mr.Header
	mailContent = parseHeader(header)
	mailContent.ID = imapData.UID

	// Process each message's part
	var p *mail.Part
	for {
		p, err = mr.NextPart()
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			log.Errorf("imap:ParseMessage NextPart failed:%s", err)
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
			if saveAttachments {
				fn := path.Join(dir, filename)
				//nolint gosec
				dst, err := os.Create(fn)
				if err != nil {
					log.Errorf("imap: Create Attachement file '%s' failed:%s", fn, err)
					break
				}
				size, err := io.Copy(dst, p.Body)
				if err != nil {
					log.Errorf("imap: write Attachment file '%s' failed:%s", fn, err)
					break
				}
				log.Debugf("imap: Attachment file '%s' (%d bytes) written", fn, size)
				_ = dst.Close()
			}
		}
	}
	log.Debug("imap:ParseMessage leaved..")
	return
}

// GetUnseenMessageIDs returns IDs of unseen Messages
func (it *ImapType) GetUnseenMessageIDs() (ids []uint32, err error) {
	log.Debug("imap:GetUnseenMessages entered..")
	// Set search criteria
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	// do search
	return it.SearchMessages(criteria)
}

// SearchMessages find messages with given criteria
func (it *ImapType) SearchMessages(criteria *imap.SearchCriteria) (ids []uint32, err error) {
	log.Debug("imap: Search entered..")
	c := it.Client
	// checks
	if c == nil {
		err = fmt.Errorf("imap Search: no Conn available")
		return
	}
	if criteria == nil {
		err = fmt.Errorf("imap Search: no criteria given")
		return
	}
	_, err = it.SelectMailbox("")
	if err != nil {
		return
	}

	// do search
	ids, err = c.Search(criteria)
	if err != nil {
		log.Errorf("imap: Search failed:%s", err)
		return
	}
	log.Debugf("imap: %d messages found:", len(ids))
	log.Debug("imap: Search leaved..")
	return
}

// SelectMailbox activates the named or stored mailxbox name
func (it *ImapType) SelectMailbox(mboxName string) (status *imap.MailboxStatus, err error) {
	c := it.Client
	if c == nil {
		err = fmt.Errorf("imap: SelectMailbox: no Conn available")
		return
	}
	if mboxName == "" {
		mboxName = it.Inbox
	}
	log.Debugf("imap: Select Mailbox '%s' ..", mboxName)

	mbox := c.Mailbox()
	curName := ""
	if mbox != nil {
		curName = mbox.Name
	}
	if mbox == nil || curName != mboxName {
		status, err = c.Select(mboxName, false)
		if err != nil {
			return nil, fmt.Errorf("imap:failed to select mailbox: %v", err)
		}
		log.Debugf("imap: selected Mailbox '%s' changed to '%s' ", curName, mboxName)
		// store current mailbox name
		it.Inbox = mboxName
		return status, nil
	}
	return mbox, nil
}
