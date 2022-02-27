// Package alert contains functions for flashing alerts to a session.
package alert

import "github.com/gorilla/sessions"

// Level is the level of the alert.
type Level uint

//go:generate stringer -type Level -linecomment
const (
	Success Level = iota + 1 // success
	Warn                     // warn
	Danger                   // danger
)

type Alert struct {
	Level   Level  // The level of the alert.
	Close   bool   // Whether a close button should be rendered.
	HTML    bool   // Whether HTML code should be escaped or not.
	Message string // The message to render in the alert.
}

const SessionKey = "alert"

// First returns the first alert flashed to the given session.
func First(sess *sessions.Session) Alert {
	val := sess.Flashes(SessionKey)

	if val == nil {
		return Alert{}
	}
	return val[0].(Alert)
}

// Flash will flash an alert with the given level and message to the given
// session.
func Flash(sess *sessions.Session, lvl Level, msg string) {
	sess.AddFlash(Alert{
		Level:   lvl,
		Close:   true,
		Message: msg,
	}, SessionKey)
}

// FlashHTML will flash an alert with the given level and message to the given
// session. The flashed alert will render any HTML in the given message.
func FlashHTML(sess *sessions.Session, lvl Level, msg string) {
	sess.AddFlash(Alert{
		Level:   lvl,
		Close:   true,
		HTML:    true,
		Message: msg,
	}, SessionKey)
}
