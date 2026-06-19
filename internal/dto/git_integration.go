package dto

type ConnectGitRequest struct {
	Provider    string `json:"provider"`
	BaseURL     string `json:"base_url"`
	AccessToken string `json:"access_token"`
}

type SyncGitRequest struct {
	Provider string `json:"provider"`
}

type UpdateTrackedProjectsRequest struct {
	ProjectIDs []string `json:"project_ids"`
}

type GitProjectResponse struct {
	ID                string `json:"id"`
	GitIntegrationID  string `json:"git_integration_id"`
	Provider          string `json:"provider"`
	GitProjectID      string `json:"git_project_id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	Description       string `json:"description"`
	DefaultBranch     string `json:"default_branch"`
	IsTracked         bool   `json:"is_tracked"`
}

type GitIntegrationResponse struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	BaseURL        string `json:"base_url"`
	Username       string `json:"username"`
	ProjectsSynced int    `json:"projects_synced"`
	TrackedCount   int    `json:"tracked_count"`
}

type ConnectGitResponse struct {
	ID             string               `json:"id"`
	Provider       string               `json:"provider"`
	BaseURL        string               `json:"base_url"`
	Username       string               `json:"username"`
	ProjectsSynced int                  `json:"projects_synced"`
	Projects       []GitProjectResponse `json:"projects"`
}

type SyncGitResponse struct {
	ID             string               `json:"id"`
	Provider       string               `json:"provider"`
	BaseURL        string               `json:"base_url"`
	Username       string               `json:"username"`
	ProjectsSynced int                  `json:"projects_synced"`
	Projects       []GitProjectResponse `json:"projects"`
}

type ListGitProjectsResponse struct {
	Projects []GitProjectResponse `json:"projects"`
	Total    int                  `json:"total"`
}

type ListGitIntegrationsResponse struct {
	Integrations []GitIntegrationResponse `json:"integrations"`
}

type UpdateTrackedProjectsResponse struct {
	TrackedCount int                  `json:"tracked_count"`
	Projects     []GitProjectResponse `json:"projects"`
}

type SyncCommitsResponse struct {
	ProjectsSynced int `json:"projects_synced"`
	CommitsAdded   int `json:"commits_added"`
}
