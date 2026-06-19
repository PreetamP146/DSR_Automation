package planning

import "time"

const (
	ProviderJira   = "jira"
	ProviderTrello = "trello"
	ProviderPlanit = "planit"
)

const (
	ActivityAssigned     = "assigned"
	ActivityStatusChange = "status_change"
	ActivityCommentAdded = "comment_added"
	ActivityCompleted    = "completed"
)

type Credentials struct {
	BaseURL   string
	Email     string
	APIToken  string
	APIKey    string
	AccountID string
}

type Account struct {
	AccountID   string
	DisplayName string
}

type ActivityEvent struct {
	ActivityType   string
	ItemKey        string
	ItemTitle      string
	Status         string
	PreviousStatus string
	Comment        string
	ExternalID     string
	OccurredAt     time.Time
}

type Provider interface {
	Name() string
	VerifyCredentials(credentials Credentials) (*Account, error)
	FetchActivities(credentials Credentials, account Account, since time.Time) ([]ActivityEvent, error)
}

func SupportedProviders() []string {
	return []string{ProviderJira, ProviderTrello}
}
