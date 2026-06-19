package jira

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	VerifyCredentials(baseURL, email, apiToken string) (*User, error)
	SearchAssignedIssues(baseURL, email, apiToken, jql string, startAt, maxResults int) (*SearchResult, error)
	ListComments(baseURL, email, apiToken, issueKey string) ([]Comment, error)
}

type client struct {
	httpClient *http.Client
}

func NewClient() Client {
	return &client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress"`
}

type SearchResult struct {
	Total  int     `json:"total"`
	Issues []Issue `json:"issues"`
}

type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
	Changelog *Changelog `json:"changelog,omitempty"`
}

type IssueFields struct {
	Summary string     `json:"summary"`
	Status  StatusField `json:"status"`
}

type StatusField struct {
	Name string `json:"name"`
}

type Changelog struct {
	Histories []History `json:"histories"`
}

type History struct {
	ID      string `json:"id"`
	Author  Author `json:"author"`
	Created string `json:"created"`
	Items   []Item `json:"items"`
}

type Item struct {
	Field      string `json:"field"`
	From       string `json:"from"`
	FromString string `json:"fromString"`
	To         string `json:"to"`
	ToString   string `json:"toString"`
}

type Comment struct {
	ID      string `json:"id"`
	Author  Author `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
}

type Author struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

func (c *client) VerifyCredentials(baseURL, email, apiToken string) (*User, error) {
	var user User
	if err := c.getJSON(baseURL, email, apiToken, "/rest/api/3/myself", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *client) SearchAssignedIssues(baseURL, email, apiToken, jql string, startAt, maxResults int) (*SearchResult, error) {
	body := map[string]interface{}{
		"jql":        jql,
		"startAt":    startAt,
		"maxResults": maxResults,
		"fields":     []string{"summary", "status", "assignee"},
		"expand":     []string{"changelog"},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := c.postJSON(baseURL, email, apiToken, "/rest/api/3/search", payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *client) ListComments(baseURL, email, apiToken, issueKey string) ([]Comment, error) {
	var response struct {
		Comments []Comment `json:"comments"`
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)
	if err := c.getJSON(baseURL, email, apiToken, path, &response); err != nil {
		return nil, err
	}
	return response.Comments, nil
}

func (c *client) getJSON(baseURL, email, apiToken, path string, dest interface{}) error {
	url := normalizeBaseURL(baseURL) + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", basicAuth(email, apiToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeResponse(resp, dest)
}

func (c *client) postJSON(baseURL, email, apiToken, path string, body []byte, dest interface{}) error {
	url := normalizeBaseURL(baseURL) + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", basicAuth(email, apiToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeResponse(resp, dest)
}

func decodeResponse(resp *http.Response, dest interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jira api error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(body, dest)
}

func basicAuth(email, apiToken string) string {
	credentials := email + ":" + apiToken
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

func normalizeBaseURL(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), "/")
}
