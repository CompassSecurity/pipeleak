package format

import (
"time"
"github.com/rs/zerolog/log"
)

func ParseISO8601(dateStr string) time.Time {
t, err := time.Parse(time.RFC3339, dateStr)
if err != nil {
log.Fatal().Err(err).Msg("Invalid date input, not ISO8601 compatible")
}
return t
}
