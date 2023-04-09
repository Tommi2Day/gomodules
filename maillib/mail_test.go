package maillib

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/require"
	"github.com/wneessen/go-mail"

	"github.com/stretchr/testify/assert"
)

const rootUser = "root"
const infoUser = "info"
const rootPass = "testpass"
const infoPass = "testpass"
const FROM = rootUser + "@" + mailDomain
const TO = infoUser + "@" + mailDomain
const mailDomain = "test.local"
const mailHostname = "mail." + mailDomain

func TestSetMailConfig(t *testing.T) {
	t.Run("Test mailConfig defaults", func(t *testing.T) {
		c := NewConfig("127.0.0.1", 25, "", "")
		actual := c.GetConfig()
		assert.Equal(t, "127.0.0.1", actual.Server, "Server default entry not equal")
		assert.Equal(t, 25, actual.Port, "Port entry should be default")
		assert.Empty(t, actual.Username, "Username entry should not set")
	})
	t.Run("Test NewConfig", func(t *testing.T) {
		s := NewSendMailConfig("test.example.com", sslPort, "testuser", "Testpass")
		s.ServerConfig.EnableSSL(true)
		actual := s.GetConfig().ServerConfig
		assert.Equal(t, "test.example.com", actual.Server, "Server entry not equal")
		assert.Equal(t, sslPort, actual.Port, "Port entry not equal")
		assert.Equal(t, "testuser", actual.Username, "Username entry not equal")
		assert.Equal(t, "Testpass", actual.Password, "Password entry not equal")
		assert.True(t, actual.SSL, "SSL entry not set")
		assert.True(t, actual.SSLinsecure, "Insecure SSL/TLS entry not set")
		assert.False(t, actual.StartTLS, "StartTLS should not set")
	})
	t.Run("Check SetMaxsize", func(t *testing.T) {
		s := NewSendMailConfig("test.example.com", sslPort, "testuser", "Testpass")
		s.SetMaxSize(2048)
		actual := s.GetConfig()
		assert.Equal(t, int64(2048), actual.maxSize)
	})
}

func TestSendMailError(t *testing.T) {
	s := NewSendMailConfig("test.example.com", 25, "testuser", "Testpass")
	t.Run("Send Mail with wrong email", func(t *testing.T) {
		l := NewMail("dummy@local", "root")
		err := s.SendMail(l, "TestMail", "My Message")
		assert.Errorf(t, err, "Error: %v", err)
	})
	t.Run("Send Mail with Send Error", func(t *testing.T) {
		l := NewMail("dummy@local", "root@example.com")
		err := s.SendMail(l, "TestMail2", "Email Address Test")
		assert.Errorf(t, err, "Error: %v", err)
	})
}

func TestMail(t *testing.T) {
	if os.Getenv("SKIP_MAIL") != "" {
		t.Skip("Skipping Mail testing in CI environment")
	}

	var err error
	mailContainer, err = prepareMailContainer()
	require.NoErrorf(t, err, "Mailserver not available: %s", err)
	require.NotNil(t, mailContainer, "Prepare failed")
	defer destroyMailContainer(mailContainer)

	t.Run("Send Mail anonym", func(t *testing.T) {
		s := NewSendMailConfig(mailServer, smtpPort, "", "")
		l := NewMail(FROM, TO)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		s.SetContentType(mail.TypeTextHTML)
		err = s.SendMail(l, "Testmail1", fmt.Sprintf("<html><body>Test at %s</body></html>", timeStr))
		assert.NoErrorf(t, err, "Sendmail anonym returned error %v", err)
	})
	t.Run("Send Mail TLS 25", func(t *testing.T) {
		s := NewSendMailConfig(mailServer, smtpPort, FROM, rootPass)
		s.ServerConfig.EnableTLS(true)
		l := NewMail(FROM, TO)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		l.SetBcc("bcc@mail.local")
		assert.Equal(t, 1, len(l.Bcc), "BCC should be set")
		err = s.SendMail(l, "Testmail2", fmt.Sprintf("Test at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail with login returned error %v", err)
	})

	t.Run("Send Mail SSL 465", func(t *testing.T) {
		s := NewSendMailConfig(mailServer, sslPort, FROM, rootPass)
		s.ServerConfig.EnableSSL(true)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		l := NewMail(FROM, TO)
		l.SetCc("cc@mail.local")
		l.SetAttach([]string{
			test.TestDir + "/mail/ssl/ca.crt",
			test.TestDir + "/mail/ssl/mail.test.local.crt",
		})
		err = s.SendMail(l, "Testmail3", fmt.Sprintf("Test with ssl and attachment at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail SSL returned error %v", err)
	})
	t.Run("Send Mail TLS 587", func(t *testing.T) {
		s := NewSendMailConfig(mailServer, tlsPort, FROM, rootPass)
		s.ServerConfig.EnableTLS(true)
		s.ServerConfig.SetTimeout(20)
		l := NewMail(FROM, TO)
		h := time.Now()
		timeStr := h.Format("15:04:05")
		err = s.SendMail(l, "Testmail4", fmt.Sprintf("Test with tls at %s", timeStr))
		assert.NoErrorf(t, err, "Sendmail TLS returned error %v", err)
	})
	t.Log("wait for Mails to proceed to Inbox")
	time.Sleep(10 * time.Second)
	t.Run("Imap Connect 143", func(t *testing.T) {
		i := NewImapConfig(mailServer, imapPort, TO, infoPass)
		i.ServerConfig.EnableTLS(false)
		i.ServerConfig.SetTimeout(20)
		err = i.Connect()
		assert.NoErrorf(t, err, "Imap Plain Connect returned error %v", err)
		i.LogOut()
	})
	t.Run("Imap Connect wrong password", func(t *testing.T) {
		i := NewImapConfig(mailServer, imapPort, TO, "WrongPass")
		i.ServerConfig.EnableTLS(false)
		i.ServerConfig.SetTimeout(20)
		err = i.Connect()
		t.Logf("expected Error:%v", err)
		assert.Errorf(t, err, "Imap Plain Connect returned not the expected error", err)
		i.LogOut()
	})
	t.Run("Imap Connect TLS 143", func(t *testing.T) {
		i := NewImapConfig(mailServer, imapPort, FROM, rootPass)
		i.ServerConfig.EnableTLS(true)
		i.ServerConfig.SetTimeout(20)
		err = i.Connect()
		assert.NoErrorf(t, err, "Imap TLS Connect returned error %v", err)
		i.LogOut()
	})
	i := NewImapConfig(mailServer, imapsPort, TO, infoPass)
	t.Run("Imap Connect SSL 993", func(t *testing.T) {
		i.ServerConfig.EnableSSL(true)
		i.ServerConfig.SetTimeout(20)
		err = i.Connect()
		assert.NoErrorf(t, err, "Imap SSL Connect returned error %v", err)
	})

	require.NotNil(t, i.Client, "Imap not connected")
	defer i.LogOut()
	t.Run("Imap List mailboxes", func(t *testing.T) {
		var mboxes []string
		mboxes, err = i.ListMailboxes()
		t.Logf("Inboxes:%v", mboxes)
		assert.NoErrorf(t, err, "List Mailboxes failed:%s", err)
		actual := len(mboxes)
		assert.Greaterf(t, actual, 0, "List Mailboxes should return at least one Mailbox")
		require.Containsf(t, mboxes, "INBOX", "Mailbox INBOX should exist")
	})

	t.Run("Imap Get Messages", func(t *testing.T) {
		var ids []uint32
		var allMsg []ImapMsg
		t.Run("Imap Inbox Status", func(t *testing.T) {
			all, seen, flags, err := i.MBoxStatus("INBOX")
			t.Logf("INBOX:all %d, seen:%d,Flags:%v", all, seen, flags)
			assert.NoErrorf(t, err, "Inbox Status failed:%s", err)
		})
		t.Run("Imap Search", func(t *testing.T) {
			criteria := imap.NewSearchCriteria()
			criteria.Body = []string{"attachment"}
			ids, err = i.SearchMessages(criteria)
			assert.NoErrorf(t, err, "Search Error:%s", err)
			assert.Equal(t, len(ids), 1, "Search Result not expected")
			t.Logf("Search ids:%v", ids)
		})
		t.Run("Imap Get Unseen", func(t *testing.T) {
			ids, err = i.GetUnseenMessageIds()
			assert.NoErrorf(t, err, "GetUnseenMessages Error:%s", err)
			t.Logf("unseen ids:%v", ids)
		})
		t.Run("Imap Read Message", func(t *testing.T) {
			allMsg, err = i.ReadMessages(ids)
			assert.NoErrorf(t, err, "ReadMessages Error:%s", err)
		})
		c := len(allMsg)
		require.Equal(t, 4, c, "Message Count not fit")
		if c > 3 {
			t.Run("Imap Parse Message", func(t *testing.T) {
				i.DownloadDir = test.TestData
				// parse and download
				msg, err := i.ParseMessage(allMsg[2], true)
				assert.NoErrorf(t, err, "ParseMessage Error:%s", err)
				require.NotNil(t, msg, "Content should not nil")
				from := msg.From
				to := msg.To
				cc := msg.CC
				subject := msg.Subject
				attach := msg.Attachments
				text := msg.TextParts
				date := msg.Date
				assert.NotEmpty(t, from, "From should be set")
				assert.Equal(t, 1, len(to), "To should be set")
				assert.Equal(t, 1, len(cc), "CC should be set")
				assert.Equal(t, subject, "Testmail3", "Subject not as expected")
				assert.Equal(t, 2, len(attach), "Attach Count not as expected")
				assert.Equal(t, 1, len(text), "Text Part Count not as expected")
				assert.NotEmpty(t, date, "Date should be set")
				fn := path.Join(test.TestData, attach[0])
				assert.FileExistsf(t, fn, "expected attachment file '%s' not found", fn)
			})
		}
		t.Run("Imap Delete", func(t *testing.T) {
			time.Sleep(3 * time.Second)
			var mbox *imap.MailboxStatus
			mbox, err = i.SelectMailbox("")
			require.NoErrorf(t, err, "Select Mailbox failed")
			require.NotNil(t, mbox, "mbox not set")
			mcount := int(mbox.Messages)
			// select all mail send
			criteria := imap.NewSearchCriteria()
			criteria.Text = []string{"Testmail"}
			ids, err = i.SearchMessages(criteria)
			assert.NoErrorf(t, err, "Search Error:%s", err)
			l := len(ids)
			t.Logf("Found ids:%v", ids)
			assert.Equal(t, mcount, l, "Messages count not equal search count")
			// delete mails
			err = i.PurgeMessages(ids)
			assert.NoErrorf(t, err, "Purge Error:%s", err)
			// sleep to proceed
			time.Sleep(2 * time.Second)
			// check again
			ids, err = i.SearchMessages(criteria)
			t.Logf("ids after delete:%v", ids)
			l = len(ids)
			assert.Equal(t, 0, l, "Mailbox Search should not return deleted messages")
		})
	})
}
