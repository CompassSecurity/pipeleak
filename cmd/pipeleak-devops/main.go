package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/CompassSecurity/pipeleak/internal/cmd/devops"
	"github.com/CompassSecurity/pipeleak/pkg/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Version information - set via ldflags during build
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var (
	originalTermState *term.State
	JsonLogoutput     bool
	LogFile           string
	LogColor          bool
	LogDebug          bool
	LogLevel          string
)

// TerminalRestorer is a function that can be called to restore terminal state
var TerminalRestorer func()

type TerminalRestoringWriter struct {
	underlying io.Writer
}

func (w *TerminalRestoringWriter) Write(p []byte) (n int, err error) {
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err == nil {
		if level, ok := logEntry["level"].(string); ok && level == "fatal" {
			_, _ = w.underlying.Write(p)
			restoreTerminalState()
			os.Exit(1)
		}
	}
	return w.underlying.Write(p)
}

func main() {
	saveTerminalState()
	defer restoreTerminalState()

	TerminalRestorer = restoreTerminalState

	rootCmd := newRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	adCmd := devops.NewAzureDevOpsRootCmd()
	adCmd.Use = "pipeleak-devops"
	adCmd.Short = "Scan Azure DevOps Pipelines logs and artifacts for secrets"
	adCmd.Long = `Pipeleak-DevOps is a tool designed to scan Azure DevOps Pipelines job output logs and artifacts for potential secrets.

This is a standalone binary for Azure DevOps-specific functionality.`
	adCmd.Version = Version
	adCmd.GroupID = ""

	adCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		initLogger(cmd)
		setGlobalLogLevel(cmd)
		go logging.ShortcutListeners(nil)
	}

	adCmd.PersistentFlags().BoolVarP(&JsonLogoutput, "json", "", false, "Use JSON as log output format")
	adCmd.PersistentFlags().StringVarP(&LogFile, "logfile", "l", "", "Log output to a file")
	adCmd.PersistentFlags().BoolVarP(&LogDebug, "verbose", "v", false, "Enable debug logging (shortcut for --log-level=debug)")
	adCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "", "Set log level globally (debug, info, warn, error). Example: --log-level=warn")
	adCmd.PersistentFlags().BoolVar(&LogColor, "color", true, "Enable colored log output (auto-disabled when using --logfile)")

	adCmd.SetVersionTemplate(`{{.Version}}
`)

	return adCmd
}

func saveTerminalState() {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, err := term.GetState(int(os.Stdin.Fd()))
		if err == nil {
			originalTermState = state
		}
	}
}

func restoreTerminalState() {
	if originalTermState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), originalTermState)
	}
}

// FatalHook is a zerolog hook that restores terminal state before fatal exits
type FatalHook struct{}

func (h FatalHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level == zerolog.FatalLevel {
		if TerminalRestorer != nil {
			TerminalRestorer()
		}
	}
}

func initLogger(cmd *cobra.Command) {
	defaultOut := os.Stdout
	colorEnabled := LogColor

	if LogFile != "" {
		// #nosec G304 - User-provided log file path via --log-file flag, user controls their own filesystem
		runLogFile, err := os.OpenFile(
			LogFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0600,
		)
		if err != nil {
			panic(err)
		}
		defaultOut = runLogFile

		rootFlags := cmd.Root().PersistentFlags()
		if !rootFlags.Changed("color") {
			colorEnabled = false
		}
	}

	fatalHook := FatalHook{}

	if JsonLogoutput {
		hitWriter := &logging.HitLevelWriter{}
		hitWriter.SetOutput(defaultOut)
		logging.SetGlobalHitWriter(hitWriter)
		log.Logger = zerolog.New(hitWriter).With().Timestamp().Logger().Hook(fatalHook)
	} else {
		output := zerolog.ConsoleWriter{
			Out:        defaultOut,
			TimeFormat: "2006-01-02T15:04:05Z07:00",
			NoColor:    !colorEnabled,
		}
		hitWriter := &logging.HitLevelWriter{}
		hitWriter.SetOutput(&output)
		logging.SetGlobalHitWriter(hitWriter)
		log.Logger = zerolog.New(hitWriter).With().Timestamp().Logger().Hook(fatalHook)
	}
}

func setGlobalLogLevel(cmd *cobra.Command) {
	if LogLevel != "" {
		switch LogLevel {
		case "trace":
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
			log.Trace().Msg("Log level set to trace (explicit)")
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			log.Debug().Msg("Log level set to debug (explicit)")
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			log.Info().Msg("Log level set to info (explicit)")
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
			log.Warn().Msg("Log level set to warn (explicit)")
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
			log.Error().Msg("Log level set to error (explicit)")
		default:
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			log.Warn().Str("logLevelSpecified", LogLevel).Msg("Invalid log level, defaulting to info")
		}
		return
	}

	if LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Log level set to debug (-v)")
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("Log level set to info (default)")
}
