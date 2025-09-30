package cmd

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/CompassSecurity/pipeleak/cmd/bitbucket"
	"github.com/CompassSecurity/pipeleak/cmd/devops"
	"github.com/CompassSecurity/pipeleak/cmd/docs"
	"github.com/CompassSecurity/pipeleak/cmd/github"
	"github.com/CompassSecurity/pipeleak/cmd/gitlab"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "pipeleak",
		Short:   "Scan job logs and artifacts for secrets",
		Long:    "Pipeleak is a tool designed to scan CI/CD job output logs and artifacts for potential secrets.",
		Example: "pipeleak gl scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.com",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogger()
		},
	}
	JsonLogoutput bool
	LogFile       string
	LogColor      bool
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(github.NewGitHubRootCmd())
	rootCmd.AddCommand(gitlab.NewGitLabRootCmd())
	rootCmd.AddCommand(gitlab.NewGitLabRootUnauthenticatedCmd())
	rootCmd.AddCommand(bitbucket.NewBitBucketRootCmd())
	rootCmd.AddCommand(devops.NewAzureDevOpsRootCmd())
	rootCmd.AddCommand(docs.NewDocsCmd(rootCmd))
	rootCmd.PersistentFlags().BoolVarP(&JsonLogoutput, "json", "", false, "Use JSON as log output format")
	rootCmd.PersistentFlags().BoolVarP(&LogColor, "coloredLog", "", true, "Output the human-readable log in color")
	rootCmd.PersistentFlags().StringVarP(&LogFile, "logfile", "l", "", "Log output to a file")

	rootCmd.AddGroup(&cobra.Group{ID: "GitHub", Title: "GitHub Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "GitLab", Title: "GitLab Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "Helper", Title: "Various Helper Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "BitBucket", Title: "BitBucket Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "AzureDevOps", Title: "Azure DevOps Commands"})
}

type CustomWriter struct {
	Writer *os.File
}

func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	originalLen := len(p)
	if bytes.HasSuffix(p, []byte("\n")) {
		p = bytes.TrimSuffix(p, []byte("\n"))
	}

	// necessary as to: https://github.com/rs/zerolog/blob/master/log.go#L474
	newlineChars := []byte("\n")
	if runtime.GOOS == "windows" {
		newlineChars = []byte("\n\r")
	}

	modified := append(p, newlineChars...)

	written, err := cw.Writer.Write(modified)
	if err != nil {
		return 0, err
	}

	if written != len(modified) {
		return 0, io.ErrShortWrite
	}

	return originalLen, nil
}

func initLogger() {
	defaultOut := &CustomWriter{Writer: os.Stdout}
	if LogFile != "" {
		runLogFile, err := os.OpenFile(
			LogFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0664,
		)
		if err != nil {
			panic(err)
		}
		defaultOut = &CustomWriter{Writer: runLogFile}
	}

	if JsonLogoutput {
		log.Logger = zerolog.New(defaultOut).With().Timestamp().Logger()
	} else {
		output := zerolog.ConsoleWriter{Out: defaultOut, TimeFormat: time.RFC3339, NoColor: !LogColor}
		log.Logger = zerolog.New(output).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
