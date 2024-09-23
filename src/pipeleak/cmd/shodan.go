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

type shodan struct {
	Module string `json:"module"`
}

type result struct {
	Hostnames []string `json:"hostnames"`
	Port      int      `json:"port"`
	IPString  string   `json:"ip_str"`
	Shodan    shodan   `json:"shodan"`
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
		log.Fatal().Stack().Err(err).Msg("Unable to parse shodan json flag")
	}

	scanCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return scanCmd
}

func Shodan(cmd *cobra.Command, args []string) {
	setLogLevel()

	jsonFile, err := os.Open(shodanJson)
	if err != nil {
		log.Fatal().Stack().Err(err)
	}
	defer jsonFile.Close()

	data, _ := io.ReadAll(jsonFile)
	ctx := context.Background()
	group := parallel.Unlimited(ctx)

	for _, line := range bytes.Split(data, []byte{'\n'}) {

		d := result{}
		_, err := marshmallow.Unmarshal(line, &d)
		if err != nil {
			log.Error().Stack().Err(err)
		} else {

			isHttps := true
			if strings.EqualFold("http", d.Shodan.Module) {
				isHttps = false
			}

			if len(d.Hostnames) == 0 {
				group.Go(func(ctx context.Context) {
					testHost(d.IPString, d.Port, isHttps)
				})
			} else {
				for _, hostname := range d.Hostnames {
					group.Go(func(ctx context.Context) {
						testHost(hostname, d.Port, isHttps)
					})
				}
			}
		}

	}

	group.Wait()
	log.Info().Msg("Done, Bye Bye 🏳️‍🌈🔥")
}

func testHost(hostname string, port int, https bool) {
	var url string
	if https {
		url = "https://" + hostname + ":" + strconv.Itoa(port)
	} else {
		url = "http://" + hostname + ":" + strconv.Itoa(port)
	}
	registration, err := isRegistrationEnabled(url)
	nrOfProjects, err := checkNrPublicRepos(url)
	if err != nil {
		log.Error().Stack().Err(err)
	} else {
		log.Info().Bool("registration", registration).Int("nrProjects", nrOfProjects).Str("url", url+"/explore").Msg("")
	}
}

func isRegistrationEnabled(base string) (bool, error) {
	u, err := url.Parse(base)
	if err != nil {
		return false, err
	}

	u.Path = path.Join(u.Path, "/users/somenotexistigusr/exists")
	s := u.String()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	res, err := client.Get(s)

	if err != nil {
		return false, err
	}

	if res.StatusCode == 200 {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			return false, err
		}

		// sanity check to avoid false positives
		if strings.Contains(string(resData), "{\"exists\":false}") {
			return true, nil
		}

		log.Debug().Msg("Missed sanity check")
		return false, err
	} else {
		log.Debug().Int("http", res.StatusCode).Msg("Registration username test request")
		return false, nil
	}
}

func checkNrPublicRepos(base string) (int, error) {
	u, err := url.Parse(base)
	if err != nil {
		return 0, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	u.Path = "/api/v4/projects"
	s := u.String()
	res, err := client.Get(s + "?per_page=100")
	if err != nil {
		return 0, err
	}

	if res.StatusCode == 200 {
		resData, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, err
		}
		var val []map[string]interface{}
		if err := json.Unmarshal(resData, &val); err != nil {
			return 0, err
		}
		return len(val), nil
	}

	return 0, err
}
