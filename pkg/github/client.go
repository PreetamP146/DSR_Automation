package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
}

type client struct {
	httpClient *http.Client
}

func NewClient() Client {
	return &client{httpClient: http.DefaultClient}
}

func (c *client) VerifyToken(baseURL, accessToken string) (*User, error) {
	endpoint, err := buildURL(baseURL, "/user")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid access token")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	if user.Login == "" {
		return nil, fmt.Errorf("invalid github user response")
	}
	return &user, nil
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
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/vnd.github+json")

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
