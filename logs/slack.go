package logs

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/GNURub/beego/encoder/json"
)

// SLACKWriter implements beego LoggerInterface and is used to send jiaoliao webhook
type SLACKWriter struct {
	WebhookURL string `json:"webhookurl"`
	Level      int    `json:"level"`
}

// newSLACKWriter create jiaoliao writer.
func newSLACKWriter() Logger {
	return &SLACKWriter{Level: LevelTrace}
}

// Init SLACKWriter with json config string
func (s *SLACKWriter) Init(jsonConfig string) error {
	return json.Decode([]byte(jsonConfig), s)
}

// WriteMsg write message in smtp writer.
// it will send an email with subject and only this message.
func (s *SLACKWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level > s.Level {
		return nil
	}

	text := fmt.Sprintf("{\"text\": \"%s %s\"}", when.Format("2006-01-02 15:04:05"), msg)

	form := url.Values{}
	form.Add("payload", text)

	resp, err := http.PostForm(s.WebhookURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("post webhook failed %s %d", resp.Status, resp.StatusCode)
	}
	return nil
}

// Flush implementing method. empty.
func (s *SLACKWriter) Flush() {
}

// Destroy implementing method. empty.
func (s *SLACKWriter) Destroy() {
}

func init() {
	Register(AdapterSlack, newSLACKWriter)
}
