package notifier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/pubsub"
	"go.skia.org/infra/go/chatbot"
	"go.skia.org/infra/go/common"
	"go.skia.org/infra/go/email"
	"go.skia.org/infra/go/issues"
	"go.skia.org/infra/go/sklog"
	"go.skia.org/infra/go/util"
)

const (
	EMAIL_FROM_ADDRESS = "noreply@skia.org"
)

// Notifier is an interface used for sending notifications from an AutoRoller.
type Notifier interface {
	// Send the given message to the given thread. This should be safe to
	// run in a goroutine.
	Send(ctx context.Context, thread string, msg *Message) error
}

// Configuration for a Notifier.
type Config struct {
	// Required fields.

	// Configuration for filtering out messages. Exactly one of these should
	// be specified.
	Filter           string   `json:"filter,omitempty"`
	MsgTypeWhitelist []string `json:"msgTypeWhitelist,omitempty"`

	// Exactly one of these should be specified.
	Email    *EmailNotifierConfig    `json:"email,omitempty"`
	Chat     *ChatNotifierConfig     `json:"chat,omitempty"`
	Monorail *MonorailNotifierConfig `json:"monorail,omitempty"`
	PubSub   *PubSubNotifierConfig   `json:"pubsub,omitempty"`

	// Optional fields.

	// If present, all messages inherit this subject line.
	Subject string `json:"subject,omitempty"`
}

// Validate the Config.
func (c *Config) Validate() error {
	if c.Filter == "" && c.MsgTypeWhitelist == nil {
		return errors.New("Either Filter or MsgTypeWhitelist is required.")
	}
	if c.Filter != "" && c.MsgTypeWhitelist != nil {
		return errors.New("Only one of Filter or MsgTypeWhitelist may be provided.")
	}
	if c.Filter != "" {
		if _, err := ParseFilter(c.Filter); err != nil {
			return err
		}
	}
	n := []util.Validator{}
	if c.Email != nil {
		n = append(n, c.Email)
	}
	if c.Chat != nil {
		n = append(n, c.Chat)
	}
	if c.PubSub != nil {
		n = append(n, c.PubSub)
	}
	if c.Monorail != nil {
		n = append(n, c.Monorail)
	}
	if len(n) != 1 {
		return fmt.Errorf("Exactly one notification config must be supplied, but got %d", len(n))
	}
	return n[0].Validate()
}

// Create a Notifier from the Config.
func (c *Config) Create(ctx context.Context, client *http.Client, emailer *email.GMail, chatBotConfigReader chatbot.ConfigReader) (Notifier, Filter, []string, string, error) {
	if err := c.Validate(); err != nil {
		return nil, FILTER_SILENT, nil, "", err
	}
	filter, err := ParseFilter(c.Filter)
	if err != nil {
		return nil, FILTER_SILENT, nil, "", err
	}
	var n Notifier
	if c.Email != nil {
		n, err = EmailNotifier(c.Email.Emails, emailer, "")
	} else if c.Chat != nil {
		n, err = ChatNotifier(c.Chat.RoomID, chatBotConfigReader)
	} else if c.PubSub != nil {
		n, err = PubSubNotifier(ctx, c.PubSub.Topic)
	} else if c.Monorail != nil {
		n, err = MonorailNotifier(client, c.Monorail.Project, c.Monorail.Owner, c.Monorail.CC, c.Monorail.Labels)
	} else {
		return nil, FILTER_SILENT, nil, "", fmt.Errorf("No config specified!")
	}
	if err != nil {
		return nil, FILTER_SILENT, nil, "", err
	}
	return n, filter, c.MsgTypeWhitelist, c.Subject, nil
}

// Create a copy of this Config.
func (c *Config) Copy() *Config {
	configCopy := &Config{
		Filter:           c.Filter,
		MsgTypeWhitelist: util.CopyStringSlice(c.MsgTypeWhitelist),
		Subject:          c.Subject,
	}
	if c.Email != nil {
		configCopy.Email = &EmailNotifierConfig{
			Emails: util.CopyStringSlice(c.Email.Emails),
		}
	}
	if c.Chat != nil {
		configCopy.Chat = &ChatNotifierConfig{
			RoomID: c.Chat.RoomID,
		}
	}
	if c.PubSub != nil {
		configCopy.PubSub = &PubSubNotifierConfig{
			Topic: c.PubSub.Topic,
		}
	}
	if c.Monorail != nil {
		configCopy.Monorail = &MonorailNotifierConfig{
			Project: c.Monorail.Project,
			Owner:   c.Monorail.Owner,
			CC:      util.CopyStringSlice(c.Monorail.CC),
			Labels:  util.CopyStringSlice(c.Monorail.Labels),
		}
	}
	return configCopy
}

// Configuration for EmailNotifier.
type EmailNotifierConfig struct {
	// List of email addresses to notify. Required.
	Emails []string `json:"emails"`
}

// Validate the EmailNotifierConfig.
func (c *EmailNotifierConfig) Validate() error {
	if c.Emails == nil || len(c.Emails) == 0 {
		return fmt.Errorf("Emails is required.")
	}
	return nil
}

// emailNotifier is a Notifier implementation which sends email to interested
// parties.
type emailNotifier struct {
	from   string
	gmail  *email.GMail
	markup string
	to     []string
}

// See documentation for Notifier interface.
func (n *emailNotifier) Send(_ context.Context, subject string, msg *Message) error {
	if n.gmail == nil {
		sklog.Warning("No gmail API client; cannot send email!")
		return nil
	}
	sklog.Infof("Sending email to %s: %s", strings.Join(n.to, ","), subject)
	return n.gmail.SendWithMarkup(n.from, n.to, subject, msg.Body, n.markup)
}

// EmailNotifier returns a Notifier which sends email to interested parties.
// Sends the same ViewAction markup with each message.
func EmailNotifier(emails []string, emailer *email.GMail, markup string) (Notifier, error) {
	return &emailNotifier{
		from:   EMAIL_FROM_ADDRESS,
		gmail:  emailer,
		markup: markup,
		to:     emails,
	}, nil
}

// Configuration for ChatNotifier.
type ChatNotifierConfig struct {
	RoomID string `json:"room"`
}

// Validate the ChatNotifierConfig.
func (c *ChatNotifierConfig) Validate() error {
	if c.RoomID == "" {
		return fmt.Errorf("RoomID is required.")
	}
	return nil
}

// chatNotifier is a Notifier implementation which sends chat messages.
type chatNotifier struct {
	configReader chatbot.ConfigReader
	roomId       string
}

// See documentation for Notifier interface.
func (n *chatNotifier) Send(_ context.Context, thread string, msg *Message) error {
	return chatbot.SendUsingConfig(msg.Body, n.roomId, thread, n.configReader)
}

// ChatNotifier returns a Notifier which sends email to interested parties.
func ChatNotifier(roomId string, configReader chatbot.ConfigReader) (Notifier, error) {
	return &chatNotifier{
		configReader: configReader,
		roomId:       roomId,
	}, nil
}

// Configuration for a PubSubNotifier.
type PubSubNotifierConfig struct {
	Topic string `json:"topic"`
}

// Validate the PubSubNotifierConfig.
func (c *PubSubNotifierConfig) Validate() error {
	if c.Topic == "" {
		return errors.New("Topic is required.")
	}
	return nil
}

// pubSubNotifier is a Notifier implementation which sends pub/sub messages.
type pubSubNotifier struct {
	topic *pubsub.Topic
}

// See documentation for Notifier interface.
func (n *pubSubNotifier) Send(ctx context.Context, subject string, msg *Message) error {
	res := n.topic.Publish(ctx, &pubsub.Message{
		Attributes: map[string]string{
			"severity": msg.Severity.String(),
			"subject":  subject,
		},
		Data: []byte(msg.Body),
	})
	_, err := res.Get(ctx)
	return err
}

// PubSubNotifier returns a Notifier which sends messages via PubSub.
func PubSubNotifier(ctx context.Context, topic string) (Notifier, error) {
	client, err := pubsub.NewClient(ctx, common.PROJECT_ID)
	if err != nil {
		return nil, err
	}

	// Create the topic if it doesn't exist.
	t := client.Topic(topic)
	if exists, err := t.Exists(ctx); err != nil {
		return nil, err
	} else if !exists {
		t, err = client.CreateTopic(ctx, topic)
		if err != nil {
			return nil, err
		}
	}
	return &pubSubNotifier{
		topic: t,
	}, nil
}

// Configuration for a MonorailNotifier.
type MonorailNotifierConfig struct {
	// Project name under which to file bugs. Required.
	Project string `json:"project"`

	// Owner of bugs filed in Monorail. Required.
	Owner string `json:"owner"`

	// List of people to CC on bugs filed in Monorail. Optional.
	CC []string `json:"cc,omitempty"`

	// List of labels to apply to bugs filed in Monorail. Optional.
	Labels []string `json:"labels,omitempty"`
}

// Validate the MonorailNotifierConfig.
func (c *MonorailNotifierConfig) Validate() error {
	if c.Owner == "" {
		return errors.New("Owner is required.")
	}
	if c.Project == "" {
		return errors.New("Project is required.")
	}
	return nil
}

// monorailNotifier is a Notifier implementation which files Monorail issues.
type monorailNotifier struct {
	tk     issues.IssueTracker
	cc     []issues.MonorailPerson
	labels []string
	owner  issues.MonorailPerson
}

// See documentation for Notifier interface.
func (n *monorailNotifier) Send(ctx context.Context, subject string, msg *Message) error {
	req := issues.IssueRequest{
		CC:          n.cc,
		Description: msg.Body,
		Labels:      n.labels,
		Owner:       n.owner,
		Status:      "New",
		Summary:     subject,
	}
	return n.tk.AddIssue(req)
}

// MonorailNotifier returns a Notifier which files bugs in Monorail.
func MonorailNotifier(c *http.Client, project, owner string, cc []string, labels []string) (Notifier, error) {
	var personCC []issues.MonorailPerson
	if cc != nil {
		personCC := make([]issues.MonorailPerson, 0, len(cc))
		for _, name := range cc {
			personCC = append(personCC, issues.MonorailPerson{
				Name: name,
			})
		}
	}
	return &monorailNotifier{
		tk:     issues.NewMonorailIssueTracker(c, project),
		cc:     personCC,
		labels: labels,
		owner: issues.MonorailPerson{
			Name: owner,
		},
	}, nil

}
