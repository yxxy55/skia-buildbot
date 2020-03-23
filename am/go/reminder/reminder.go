// Package reminder emails periodic reminders to active alert owners.
package reminder

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"go.skia.org/infra/am/go/incident"
	"go.skia.org/infra/am/go/silence"
	"go.skia.org/infra/go/email"
	"go.skia.org/infra/go/httputils"
	"go.skia.org/infra/go/rotations"
	"go.skia.org/infra/go/sklog"
)

const (
	// Currently setup to email reminders daily at 4am UTC time.
	reminderHourUTC  = 4
	reminderDuration = 24 * time.Hour

	trooperURL = "https://tree-status.skia.org/current-trooper"

	emailTemplate = `
Hi {{.Owner}},
<br/><br/>


You either own or were assigned these alerts on am.skia.org:
<ul>
  {{range $a := .Alerts}}
    <li>{{$a}}</li>
  {{end}}
</ul>

This is a friendly reminder to add a silence or to resolve them whenever possible.
<br/><br/>

Thanks!
`
)

var (
	emailTemplateParsed = template.Must(template.New("reminder_email").Parse(emailTemplate))
)

type emailTicker struct {
	t         *time.Timer
	iStore    *incident.Store
	sStore    *silence.Store
	emailAuth *email.GMail
}

// getDailyNextTickDuration returns the duration after which the reminder should be sent.
// If startTimeUTC is after reminderHourUTC then the next day's reminderHourUTC is used.
// If startTimeUTC is before reminderHourUTC then the current day's reminderHourUTC is used.
func getDailyNextTickDuration(startTimeUTC time.Time, reminderHourUTC int) time.Duration {
	nextTick := time.Date(startTimeUTC.Year(), startTimeUTC.Month(), startTimeUTC.Day(), reminderHourUTC, 0, 0, 0, time.UTC)
	if nextTick.Before(startTimeUTC) {
		nextTick = nextTick.Add(reminderDuration)
	}
	sklog.Infof("[reminder] Next tick is %s", nextTick)
	return nextTick.Sub(startTimeUTC)
}

// getOwnersToAlerts returns a map of owners to alerts.
// An owner of an alert is determined by looking at "assigned_to" of an active alert. If "assigned_to"
// does not exist then it looks at the "owner" of the active alert.
func getOwnersToAlerts(ins []incident.Incident, silences []silence.Silence) map[string][]incident.Incident {
	ownersToAlerts := map[string][]incident.Incident{}
	for _, i := range ins {
		if !i.IsSilenced(silences) && (i.Params["owner"] != "" || i.Params["assigned_to"] != "") {
			owner := i.Params["assigned_to"]
			if owner == "" {
				owner = i.Params["owner"]
			}
			ownersToAlerts[owner] = append(ownersToAlerts[owner], i)
		}
	}
	return ownersToAlerts
}

func (et emailTicker) updateEmailTicker() {
	et.t.Reset(getDailyNextTickDuration(time.Now().UTC(), reminderHourUTC))
}

// remindAlertOwners sends a reminder email with a list of firing alerts to
// the owners/assignees of the alerts.
func (et emailTicker) remindAlertOwners() error {
	ins, err := et.iStore.GetAll()
	if err != nil {
		return fmt.Errorf("Failed to load incidents: %s", err)
	}
	silences, err := et.sStore.GetAll()
	if err != nil {
		return fmt.Errorf("Failed to load silences: %s", err)
	}
	if silences == nil {
		silences = []silence.Silence{}
	}

	// Find the current trooper.
	troopers, err := rotations.FromURL(httputils.NewTimeoutClient(), trooperURL)
	if err != nil {
		return fmt.Errorf("Could not get current trooper: %s", err)
	}
	if len(troopers) != 1 {
		return fmt.Errorf("Expected 1 entry from %s. Instead got %s", trooperURL, troopers)
	}
	trooper := troopers[0]

	// Send reminder emails to alert owners (but not to the trooper).
	ownersToAlerts := getOwnersToAlerts(ins, silences)
	for o, alerts := range ownersToAlerts {
		if o == trooper {
			sklog.Infof("Not going to email %s because they are the current trooper", o)
			continue
		}
		sklog.Infof("Going to email %s for these alerts:\n", o)
		alertDescriptions := []string{}
		for _, a := range alerts {
			desc := fmt.Sprintf("%s - %s", a.Params["alertname"], a.Params["abbr"])
			alertDescriptions = append(alertDescriptions, desc)
			sklog.Infof("\t%s\n", desc)
		}
		emailBytes := new(bytes.Buffer)
		if err := emailTemplateParsed.Execute(emailBytes, struct {
			Owner  string
			Alerts []string
		}{
			Owner:  o,
			Alerts: alertDescriptions,
		}); err != nil {
			return fmt.Errorf("Failed to execute email template: %s", err)
		}

		emailSubject := "You have active alerts on am.skia.org"
		viewActionMarkup, err := email.GetViewActionMarkup("am.skia.org/?tab=0", "View Alerts", "View alerts owned by you")
		if err != nil {
			return fmt.Errorf("Failed to get view action markup: %s", err)
		}
		if err := et.emailAuth.SendWithMarkup("Alert Manager", []string{o, "rmistry@google.com" /*temporary*/}, emailSubject, emailBytes.String(), viewActionMarkup); err != nil {
			return fmt.Errorf("Could not send email: %s", err)
		}
	}

	return nil
}

func StartReminderTicker(iStore *incident.Store, sStore *silence.Store, emailAuth *email.GMail) {
	et := emailTicker{
		t:         time.NewTimer(getDailyNextTickDuration(time.Now().UTC(), reminderHourUTC)),
		iStore:    iStore,
		sStore:    sStore,
		emailAuth: emailAuth,
	}
	go func() {
		for {
			<-et.t.C
			sklog.Infof("[reminder] Going to send reminders")
			if err := et.remindAlertOwners(); err != nil {
				sklog.Errorf("Error emailing alert owners: %s", err)
			}
			et.updateEmailTicker()
		}
	}()
}
