package handler

import (
	"database/sql"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/njm2360/vrchat-join-manager/server/internal/timeutil"
)

// parseTime は DB 由来の ISO 8601 文字列 (例: "2026-05-28T01:23:45Z") を time.Time にする。
// 不正な値はゼロ値を返す。
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, timeutil.Layout, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := parseTime(s)
	return &t
}

func parseTimeFromNullable(n sql.NullString) *time.Time {
	if !n.Valid {
		return nil
	}
	return parseTimePtr(n.String)
}

// parseDate は "YYYY-MM-DD" 形式の文字列を openapi_types.Date に変換する。
func parseDate(s string) openapi_types.Date {
	t, _ := time.Parse("2006-01-02", s)
	return openapi_types.Date{Time: t}
}

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
