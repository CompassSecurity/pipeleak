package logging

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// HitLevel defines a custom log level for security finding hits.
// It is implemented as a wrapper around WarnLevel but appears as "hit" in JSON output.
// This level is used to distinguish security scan results from regular warnings.
const HitLevel zerolog.Level = zerolog.WarnLevel

// HitLevelWriter wraps an io.Writer to transform logs with the hit marker to use "hit" as the level.
// This allows Hit() logs to appear with "level":"hit" in JSON output while maintaining compatibility with zerolog.
type HitLevelWriter struct {
	out       io.Writer
	mu        sync.Mutex
	nextIsHit bool
}

// Write processes log output, replacing the level field with "hit" for marked events.
func (w *HitLevelWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	isHit := w.nextIsHit
	w.nextIsHit = false
	w.mu.Unlock()

	if isHit && len(p) > 0 {
		// Parse JSON and replace level field
		var logEntry map[string]interface{}
		if err := json.Unmarshal(p, &logEntry); err == nil {
			if logEntry["level"] == "warn" {
				logEntry["level"] = "hit"
			}
			// Remove the internal hit marker field
			delete(logEntry, "_hit")

			if newBytes, err := json.Marshal(logEntry); err == nil {
				newBytes = append(newBytes, '\n')
				return w.out.Write(newBytes)
			}
		}
	}

	return w.out.Write(p)
}

// markNextAsHit flags the next log event to be transformed to hit level.
func (w *HitLevelWriter) markNextAsHit() {
	w.mu.Lock()
	w.nextIsHit = true
	w.mu.Unlock()
}

// NewHitLevelWriter creates a new HitLevelWriter that wraps the given io.Writer.
// This is useful for testing or creating custom Hit loggers.
func NewHitLevelWriter(out io.Writer) *HitLevelWriter {
	return &HitLevelWriter{out: out}
}

// HitEvent wraps a zerolog.Event to mark it as a hit-level log.
// It provides a fluent interface for building log events that will be output with "level":"hit".
type HitEvent struct {
	event  *zerolog.Event
	writer *HitLevelWriter
}

// Str adds a string field to the hit log event.
func (h *HitEvent) Str(key, val string) *HitEvent {
	h.event.Str(key, val)
	return h
}

// Int adds an integer field to the hit log event.
func (h *HitEvent) Int(key string, val int) *HitEvent {
	h.event.Int(key, val)
	return h
}

// Bool adds a boolean field to the hit log event.
func (h *HitEvent) Bool(key string, val bool) *HitEvent {
	h.event.Bool(key, val)
	return h
}

// Err adds an error field to the hit log event.
func (h *HitEvent) Err(err error) *HitEvent {
	h.event.Err(err)
	return h
}

// Msg sends the hit log event with the specified message.
// The log will have "level": "hit" in JSON output.
func (h *HitEvent) Msg(msg string) {
	if h.writer != nil {
		h.writer.markNextAsHit()
	}
	// Add internal marker for hit logs (will be removed by writer)
	h.event.Bool("_hit", true).Msg(msg)
}

// globalHitWriter is used for the global Hit() function.
var globalHitWriter *HitLevelWriter
var globalHitWriterOnce sync.Once

// setupGlobalHitWriter ensures the global logger uses a HitLevelWriter.
func setupGlobalHitWriter() {
	globalHitWriterOnce.Do(func() {
		// Create a HitLevelWriter that wraps stdout/stderr
		// In production, this will wrap the actual configured output
		out := os.Stderr
		globalHitWriter = &HitLevelWriter{out: out}

		// Wrap the global logger to use the HitLevelWriter
		// This preserves any existing configuration (timestamp, etc.)
		log.Logger = zerolog.New(globalHitWriter).With().Timestamp().Logger()
	})
}

// Hit creates a hit-level log event using the global logger.
// This is the primary method for logging security findings.
// The resulting log will have "level":"hit" in JSON output instead of "level":"warn".
// Example: logging.Hit().Str("ruleName", "secret-key").Str("value", "***").Msg("HIT")
func Hit() *HitEvent {
	// Only setup if not already done (e.g., in tests)
	if globalHitWriter == nil {
		setupGlobalHitWriter()
	}
	return &HitEvent{
		event:  log.Warn(),
		writer: globalHitWriter,
	}
}

// ParseLevel extends zerolog's ParseLevel to support "hit" level.
// It returns HitLevel when "hit" is specified, otherwise delegates to zerolog.ParseLevel.
func ParseLevel(levelStr string) (zerolog.Level, error) {
	if levelStr == "hit" {
		return HitLevel, nil
	}
	return zerolog.ParseLevel(levelStr)
}

// SetGlobalHitWriter sets the global HitLevelWriter for testing purposes.
// This should only be used in tests to capture hit log output.
func SetGlobalHitWriter(writer *HitLevelWriter) {
	globalHitWriter = writer
}
