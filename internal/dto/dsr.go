package dto

import (
	"dsr-automation/pkg/activity"
	"time"
)

type GenerateDSRRequest struct {
	ReportDate    string   `json:"report_date"`
	GitProjectIDs []string `json:"git_project_ids"`
}

type DSRReportResponse struct {
	ID             string    `json:"id"`
	ReportDate     string    `json:"report_date"`
	GitProjectID   *string   `json:"git_project_id,omitempty"`
	GitProjectName *string   `json:"git_project_name,omitempty"`
	YesterdayWork  string    `json:"yesterday_work"`
	TodayPlan      string    `json:"today_plan"`
	Blockers       string    `json:"blockers"`
	Summary        string    `json:"summary"`
	AIModel        string    `json:"ai_model"`
	PromptVersion  string    `json:"prompt_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GenerateDSRResponse struct {
	Report   DSRReportResponse `json:"report"`
	Activity activity.Summary  `json:"activity"`
}

type ListDSRActivityResponse struct {
	ReportDate       string           `json:"report_date"`
	Activity         activity.Summary `json:"activity"`
	TotalRemoteCommits int            `json:"total_remote_commits"`
	TotalLocalCommits  int            `json:"total_local_commits"`
	TotalPlanningEvents  int              `json:"total_planning_events"`
}

type ListDSRReportsResponse struct {
	Reports []DSRReportResponse `json:"reports"`
	Total   int                 `json:"total"`
}
