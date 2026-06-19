package jira

import (
	"fmt"
	"log"
	"strings"
	"time"

	jiraclient "dsr-automation/pkg/jira"
	"dsr-automation/pkg/planning"
)

type Provider struct {
	client jiraclient.Client
}

func NewProvider(client jiraclient.Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) Name() string {
	return planning.ProviderJira
}

func (p *Provider) VerifyCredentials(credentials planning.Credentials) (*planning.Account, error) {
	user, err := p.client.VerifyCredentials(credentials.BaseURL, credentials.Email, credentials.APIToken)
	if err != nil {
		return nil, err
	}
	return &planning.Account{
		AccountID:   user.AccountID,
		DisplayName: user.DisplayName,
	}, nil
}

func (p *Provider) FetchActivities(credentials planning.Credentials, account planning.Account, since time.Time) ([]planning.ActivityEvent, error) {
	jql := fmt.Sprintf(
		`assignee = currentUser() AND updated >= "%s" ORDER BY updated DESC`,
		since.Format("2006/01/02 15:04"),
	)

	var events []planning.ActivityEvent
	startAt := 0
	const pageSize = 50

	for {
		result, err := p.client.SearchAssignedIssues(
			credentials.BaseURL,
			credentials.Email,
			credentials.APIToken,
			jql,
			startAt,
			pageSize,
		)
		if err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			events = append(events, extractChangelogEvents(account.AccountID, issue, since)...)
			commentEvents, err := p.extractCommentEvents(credentials, account, issue, since)
			if err != nil {
				log.Printf("jira comment fetch failed for %s: %v", issue.Key, err)
				continue
			}
			events = append(events, commentEvents...)
		}

		startAt += len(result.Issues)
		if startAt >= result.Total || len(result.Issues) == 0 {
			break
		}
	}

	return events, nil
}

func extractChangelogEvents(accountID string, issue jiraclient.Issue, since time.Time) []planning.ActivityEvent {
	if issue.Changelog == nil {
		return nil
	}

	var events []planning.ActivityEvent
	for _, history := range issue.Changelog.Histories {
		occurredAt, err := jiraclient.ParseJiraTime(history.Created)
		if err != nil || occurredAt.Before(since) {
			continue
		}

		for _, item := range history.Items {
			switch item.Field {
			case "assignee":
				if item.To != accountID {
					continue
				}
				events = append(events, planning.ActivityEvent{
					ActivityType: planning.ActivityAssigned,
					ItemKey:      issue.Key,
					ItemTitle:    issue.Fields.Summary,
					Status:       issue.Fields.Status.Name,
					ExternalID:   fmt.Sprintf("jira:%s:assignee:%s", issue.Key, history.ID),
					OccurredAt:   occurredAt,
				})
			case "status":
				activityType := planning.ActivityStatusChange
				if isDoneStatus(item.ToString) {
					activityType = planning.ActivityCompleted
				}
				events = append(events, planning.ActivityEvent{
					ActivityType:   activityType,
					ItemKey:        issue.Key,
					ItemTitle:      issue.Fields.Summary,
					Status:         item.ToString,
					PreviousStatus: item.FromString,
					ExternalID:     fmt.Sprintf("jira:%s:status:%s", issue.Key, history.ID),
					OccurredAt:     occurredAt,
				})
			}
		}
	}

	return events
}

func (p *Provider) extractCommentEvents(credentials planning.Credentials, account planning.Account, issue jiraclient.Issue, since time.Time) ([]planning.ActivityEvent, error) {
	comments, err := p.client.ListComments(credentials.BaseURL, credentials.Email, credentials.APIToken, issue.Key)
	if err != nil {
		return nil, err
	}

	var events []planning.ActivityEvent
	for _, comment := range comments {
		if comment.Author.AccountID != account.AccountID {
			continue
		}

		occurredAt, err := jiraclient.ParseJiraTime(comment.Created)
		if err != nil || occurredAt.Before(since) {
			continue
		}

		events = append(events, planning.ActivityEvent{
			ActivityType: planning.ActivityCommentAdded,
			ItemKey:      issue.Key,
			ItemTitle:    issue.Fields.Summary,
			Status:       issue.Fields.Status.Name,
			Comment:      truncate(comment.Body, 500),
			ExternalID:   fmt.Sprintf("jira:%s:comment:%s", issue.Key, comment.ID),
			OccurredAt:   occurredAt,
		})
	}

	return events, nil
}

func isDoneStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	for _, candidate := range []string{"done", "closed", "resolved", "complete", "completed"} {
		if normalized == candidate {
			return true
		}
	}
	return false
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
