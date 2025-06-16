package gitlab

import (
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/CompassSecurity/pipeleak/cmd/gitlab/util"
	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"resty.dev/v3"
)

var minAccessLevel int

func NewEnumCmd() *cobra.Command {
	enumCmd := &cobra.Command{
		Use:   "enum [no options!]",
		Short: "Enumerate access rights of a Gitlab access token",
		Run:   Enum,
	}
	enumCmd.Flags().StringVarP(&gitlabUrl, "gitlab", "g", "", "GitLab instance URL")
	err := enumCmd.MarkFlagRequired("gitlab")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Unable to require gitlab flag")
	}

	enumCmd.Flags().StringVarP(&gitlabApiToken, "token", "t", "", "GitLab API Token")
	err = enumCmd.MarkFlagRequired("token")
	if err != nil {
		log.Fatal().Msg("Unable to require token flag")
	}
	enumCmd.MarkFlagsRequiredTogether("gitlab", "token")

	enumCmd.PersistentFlags().IntVarP(&minAccessLevel, "level", "", int(gitlab.GuestPermissions), "Minimum repo access level. See https://docs.gitlab.com/api/access_requests/#valid-access-levels for integer values")

	enumCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	return enumCmd
}

func Enum(cmd *cobra.Command, args []string) {
	helper.SetLogLevel(verbose)
	git, err := util.GetGitlabClient(gitlabApiToken, gitlabUrl)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed creating gitlab client")
	}

	user, _, err := git.Users.CurrentUser()

	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed fetching current usert")
	}

	log.Info().Msg("Enumerating User")
	log.Warn().Str("username", user.Username).Str("name", user.Name).Str("email", user.Email).Bool("admin", user.IsAdmin).Bool("bot", user.Bot).Msg("Current user")

	log.Info().Msg("Enumerating Access Token")
	client := *resty.New().SetRedirectPolicy(resty.FlexibleRedirectPolicy(5))
	enumCurrentToken(client, gitlabUrl, gitlabApiToken)

	log.Info().Msg("Enumerating Projects and Groups")
	page := 1
	for page != -1 {
		page = listTokenAssociations(client, gitlabUrl, gitlabApiToken, minAccessLevel, page)
	}

	log.Info().Msg("Done")
}

type TokenAssociations struct {
	Groups []struct {
		ID             int         `json:"id"`
		WebURL         string      `json:"web_url"`
		Name           string      `json:"name"`
		ParentID       interface{} `json:"parent_id"`
		OrganizationID int         `json:"organization_id"`
		AccessLevels   int         `json:"access_levels"`
		Visibility     string      `json:"visibility"`
	} `json:"groups"`
	Projects []struct {
		ID                int       `json:"id"`
		Description       string    `json:"description"`
		Name              string    `json:"name"`
		NameWithNamespace string    `json:"name_with_namespace"`
		Path              string    `json:"path"`
		PathWithNamespace string    `json:"path_with_namespace"`
		CreatedAt         time.Time `json:"created_at"`
		AccessLevels      struct {
			ProjectAccessLevel int `json:"project_access_level"`
			GroupAccessLevel   int `json:"group_access_level"`
		} `json:"access_levels"`
		Visibility string `json:"visibility"`
		WebURL     string `json:"web_url"`
		Namespace  struct {
			ID        int         `json:"id"`
			Name      string      `json:"name"`
			Path      string      `json:"path"`
			Kind      string      `json:"kind"`
			FullPath  string      `json:"full_path"`
			ParentID  interface{} `json:"parent_id"`
			AvatarURL string      `json:"avatar_url"`
			WebURL    string      `json:"web_url"`
		} `json:"namespace"`
	} `json:"projects"`
}

type SelfToken struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Revoked     bool      `json:"revoked"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Scopes      []string  `json:"scopes"`
	UserID      int       `json:"user_id"`
	LastUsedAt  time.Time `json:"last_used_at"`
	Active      bool      `json:"active"`
	ExpiresAt   string    `json:"expires_at"`
	LastUsedIps []string  `json:"last_used_ips"`
}

func enumCurrentToken(client resty.Client, baseUrl string, pat string) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse base URL")
	}
	u.Path = path.Join(u.Path, "api/v4/personal_access_tokens/self")
	currentToken := &SelfToken{}
	res, err := client.R().
		SetHeader("PRIVATE-TOKEN", pat).
		SetResult(currentToken).
		Get(u.String())

	if err != nil {
		log.Error().Err(err).Str("url", u.String()).Msg("Failed fetching token details (network or client error)")
		return
	}

	if res != nil && res.StatusCode() != 200 {
		log.Error().Int("status", res.StatusCode()).Str("url", u.String()).Str("response", res.String()).Msg("Failed fetching token details (HTTP error)")
		return
	}

	log.Warn().
		Int("id", currentToken.ID).
		Str("name", currentToken.Name).
		Bool("revoked", currentToken.Revoked).
		Time("created", currentToken.CreatedAt).
		Str("description", currentToken.Description).
		Str("scopes", strings.Join(currentToken.Scopes, ",")).
		Int("userId", currentToken.UserID).
		Time("lastUsedAt", currentToken.LastUsedAt).
		Bool("active", currentToken.Active).
		Str("lastUsedIps", strings.Join(currentToken.LastUsedIps, ",")).
		Msg("Current Token")
}

// https://docs.gitlab.com/api/personal_access_tokens/#list-all-token-associations
func listTokenAssociations(client resty.Client, baseUrl string, pat string, accessLevel int, page int) int {
	u, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse base URL")
	}
	u.Path = path.Join(u.Path, "api/v4/personal_access_tokens/self/associations")
	resp := &TokenAssociations{}
	res, err := client.R().
		SetHeader("PRIVATE-TOKEN", pat).
		SetResult(resp).
		SetQueryParam("min_access_level", strconv.Itoa(accessLevel)).
		SetQueryParam("per_page", "100").
		SetQueryParam("page", strconv.Itoa(page)).
		Get(u.String())

	if err != nil {
		log.Error().Err(err).Str("url", u.String()).Msg("Failed fetching token associations (network or client error)")
		return -1
	}
	if res != nil && res.StatusCode() != 200 {
		log.Error().Int("status", res.StatusCode()).Str("url", u.String()).Str("response", res.String()).Msg("Failed fetching token associations (HTTP error)")
		return -1
	}

	for _, group := range resp.Groups {
		log.Warn().Str("group", group.WebURL).Int("accessLevel", group.AccessLevels).Str("name", group.Name).Str("visibility", string(group.Visibility)).Msg("Group")
	}

	for _, project := range resp.Projects {
		log.Warn().Str("project", project.WebURL).Str("name", project.NameWithNamespace).Int("groupAccessLevel", project.AccessLevels.GroupAccessLevel).Int("projectAccessLevel", project.AccessLevels.ProjectAccessLevel).Msg("Project")
	}

	nextPage, err := strconv.Atoi(res.Header().Get("x-next-page"))
	if err != nil {
		nextPage = -1
	}

	return nextPage
}
