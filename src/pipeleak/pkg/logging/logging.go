package logging

import (
	"sync"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetLogLevel(verbose bool) {
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Verbose log output enabled")
	}
}

type ShortcutStatusFN func() *zerolog.Event

var (
	statusHookMutex sync.RWMutex
	statusHook      ShortcutStatusFN
)

// RegisterStatusHook allows commands to register a custom status function
func RegisterStatusHook(hook ShortcutStatusFN) {
	statusHookMutex.Lock()
	defer statusHookMutex.Unlock()
	statusHook = hook
}

// GetStatusHook returns the registered status hook or a default one
func GetStatusHook() ShortcutStatusFN {
	statusHookMutex.RLock()
	defer statusHookMutex.RUnlock()
	if statusHook != nil {
		return statusHook
	}
	return defaultStatusHook
}

func defaultStatusHook() *zerolog.Event {
	return log.Info().Str("status", "nothing to show")
}

func ShortcutListeners(status ShortcutStatusFN) {
	err := keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.CtrlC, keys.Escape:
			return true, nil
		case keys.RuneKey:
			if key.String() == "t" {
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Info().Str("logLevel", "trace").Msg("New Log level")
			}

			if key.String() == "d" {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Info().Str("logLevel", "debug").Msg("New Log level")
			}

			if key.String() == "i" {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				log.Info().Str("logLevel", "info").Msg("New Log level")
			}

			if key.String() == "w" {
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
				log.Info().Str("logLevel", "warn").Msg("New Log level")
			}

			if key.String() == "e" {
				zerolog.SetGlobalLevel(zerolog.ErrorLevel)
				log.Info().Str("logLevel", "error").Msg("New Log level")
			}

			if key.String() == "s" {
				// Use the registered status hook or default
				currentHook := GetStatusHook()
				log := currentHook()
				log.Msg("Status")
			}
		}

		return false, nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed hooking keyboard bindings")
	}
}
