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
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
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

type PublicRepository struct {
	Type     string `json:"type"`
	FullName string `json:"full_name"`
	Links    struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
		Pullrequests struct {
			Href string `json:"href"`
		} `json:"pullrequests"`
		Commits struct {
			Href string `json:"href"`
		} `json:"commits"`
		Forks struct {
			Href string `json:"href"`
		} `json:"forks"`
		Watchers struct {
			Href string `json:"href"`
		} `json:"watchers"`
		Branches struct {
			Href string `json:"href"`
		} `json:"branches"`
		Tags struct {
			Href string `json:"href"`
		} `json:"tags"`
		Downloads struct {
			Href string `json:"href"`
		} `json:"downloads"`
		Source struct {
			Href string `json:"href"`
		} `json:"source"`
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
		Hooks struct {
			Href string `json:"href"`
		} `json:"hooks"`
	} `json:"links"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Scm         string `json:"scm"`
	Website     string `json:"website"`
	Owner       struct {
		DisplayName string `json:"display_name"`
		Links       struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
		Type      string `json:"type"`
		UUID      string `json:"uuid"`
		AccountID string `json:"account_id"`
		Nickname  string `json:"nickname"`
	} `json:"owner"`
	Workspace struct {
		Type  string `json:"type"`
		UUID  string `json:"uuid"`
		Name  string `json:"name"`
		Slug  string `json:"slug"`
		Links struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	} `json:"workspace"`
	IsPrivate bool `json:"is_private"`
	Project   struct {
		Type  string `json:"type"`
		Key   string `json:"key"`
		UUID  string `json:"uuid"`
		Name  string `json:"name"`
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	} `json:"project"`
	ForkPolicy string    `json:"fork_policy"`
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
	Size       int       `json:"size"`
	Language   string    `json:"language"`
	UUID       string    `json:"uuid"`
	Mainbranch struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"mainbranch"`
	OverrideSettings struct {
		DefaultMergeStrategy bool `json:"default_merge_strategy"`
		BranchingModel       bool `json:"branching_model"`
	} `json:"override_settings"`
	Parent                interface{} `json:"parent"`
	EnforcedSignedCommits interface{} `json:"enforced_signed_commits"`
	HasIssues             bool        `json:"has_issues"`
	HasWiki               bool        `json:"has_wiki"`
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

type Artifact struct {
	UUID          string    `json:"uuid"`
	StepUUID      string    `json:"step_uuid"`
	Path          string    `json:"path"`
	ArtifactType  string    `json:"artifactType"`
	FileSizeBytes int       `json:"file_size_bytes"`
	CreatedOn     time.Time `json:"created_on"`
	StorageType   string    `json:"storageType"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
}

type DownloadArtifact struct {
	Name      string    `json:"name"`
	Size      int       `json:"size"`
	CreatedOn time.Time `json:"created_on"`
	User      struct {
		Type  string `json:"type"`
		Links struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
		CreatedOn     time.Time `json:"created_on"`
		DisplayName   string    `json:"display_name"`
		UUID          string    `json:"uuid"`
		AccountID     string    `json:"account_id"`
		AccountStatus string    `json:"account_status"`
		Kind          string    `json:"kind"`
	} `json:"user"`
	Downloads int `json:"downloads"`
	Links     struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
	Type string `json:"type"`
}
