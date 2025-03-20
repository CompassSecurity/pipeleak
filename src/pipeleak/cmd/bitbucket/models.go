package bitbucket

import "time"

type PaginatedResponse[T any] struct {
	Pagelen  int    `json:"pagelen"`
	Page     int    `json:"page"`
	Size     int    `json:"size"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Values   []T    `json:"values"`
}

type Workspace struct {
	UUID  string `json:"uuid"`
	Links struct {
		Owners struct {
			Href string `json:"href"`
		} `json:"owners"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Repositories struct {
			Href string `json:"href"`
		} `json:"repositories"`
		Snippets struct {
			Href string `json:"href"`
		} `json:"snippets"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
		Members struct {
			Href string `json:"href"`
		} `json:"members"`
		Projects struct {
			Href string `json:"href"`
		} `json:"projects"`
	} `json:"links"`
	CreatedOn time.Time `json:"created_on"`
	Type      string    `json:"type"`
	Slug      string    `json:"slug"`
	IsPrivate bool      `json:"is_private"`
	Name      string    `json:"name"`
}

type Repository struct {
	Type  string `json:"type"`
	Links struct {
		Self struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"self"`
		HTML struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"avatar"`
		Pullrequests struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"pullrequests"`
		Commits struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"commits"`
		Forks struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"forks"`
		Watchers struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"watchers"`
		Downloads struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"downloads"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
		Hooks struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"hooks"`
	} `json:"links"`
	UUID      string `json:"uuid"`
	FullName  string `json:"full_name"`
	IsPrivate bool   `json:"is_private"`
	Scm       string `json:"scm"`
	Owner     struct {
		Type string `json:"type"`
	} `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Size        int    `json:"size"`
	Language    string `json:"language"`
	HasIssues   bool   `json:"has_issues"`
	HasWiki     bool   `json:"has_wiki"`
	ForkPolicy  string `json:"fork_policy"`
	Project     struct {
		Type string `json:"type"`
	} `json:"project"`
	Mainbranch struct {
		Type string `json:"type"`
	} `json:"mainbranch"`
}

type Pipeline struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	Creator     struct {
		Type string `json:"type"`
	} `json:"creator"`
	Repository struct {
		Type string `json:"type"`
	} `json:"repository"`
	Target struct {
		Type string `json:"type"`
	} `json:"target"`
	Trigger struct {
		Type string `json:"type"`
	} `json:"trigger"`
	State struct {
		Type string `json:"type"`
	} `json:"state"`
	Variables []struct {
		Type string `json:"type"`
	} `json:"variables"`
	CreatedOn            string `json:"created_on"`
	CompletedOn          string `json:"completed_on"`
	BuildSecondsUsed     int    `json:"build_seconds_used"`
	ConfigurationSources []struct {
		Source string `json:"source"`
		URI    string `json:"uri"`
	} `json:"configuration_sources"`
	Links struct {
		Type string `json:"type"`
	} `json:"links"`
}

type PipelineStep struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	StartedOn   string `json:"started_on"`
	CompletedOn string `json:"completed_on"`
	State       struct {
		Type string `json:"type"`
	} `json:"state"`
	Image struct {
	} `json:"image"`
	SetupCommands []struct {
	} `json:"setup_commands"`
	ScriptCommands []struct {
	} `json:"script_commands"`
}
