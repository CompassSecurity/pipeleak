package devops

import "time"

type PaginatedResponse[T any] struct {
	Count             int `json:"count"`
	Value             []T `json:"value"`
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
