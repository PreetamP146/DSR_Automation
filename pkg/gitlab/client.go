package gitlab

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
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	Description       string `json:"description"`
	DefaultBranch     string `json:"default_branch"`
}

type Client interface {
	VerifyToken(baseURL, accessToken string) (*User, error)
	ListMemberProjects(baseURL, accessToken string) ([]Project, error)
}

type client struct {
	httpClient *http.Client
}

func NewClient() Client {
	return &client{httpClient: http.DefaultClient}
}

func (c *client) VerifyToken(baseURL, accessToken string) (*User, error) {
	endpoint, err := buildURL(baseURL, "/api/v4/user")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", accessToken)

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
		return nil, fmt.Errorf("gitlab api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	if user.Username == "" {
		return nil, fmt.Errorf("invalid gitlab user response")
	}

	return &user, nil
}

func (c *client) ListMemberProjects(baseURL, accessToken string) ([]Project, error) {
	var all []Project
	page := 1

	for {
		query := url.Values{}
		query.Set("membership", "true")
		query.Set("simple", "true")
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))

		endpoint, err := buildURL(baseURL, "/api/v4/projects?"+query.Encode())
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("PRIVATE-TOKEN", accessToken)

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
			return nil, fmt.Errorf("gitlab api error: status %d, body: %s", resp.StatusCode, string(body))
		}

		var batch []Project
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
