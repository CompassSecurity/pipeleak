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

type Repository struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	Revision       int       `json:"revision"`
	Visibility     string    `json:"visibility"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}
