package devops

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

type Pipeline struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Web struct {
			Href string `json:"href"`
		} `json:"web"`
	} `json:"_links"`
	URL      string `json:"url"`
	ID       int    `json:"id"`
	Revision int    `json:"revision"`
	Name     string `json:"name"`
	Folder   string `json:"folder"`
}

type PipelineRun struct {
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Web struct {
			Href string `json:"href"`
		} `json:"web"`
		PipelineWeb struct {
			Href string `json:"href"`
		} `json:"pipeline.web"`
		Pipeline struct {
			Href string `json:"href"`
		} `json:"pipeline"`
	} `json:"_links"`
	TemplateParameters struct {
	} `json:"templateParameters"`
	Pipeline struct {
		URL      string `json:"url"`
		ID       int    `json:"id"`
		Revision int    `json:"revision"`
		Name     string `json:"name"`
		Folder   string `json:"folder"`
	} `json:"pipeline"`
	State        string    `json:"state"`
	Result       string    `json:"result"`
	CreatedDate  time.Time `json:"createdDate"`
	FinishedDate time.Time `json:"finishedDate"`
	URL          string    `json:"url"`
	ID           int       `json:"id"`
	Name         string    `json:"name"`
}
