package activity

import "time"

type RemoteCommit struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	URL     string    `json:"url,omitempty"`
}

type RemoteProject struct {
	ProjectID         string         `json:"project_id"`
	ProjectName       string         `json:"project_name"`
	PathWithNamespace string         `json:"path_with_namespace"`
	Provider          string         `json:"provider"`
	Commits           []RemoteCommit `json:"commits"`
}

type LocalCommit struct {
	SHA            string    `json:"sha"`
	Message        string    `json:"message"`
	Branch         string    `json:"branch"`
	RepositoryName string    `json:"repository_name"`
	Author         string    `json:"author"`
	Date           time.Time `json:"date"`
}

type PlanningItem struct {
	Provider       string    `json:"provider"`
	ActivityType   string    `json:"activity_type"`
	ItemKey        string    `json:"item_key"`
	ItemTitle      string    `json:"item_title"`
	Status         string    `json:"status,omitempty"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	Comment        string    `json:"comment,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type ActiveRepo struct {
	RepositoryName string    `json:"repository_name"`
	RepositoryPath string    `json:"repository_path"`
	Source         string    `json:"source"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Summary struct {
	ReportDate         string          `json:"report_date"`
	RemoteGit          []RemoteProject `json:"remote_git"`
	LocalCommits       []LocalCommit   `json:"local_commits"`
	PlanningActivities []PlanningItem  `json:"planning_activities"`
	ActiveRepository   *ActiveRepo     `json:"active_repository,omitempty"`
}

func (s Summary) TotalRemoteCommits() int {
	total := 0
	for _, project := range s.RemoteGit {
		total += len(project.Commits)
	}
	return total
}

func (s Summary) HasAnyActivity() bool {
	if len(s.LocalCommits) > 0 || len(s.PlanningActivities) > 0 {
		return true
	}
	return s.TotalRemoteCommits() > 0
}
