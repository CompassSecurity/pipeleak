package cmd

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/perimeterx/marshmallow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	shodanJson string
)

type result struct {
	Hostnames []string `json:"hostnames"`
	Port      int      `json:"port"`
}

func NewFindCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "find [no options!]",
		Short: "Find self-registerable gitlab instances from shodan output",
		Run:   Find,
	}

	scanCmd.Flags().StringVarP(&shodanJson, "json", "j", "", "Shodan search export JSON file path")
	err := scanCmd.MarkFlagRequired("json")
	if err != nil {
		log.Error().Msg("Unable to parse shodan json flag: " + err.Error())
	}

	scanCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return scanCmd
}

func Find(cmd *cobra.Command, args []string) {
	setLogLevel()

	jsonFile, err := os.Open(shodanJson)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer jsonFile.Close()

	data, _ := io.ReadAll(jsonFile)
	for _, line := range bytes.Split(data, []byte{'\n'}) {

		d := result{}
		_, err := marshmallow.Unmarshal(line, &d)
		if err != nil {
			log.Error().Msg(err.Error())
		} else {
			for _, hostname := range d.Hostnames {
				url := hostname
				if d.Port == 443 {
					url = "https://" + hostname
				} else {
					url = "http://" + hostname
				}
				enabled := isRegistrationEnabled(url)
				if enabled {
					log.Info().Msg(url)
				}
			}
		}

	}

	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

func isRegistrationEnabled(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		log.Error().Msg(err.Error())
	}

	u.Path = path.Join(u.Path, "/users/somenotexistigusr/exists")
	s := u.String()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	res, err := client.Get(s)

	if err != nil {
		log.Debug().Msg(err.Error())
		return false
	}

	if res.StatusCode == 200 {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			log.Debug().Msg(err.Error())
			return false
		}

		// sanity check to avoid false positives
		if strings.Contains(string(resData), "{\"exists\":false}") {
			return true
		}

		log.Debug().Msg("Missed sanity check")
		return false
	} else {
		log.Debug().Msg("resp: " + strconv.Itoa(res.StatusCode))
		return false
	}
}
