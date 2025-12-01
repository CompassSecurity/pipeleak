package scan

import "time"

type PaginatedResponse[T any] struct {
	Count int `json:"count"`
	Value []T `json:"value"`
}

type AuthenticatedUser struct {
	DisplayName  string    `json:"displayName"`
	PublicAlias  string    `json:"publicAlias"`
	EmailAddress string    `json:"emailAddress"`
	CoreRevision int       `json:"coreRevision"`
	TimeStamp    time.Time `json:"timeStamp"`
	ID           string    `json:"id"`
	Revision     int       `json:"revision"`
}

type Account struct {
	AccountID   string `json:"accountId"`
	AccountURI  string `json:"accountUri"`
	AccountName string `json:"accountName"`
}

type Project struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	Revision       int       `json:"revision"`
	Visibility     string    `json:"visibility"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

type Build struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Web struct {
			Href string `json:"href"`
		} `json:"web"`
		SourceVersionDisplayURI struct {
			Href string `json:"href"`
		} `json:"sourceVersionDisplayUri"`
		Timeline struct {
			Href string `json:"href"`
		} `json:"timeline"`
		Badge struct {
			Href string `json:"href"`
		} `json:"badge"`
	} `json:"_links"`
	Properties struct {
	} `json:"properties"`
	Tags              []interface{} `json:"tags"`
	ValidationResults []interface{} `json:"validationResults"`
	Plans             []struct {
		PlanID string `json:"planId"`
	} `json:"plans"`
	TriggerInfo struct {
	} `json:"triggerInfo"`
	ID          int       `json:"id"`
	BuildNumber string    `json:"buildNumber"`
	Status      string    `json:"status"`
	Result      string    `json:"result"`
	QueueTime   time.Time `json:"queueTime"`
	StartTime   time.Time `json:"startTime"`
	FinishTime  time.Time `json:"finishTime"`
	URL         string    `json:"url"`
	Definition  struct {
		Drafts      []interface{} `json:"drafts"`
		ID          int           `json:"id"`
		Name        string        `json:"name"`
		URL         string        `json:"url"`
		URI         string        `json:"uri"`
		Path        string        `json:"path"`
		Type        string        `json:"type"`
		QueueStatus string        `json:"queueStatus"`
		Revision    int           `json:"revision"`
		Project     struct {
			ID             string    `json:"id"`
			Name           string    `json:"name"`
			URL            string    `json:"url"`
			State          string    `json:"state"`
			Revision       int       `json:"revision"`
			Visibility     string    `json:"visibility"`
			LastUpdateTime time.Time `json:"lastUpdateTime"`
		} `json:"project"`
	} `json:"definition"`
	BuildNumberRevision int `json:"buildNumberRevision"`
	Project             struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		URL            string    `json:"url"`
		State          string    `json:"state"`
		Revision       int       `json:"revision"`
		Visibility     string    `json:"visibility"`
		LastUpdateTime time.Time `json:"lastUpdateTime"`
	} `json:"project"`
	URI           string `json:"uri"`
	SourceBranch  string `json:"sourceBranch"`
	SourceVersion string `json:"sourceVersion"`
	Queue         struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Pool struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			IsHosted bool   `json:"isHosted"`
		} `json:"pool"`
	} `json:"queue"`
	Priority     string `json:"priority"`
	Reason       string `json:"reason"`
	RequestedFor struct {
		DisplayName string `json:"displayName"`
		URL         string `json:"url"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"_links"`
		ID         string `json:"id"`
		UniqueName string `json:"uniqueName"`
		ImageURL   string `json:"imageUrl"`
		Descriptor string `json:"descriptor"`
	} `json:"requestedFor"`
	RequestedBy struct {
		DisplayName string `json:"displayName"`
		URL         string `json:"url"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"_links"`
		ID         string `json:"id"`
		UniqueName string `json:"uniqueName"`
		ImageURL   string `json:"imageUrl"`
		Descriptor string `json:"descriptor"`
	} `json:"requestedBy"`
	LastChangedDate time.Time `json:"lastChangedDate"`
	LastChangedBy   struct {
		DisplayName string `json:"displayName"`
		URL         string `json:"url"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"_links"`
		ID         string `json:"id"`
		UniqueName string `json:"uniqueName"`
		ImageURL   string `json:"imageUrl"`
		Descriptor string `json:"descriptor"`
	} `json:"lastChangedBy"`
	OrchestrationPlan struct {
		PlanID string `json:"planId"`
	} `json:"orchestrationPlan"`
	Logs struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"logs"`
	Repository struct {
		ID                 string      `json:"id"`
		Type               string      `json:"type"`
		Name               string      `json:"name"`
		URL                string      `json:"url"`
		Clean              interface{} `json:"clean"`
		CheckoutSubmodules bool        `json:"checkoutSubmodules"`
	} `json:"repository"`
	RetainedByRelease            bool        `json:"retainedByRelease"`
	TriggeredByBuild             interface{} `json:"triggeredByBuild"`
	AppendCommitMessageToRunName bool        `json:"appendCommitMessageToRunName"`
}

type BuildLog struct {
	LineCount     int       `json:"lineCount"`
	CreatedOn     time.Time `json:"createdOn"`
	LastChangedOn time.Time `json:"lastChangedOn"`
	ID            int       `json:"id"`
	Type          string    `json:"type"`
	URL           string    `json:"url"`
}

type Artifact struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Source   string `json:"source"`
	Resource struct {
		Type       string `json:"type"`
		Data       string `json:"data"`
		Properties struct {
			Localpath    string `json:"localpath"`
			Artifactsize string `json:"artifactsize"`
		} `json:"properties"`
		URL         string `json:"url"`
		DownloadURL string `json:"downloadUrl"`
	} `json:"resource"`
}
