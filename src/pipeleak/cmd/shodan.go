package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
	"github.com/wandb/parallel"
)

var (
	shodanJson string
)

type result struct {
	Hostnames []string `json:"hostnames"`
	Port      int      `json:"port"`
}

func NewShodanCmd() *cobra.Command {
	scanCmd := &cobra.Command{
		Use:   "shodan [no options!]",
		Short: "Find self-registerable gitlab instances from shodan output",
		Run:   Shodan,
	}

	scanCmd.Flags().StringVarP(&shodanJson, "json", "j", "", "Shodan search export JSON file path")
	err := scanCmd.MarkFlagRequired("json")
	if err != nil {
		log.Error().Msg("Unable to parse shodan json flag: " + err.Error())
	}

	scanCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return scanCmd
}

func Shodan(cmd *cobra.Command, args []string) {
	setLogLevel()

	jsonFile, err := os.Open(shodanJson)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer jsonFile.Close()

	data, _ := io.ReadAll(jsonFile)
	ctx := context.Background()
	group := parallel.Limited(ctx, 50)
	for _, line := range bytes.Split(data, []byte{'\n'}) {

		d := result{}
		_, err := marshmallow.Unmarshal(line, &d)
		if err != nil {
			log.Error().Msg(err.Error())
		} else {
			for _, hostname := range d.Hostnames {
				group.Go(func(ctx context.Context) {
					testHostname(hostname, d.Port)
				})
			}
		}

	}

	group.Wait()
	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

func testHostname(hostname string, port int) {
	var url string
	if port == 443 {
		url = "https://" + hostname
	} else {
		url = "http://" + hostname
	}
	enabled, nrOfProjects := isRegistrationEnabled(url)
	if enabled {
		log.Info().Msg("public projects: " + strconv.Itoa(nrOfProjects) + " | " + url + "/explore")
	}
}

func isRegistrationEnabled(base string) (bool, int) {
	u, err := url.Parse(base)
	if err != nil {
		log.Debug().Msg(err.Error())
		return false, 0
	}

	u.Path = path.Join(u.Path, "/users/somenotexistigusr/exists")
	s := u.String()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	res, err := client.Get(s)

	if err != nil {
		log.Debug().Msg(err.Error())
		return false, 0
	}

	if res.StatusCode == 200 {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			log.Debug().Msg(err.Error())
			return false, 0
		}

		// sanity check to avoid false positives
		if strings.Contains(string(resData), "{\"exists\":false}") {
			nr := checkNrPublicRepos(client, u)
			return true, nr
		}

		log.Debug().Msg("Missed sanity check")
		return false, 0
	} else {
		log.Debug().Msg("resp: " + strconv.Itoa(res.StatusCode))
		return false, 0
	}
}

func checkNrPublicRepos(client *http.Client, u *url.URL) int {
	u.Path = "/api/v4/projects"
	s := u.String()
	res, err := client.Get(s + "?per_page=100")
	if err != nil {
		log.Debug().Msg(err.Error())
		return 0
	}

	if res.StatusCode == 200 {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			log.Debug().Msg(err.Error())
			return 0
		}
		var val []map[string]interface{}
		if err := json.Unmarshal(resData, &val); err != nil {
			log.Debug().Msg(err.Error())
			return 0
		}
		return len(val)
	}

	return 0
}