package dto

import "time"

type RegisterLocalGitRepoRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type UpdateTrackedLocalReposRequest struct {
	RepositoryIDs []string `json:"repository_ids"`
}

type LocalGitRepositoryResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Path          string     `json:"path"`
	IsTracked     bool       `json:"is_tracked"`
	LastScannedAt *time.Time `json:"last_scanned_at,omitempty"`
}

type ListLocalGitReposResponse struct {
	Repositories []LocalGitRepositoryResponse `json:"repositories"`
	Total        int                          `json:"total"`
}

type LocalGitCommitResponse struct {
	ID             string    `json:"id"`
	SHA            string    `json:"sha"`
	Message        string    `json:"message"`
	Branch         string    `json:"branch"`
	RepositoryName string    `json:"repository_name"`
	Author         string    `json:"author"`
	CommittedAt    time.Time `json:"committed_at"`
}

type ConnectPlanningRequest struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	Email    string `json:"email"`
	APIToken string `json:"api_token"`
	APIKey   string `json:"api_key"`
}

type PlanningIntegrationResponse struct {
	ID          string     `json:"id"`
	Provider    string     `json:"provider"`
	BaseURL     string     `json:"base_url,omitempty"`
	Email       string     `json:"email,omitempty"`
	DisplayName string     `json:"display_name"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
}

type ListPlanningIntegrationsResponse struct {
	Integrations []PlanningIntegrationResponse `json:"integrations"`
	Total        int                           `json:"total"`
}

type PlanningActivityResponse struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	ActivityType   string    `json:"activity_type"`
	ItemKey        string    `json:"item_key"`
	ItemTitle      string    `json:"item_title"`
	Status         string    `json:"status,omitempty"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	Comment        string    `json:"comment,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type SetActiveRepositoryRequest struct {
	RepositoryName string `json:"repository_name"`
	RepositoryPath string `json:"repository_path"`
	Source         string `json:"source"`
}

type ActiveRepositoryResponse struct {
	RepositoryName string    `json:"repository_name"`
	RepositoryPath string    `json:"repository_path"`
	Source         string    `json:"source"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SyncActivityResponse struct {
	LocalReposSynced           int `json:"local_repos_synced"`
	LocalCommitsAdded          int `json:"local_commits_added"`
	PlanningIntegrationsSynced int `json:"planning_integrations_synced"`
	PlanningActivitiesAdded    int `json:"planning_activities_added"`
}

type ActivitySummaryResponse struct {
	ReportDate         string                     `json:"report_date"`
	LocalCommits       []LocalGitCommitResponse   `json:"local_commits"`
	PlanningActivities []PlanningActivityResponse `json:"planning_activities"`
	ActiveRepository   *ActiveRepositoryResponse  `json:"active_repository,omitempty"`
}

type ListPlanningProvidersResponse struct {
	Providers []string `json:"providers"`
}
