package dto

type ConnectGitRequest struct {
	Provider    string `json:"provider"`
	BaseURL     string `json:"base_url"`
	AccessToken string `json:"access_token"`
}

type GitProjectResponse struct {
	ID                string `json:"id"`
	GitProjectID      string `json:"git_project_id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	Description       string `json:"description"`
	DefaultBranch     string `json:"default_branch"`
}

type ConnectGitResponse struct {
	ID             string               `json:"id"`
	Provider       string               `json:"provider"`
	BaseURL        string               `json:"base_url"`
	Username       string               `json:"username"`
	ProjectsSynced int                  `json:"projects_synced"`
	Projects       []GitProjectResponse `json:"projects"`
}
