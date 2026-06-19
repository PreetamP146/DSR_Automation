package gitactivity

import "time"

type Commit struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	URL     string    `json:"url"`
}

type ProjectActivity struct {
	ProjectID         string   `json:"project_id"`
	ProjectName       string   `json:"project_name"`
	PathWithNamespace string   `json:"path_with_namespace"`
	Provider          string   `json:"provider"`
	Commits           []Commit `json:"commits"`
}
