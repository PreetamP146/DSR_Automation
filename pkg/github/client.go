package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type User struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

type Repo struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
}

type Client interface {
	VerifyToken(baseURL, accessToken string) (*User, error)
	ListMemberRepos(baseURL, accessToken string) ([]Repo, error)
	ListCommits(baseURL, accessToken, owner, repo, author string, since, until time.Time) ([]Commit, error)
}

type client struct {
	httpClient *http.Client
}

func NewClient() Client {
	return &client{httpClient: http.DefaultClient}
}

func githubAuthHeaders(accessToken string) []string {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil
	}

	// Classic PATs (ghp_) work with the legacy "token" scheme; fine-grained/OAuth use Bearer.
	if strings.HasPrefix(token, "ghp_") || strings.HasPrefix(token, "ghu_") {
		return []string{"token " + token, "Bearer " + token}
	}

	return []string{"Bearer " + token, "token " + token}
}

func setGitHubAuth(req *http.Request, accessToken string) {
	headers := githubAuthHeaders(accessToken)
	if len(headers) > 0 {
		req.Header.Set("Authorization", headers[0])
	}
}

func (c *client) VerifyToken(baseURL, accessToken string) (*User, error) {
	endpoint, err := buildURL(baseURL, "/user")
	if err != nil {
		return nil, err
	}

	var lastStatus int
	for _, authHeader := range githubAuthHeaders(accessToken) {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			var user User
			decodeErr := json.NewDecoder(resp.Body).Decode(&user)
			resp.Body.Close()
			if decodeErr != nil {
				return nil, decodeErr
			}
			if user.Login == "" {
				return nil, fmt.Errorf("invalid github user response")
			}
			return &user, nil
		}

		lastStatus = resp.StatusCode
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			return nil, fmt.Errorf("github api error: status %d, body: %s", resp.StatusCode, string(body))
		}
	}

	if lastStatus == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid access token")
	}

	return nil, fmt.Errorf("github api error: status %d", lastStatus)
}

func (c *client) ListMemberRepos(baseURL, accessToken string) ([]Repo, error) {
	var all []Repo
	page := 1

	for {
		query := url.Values{}
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))
		query.Set("affiliation", "owner,collaborator,organization_member")

		endpoint, err := buildURL(baseURL, "/user/repos?"+query.Encode())
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		setGitHubAuth(req, accessToken)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, fmt.Errorf("invalid access token")
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("github api error: status %d, body: %s", resp.StatusCode, string(body))
		}

		var batch []Repo
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}

	return all, nil
}

type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
	URL     string
}

func (c *client) ListCommits(baseURL, accessToken, owner, repo, author string, since, until time.Time) ([]Commit, error) {
	var all []Commit
	page := 1

	for {
		query := url.Values{}
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))
		if !since.IsZero() {
			query.Set("since", since.UTC().Format(time.RFC3339))
		}
		if !until.IsZero() {
			query.Set("until", until.UTC().Format(time.RFC3339))
		}
		if author != "" {
			query.Set("author", author)
		}

		path := fmt.Sprintf("/repos/%s/%s/commits?%s", url.PathEscape(owner), url.PathEscape(repo), query.Encode())
		endpoint, err := buildURL(baseURL, path)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		setGitHubAuth(req, accessToken)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, fmt.Errorf("invalid access token")
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("github api error: status %d, body: %s", resp.StatusCode, string(body))
		}

		var batch []struct {
			SHA     string `json:"sha"`
			HTMLURL string `json:"html_url"`
			Commit  struct {
				Message string `json:"message"`
				Author  struct {
					Name  string    `json:"name"`
					Date  time.Time `json:"date"`
					Email string    `json:"email"`
				} `json:"author"`
			} `json:"commit"`
			Author *struct {
				Login string `json:"login"`
			} `json:"author"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			commitAuthor := item.Commit.Author.Name
			if item.Author != nil && item.Author.Login != "" {
				commitAuthor = item.Author.Login
			}
			all = append(all, Commit{
				SHA:     item.SHA,
				Message: strings.TrimSpace(item.Commit.Message),
				Author:  commitAuthor,
				Date:    item.Commit.Author.Date,
				URL:     item.HTMLURL,
			})
		}

		if len(batch) < 100 {
			break
		}
		page++
	}

	return all, nil
}

func buildURL(baseURL, path string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimSuffix(baseURL, "/")
	if baseURL == "" {
		return "", fmt.Errorf("base url is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base url")
	}

	return baseURL + path, nil
}
