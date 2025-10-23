package cmd

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestExecute(t *testing.T) {
	t.Run("Execute returns without error", func(t *testing.T) {
		rootCmd.SetArgs([]string{"--help"})
		err := Execute()
		assert.NoError(t, err)
	})

	t.Run("Execute with invalid command", func(t *testing.T) {
		rootCmd.SetArgs([]string{"invalid-command"})
		err := Execute()
		assert.Error(t, err)
	})
}

func TestRootCommand(t *testing.T) {
	t.Run("root command is initialized", func(t *testing.T) {
		assert.NotNil(t, rootCmd)
		assert.Equal(t, "pipeleak", rootCmd.Use)
		assert.Contains(t, rootCmd.Short, "Scan")
		assert.NotEmpty(t, rootCmd.Long)
		assert.NotEmpty(t, rootCmd.Example)
	})

	t.Run("root command has subcommands", func(t *testing.T) {
		commands := rootCmd.Commands()
		assert.Greater(t, len(commands), 0)

		commandNames := make([]string, 0)
		for _, cmd := range commands {
			commandNames = append(commandNames, cmd.Name())
		}

		expectedCommands := []string{"gh", "gl", "gluna", "bb", "ad", "gitea", "docs"}
		for _, expected := range expectedCommands {
			assert.Contains(t, commandNames, expected, "expected command %s not found", expected)
		}
	})

	t.Run("root command has persistent flags", func(t *testing.T) {
		jsonFlag := rootCmd.PersistentFlags().Lookup("json")
		assert.NotNil(t, jsonFlag)
		assert.Equal(t, "false", jsonFlag.DefValue)

		colorFlag := rootCmd.PersistentFlags().Lookup("coloredLog")
		assert.NotNil(t, colorFlag)
		assert.Equal(t, "true", colorFlag.DefValue)

		logfileFlag := rootCmd.PersistentFlags().Lookup("logfile")
		assert.NotNil(t, logfileFlag)
		assert.Equal(t, "", logfileFlag.DefValue)
	})

	t.Run("root command has groups", func(t *testing.T) {
		groups := rootCmd.Groups()
		assert.Greater(t, len(groups), 0)

		groupIDs := make([]string, 0)
		for _, group := range groups {
			groupIDs = append(groupIDs, group.ID)
		}

		expectedGroups := []string{"GitHub", "GitLab", "Helper", "BitBucket", "AzureDevOps", "Gitea"}
		for _, expected := range expectedGroups {
			assert.Contains(t, groupIDs, expected, "expected group %s not found", expected)
		}
	})
}

func TestCustomWriter(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectedWrite bool
		validateFunc  func(*testing.T, []byte, int, error)
	}{
		{
			name:          "writes data with newline",
			input:         []byte("test log message\n"),
			expectedWrite: true,
			validateFunc: func(t *testing.T, output []byte, n int, err error) {
				assert.NoError(t, err)
				assert.Greater(t, n, 0)
				assert.Contains(t, string(output), "test log message")
			},
		},
		{
			name:          "writes data without newline",
			input:         []byte("test log message"),
			expectedWrite: true,
			validateFunc: func(t *testing.T, output []byte, n int, err error) {
				assert.NoError(t, err)
				assert.Greater(t, n, 0)
				assert.Contains(t, string(output), "test log message")
			},
		},
		{
			name:          "handles empty input",
			input:         []byte(""),
			expectedWrite: true,
			validateFunc: func(t *testing.T, output []byte, n int, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:          "removes trailing newline and adds platform-specific newline",
			input:         []byte("message\n"),
			expectedWrite: true,
			validateFunc: func(t *testing.T, output []byte, n int, err error) {
				assert.NoError(t, err)
				assert.Greater(t, n, 0)
				outputStr := string(output)
				if runtime.GOOS == "windows" {
					assert.Contains(t, outputStr, "\n\r")
				} else {
					assert.Contains(t, outputStr, "\n")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-writer-*.log")
			assert.NoError(t, err)
			defer func() {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())
			}()

			cw := &CustomWriter{Writer: tmpFile}
			n, err := cw.Write(tt.input)

			_ = tmpFile.Sync()
			output, _ := os.ReadFile(tmpFile.Name())

			tt.validateFunc(t, output, n, err)
		})
	}
}

func TestCustomWriterWrite(t *testing.T) {
	t.Run("returns original length", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-writer-*.log")
		assert.NoError(t, err)
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}()

		cw := &CustomWriter{Writer: tmpFile}
		input := []byte("test message\n")
		n, err := cw.Write(input)

		assert.NoError(t, err)
		assert.Equal(t, len(input), n)
	})

	t.Run("handles multiple writes", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-writer-*.log")
		assert.NoError(t, err)
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}()

		cw := &CustomWriter{Writer: tmpFile}

		_, err = cw.Write([]byte("line1\n"))
		assert.NoError(t, err)

		_, err = cw.Write([]byte("line2\n"))
		assert.NoError(t, err)

		_ = tmpFile.Sync()
		content, _ := os.ReadFile(tmpFile.Name())
		contentStr := string(content)

		assert.Contains(t, contentStr, "line1")
		assert.Contains(t, contentStr, "line2")
	})
}

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name         string
		jsonOutput   bool
		logColor     bool
		logFile      string
		setupFunc    func()
		cleanupFunc  func()
		validateFunc func(*testing.T)
	}{
		{
			name:       "initializes logger with default settings",
			jsonOutput: false,
			logColor:   true,
			logFile:    "",
			setupFunc: func() {
				JsonLogoutput = false
				LogColor = true
				LogFile = ""
			},
			cleanupFunc: func() {},
			validateFunc: func(t *testing.T) {
				assert.NotNil(t, log.Logger)
			},
		},
		{
			name:       "initializes logger with JSON output",
			jsonOutput: true,
			logColor:   false,
			logFile:    "",
			setupFunc: func() {
				JsonLogoutput = true
				LogColor = false
				LogFile = ""
			},
			cleanupFunc: func() {},
			validateFunc: func(t *testing.T) {
				assert.NotNil(t, log.Logger)
			},
		},
		{
			name:       "initializes logger with log file",
			jsonOutput: false,
			logColor:   true,
			logFile:    "",
			setupFunc: func() {
				tmpFile, _ := os.CreateTemp("", "test-log-*.log")
				LogFile = tmpFile.Name()
				_ = tmpFile.Close()
				JsonLogoutput = false
				LogColor = true
			},
			cleanupFunc: func() {
				if LogFile != "" {
					_ = os.Remove(LogFile)
				}
			},
			validateFunc: func(t *testing.T) {
				assert.NotNil(t, log.Logger)
				assert.FileExists(t, LogFile)
			},
		},
		{
			name:       "initializes logger without color",
			jsonOutput: false,
			logColor:   false,
			logFile:    "",
			setupFunc: func() {
				JsonLogoutput = false
				LogColor = false
				LogFile = ""
			},
			cleanupFunc: func() {},
			validateFunc: func(t *testing.T) {
				assert.NotNil(t, log.Logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			defer tt.cleanupFunc()

			assert.NotPanics(t, func() {
				initLogger()
			})

			tt.validateFunc(t)
		})
	}
}

func TestPersistentPreRun(t *testing.T) {
	t.Run("PersistentPreRun initializes logger", func(t *testing.T) {
		assert.NotNil(t, rootCmd.PersistentPreRun)

		assert.NotPanics(t, func() {
			rootCmd.PersistentPreRun(rootCmd, []string{})
		})
	})
}

func TestGlobalVariables(t *testing.T) {
	t.Run("global variables exist and can be accessed", func(t *testing.T) {
		_ = JsonLogoutput
		_ = LogFile
		_ = LogColor
	})
}

func TestCustomWriterNewlineHandling(t *testing.T) {
	t.Run("handles platform-specific newlines correctly", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-newline-*.log")
		assert.NoError(t, err)
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}()

		cw := &CustomWriter{Writer: tmpFile}
		input := []byte("test\n")
		_, err = cw.Write(input)
		assert.NoError(t, err)

		_ = tmpFile.Sync()
		output, _ := os.ReadFile(tmpFile.Name())

		if runtime.GOOS == "windows" {
			assert.Contains(t, string(output), "\n\r", "Windows should have \\n\\r")
		} else {
			assert.True(t, bytes.HasSuffix(output, []byte("\n")), "Unix should have \\n")
		}
	})
}

func TestCommandExecution(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name:      "help command",
			args:      []string{"--help"},
			expectErr: false,
		},
		{
			name:      "version flag",
			args:      []string{"--version"},
			expectErr: true,
		},
		{
			name:      "json flag",
			args:      []string{"--json", "--help"},
			expectErr: false,
		},
		{
			name:      "coloredLog flag",
			args:      []string{"--coloredLog=false", "--help"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkCustomWriterWrite(b *testing.B) {
	tmpFile, _ := os.CreateTemp("", "bench-writer-*.log")
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	cw := &CustomWriter{Writer: tmpFile}
	data := []byte("test log message\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cw.Write(data)
	}
}

func BenchmarkInitLogger(b *testing.B) {
	JsonLogoutput = false
	LogColor = true
	LogFile = ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		initLogger()
	}
}
