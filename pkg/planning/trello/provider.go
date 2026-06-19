package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dsr-automation/pkg/planning"
)

const baseURL = "https://api.trello.com/1"

type Provider struct {
	httpClient *http.Client
}

func NewProvider() *Provider {
	return &Provider{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (p *Provider) Name() string {
	return planning.ProviderTrello
}

func (p *Provider) VerifyCredentials(credentials planning.Credentials) (*planning.Account, error) {
	var member struct {
		ID       string `json:"id"`
		FullName string `json:"fullName"`
		Username string `json:"username"`
	}
	if err := p.get(credentials, "/members/me", &member); err != nil {
		return nil, err
	}

	displayName := member.FullName
	if displayName == "" {
		displayName = member.Username
	}

	return &planning.Account{
		AccountID:   member.ID,
		DisplayName: displayName,
	}, nil
}

func (p *Provider) FetchActivities(credentials planning.Credentials, account planning.Account, since time.Time) ([]planning.ActivityEvent, error) {
	query := url.Values{}
	query.Set("filter", "commentCard,updateCard,addMemberToCard")
	query.Set("since", since.UTC().Format(time.RFC3339))
	query.Set("limit", "1000")

	var actions []trelloAction
	if err := p.get(credentials, "/members/me/actions?"+query.Encode(), &actions); err != nil {
		return nil, err
	}

	events := make([]planning.ActivityEvent, 0, len(actions))
	for _, action := range actions {
		occurredAt, err := time.Parse(time.RFC3339, action.Date)
		if err != nil {
			continue
		}
		if occurredAt.Before(since) {
			continue
		}

		event, ok := mapAction(account.AccountID, action, occurredAt.UTC())
		if ok {
			events = append(events, event)
		}
	}

	return events, nil
}

type trelloAction struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Date          string `json:"date"`
	MemberCreator struct {
		ID string `json:"id"`
	} `json:"memberCreator"`
	Data struct {
		Text       string `json:"text"`
		Card       struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			ShortLink string `json:"shortLink"`
		} `json:"card"`
		ListBefore struct {
			Name string `json:"name"`
		} `json:"listBefore"`
		ListAfter struct {
			Name string `json:"name"`
		} `json:"listAfter"`
		Member struct {
			ID string `json:"id"`
		} `json:"member"`
	} `json:"data"`
}

func mapAction(accountID string, action trelloAction, occurredAt time.Time) (planning.ActivityEvent, bool) {
	itemKey := action.Data.Card.ShortLink
	if itemKey == "" {
		itemKey = action.Data.Card.ID
	}
	if itemKey == "" {
		return planning.ActivityEvent{}, false
	}

	itemTitle := action.Data.Card.Name

	switch action.Type {
	case "addMemberToCard":
		if action.Data.Member.ID != accountID {
			return planning.ActivityEvent{}, false
		}
		return planning.ActivityEvent{
			ActivityType: planning.ActivityAssigned,
			ItemKey:      itemKey,
			ItemTitle:    itemTitle,
			ExternalID:   fmt.Sprintf("trello:%s", action.ID),
			OccurredAt:   occurredAt,
		}, true

	case "updateCard":
		if action.MemberCreator.ID != accountID {
			return planning.ActivityEvent{}, false
		}
		if action.Data.ListBefore.Name == "" && action.Data.ListAfter.Name == "" {
			return planning.ActivityEvent{}, false
		}
		activityType := planning.ActivityStatusChange
		if isDoneList(action.Data.ListAfter.Name) {
			activityType = planning.ActivityCompleted
		}
		return planning.ActivityEvent{
			ActivityType:   activityType,
			ItemKey:        itemKey,
			ItemTitle:      itemTitle,
			Status:         action.Data.ListAfter.Name,
			PreviousStatus: action.Data.ListBefore.Name,
			ExternalID:     fmt.Sprintf("trello:%s", action.ID),
			OccurredAt:     occurredAt,
		}, true

	case "commentCard":
		if action.MemberCreator.ID != accountID {
			return planning.ActivityEvent{}, false
		}
		return planning.ActivityEvent{
			ActivityType: planning.ActivityCommentAdded,
			ItemKey:      itemKey,
			ItemTitle:    itemTitle,
			Comment:      truncate(action.Data.Text, 500),
			ExternalID:   fmt.Sprintf("trello:%s", action.ID),
			OccurredAt:   occurredAt,
		}, true

	default:
		return planning.ActivityEvent{}, false
	}
}

func isDoneList(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range []string{"done", "complete", "completed", "closed"} {
		if normalized == candidate {
			return true
		}
	}
	return false
}

func (p *Provider) get(credentials planning.Credentials, path string, dest interface{}) error {
	reqURL := baseURL + path
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	reqURL += separator + "key=" + url.QueryEscape(credentials.APIKey) + "&token=" + url.QueryEscape(credentials.APIToken)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("trello api error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.Unmarshal(body, dest)
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
