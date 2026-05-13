package timeutil

import "time"

// Format used by the Python implementation:
// dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
const Layout = "2006-01-02T15:04:05Z"

// FormatUTC formats t as UTC in the legacy Python timestamp layout.
func FormatUTC(t time.Time) string {
	return t.UTC().Format(Layout)
}

// FormatUTCPtr returns nil if t is nil; otherwise the formatted string pointer.
func FormatUTCPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatUTC(*t)
	return &s
}
