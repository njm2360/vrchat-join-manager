package handler

import (
	"database/sql"
	"time"

	"github.com/njm2360/vrchat-join-manager/server/internal/timeutil"
)

func strPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	s := n.String
	return &s
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func timePtrToStrPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := timeutil.FormatUTC(*t)
	return &s
}

func timeToStr(t time.Time) string {
	return timeutil.FormatUTC(t)
}

func enumStrOr[T ~string](p *T, fallback string) string {
	if p == nil {
		return fallback
	}
	if s := string(*p); s != "" {
		return s
	}
	return fallback
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
