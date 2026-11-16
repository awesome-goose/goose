package sql

import (
	"context"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// postgresDialector opens Postgres so that timestamptz values come back in Config.TimeZone.
//
// The session's TimeZone decides how Postgres prints a timestamptz, but pgx scans one into
// time.Local whatever the session says, so a server whose machine runs at +01:00 answered
// "2026-11-10T19:28:12+01:00" for columns created as timestamptz, and "...Z" for plain
// timestamp ones (BUGS #23). With a TimeZone set, timestamptz is scanned in that zone, the same
// zone NowFunc already writes in; the instant never changes, only the offset it is shown with.
// An empty TimeZone keeps the old behaviour (machine local time).
func postgresDialector(dsn, timeZone string) gorm.Dialector {
	loc := scanLocation(timeZone)
	if loc == nil {
		return postgres.Open(dsn)
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return postgres.Open(dsn) // gorm reports the bad DSN on Open, as before
	}
	// gorm's own dialector sends the DSN's TimeZone as the session timezone; keep doing so.
	if m := timeZoneParam.FindStringSubmatch(dsn); len(m) > 2 {
		config.RuntimeParams["timezone"] = m[2]
	}
	db := stdlib.OpenDB(*config, stdlib.OptionAfterConnect(func(_ context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "timestamptz",
			OID:   pgtype.TimestamptzOID,
			Codec: &pgtype.TimestamptzCodec{ScanLocation: loc},
		})
		return nil
	}))
	return postgres.New(postgres.Config{Conn: db})
}

// timeZoneParam is gorm.io/driver/postgres's own matcher for a DSN's TimeZone.
var timeZoneParam = regexp.MustCompile("(time_zone|TimeZone)=(.*?)($|&| )")

func scanLocation(timeZone string) *time.Location {
	if timeZone == "" {
		return nil
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return nil
	}
	return loc
}
