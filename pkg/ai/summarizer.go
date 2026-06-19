package ai

import (
	"bytes"
	"context"
	"dsr-automation/pkg/activity"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	ModelTemplate = "template"
	PromptVersion = "v3"
)

type DSRContent struct {
	YesterdayWork string
	TodayPlan     string
	Blockers      string
	Summary       string
	Model         string
}

type Summarizer interface {
	GenerateDSR(ctx context.Context, reportDate time.Time, data activity.Summary) (*DSRContent, error)
}

type Config struct {
	OpenAIAPIKey string
	OpenAIModel  string
}

func NewSummarizer(cfg Config) Summarizer {
	template := &templateSummarizer{}
	if strings.TrimSpace(cfg.OpenAIAPIKey) == "" {
		return template
	}

	model := strings.TrimSpace(cfg.OpenAIModel)
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &fallbackSummarizer{
		primary: &openAISummarizer{
			apiKey: strings.TrimSpace(cfg.OpenAIAPIKey),
			model:  model,
			client: http.DefaultClient,
		},
		fallback: template,
	}
}

type fallbackSummarizer struct {
	primary  Summarizer
	fallback Summarizer
}

func (s *fallbackSummarizer) GenerateDSR(ctx context.Context, reportDate time.Time, data activity.Summary) (*DSRContent, error) {
	content, err := s.primary.GenerateDSR(ctx, reportDate, data)
	if err == nil {
		return content, nil
	}

	log.Printf("ai summarizer: primary failed (%v), using template fallback", err)
	return s.fallback.GenerateDSR(ctx, reportDate, data)
}

type templateSummarizer struct{}

func (s *templateSummarizer) GenerateDSR(_ context.Context, reportDate time.Time, data activity.Summary) (*DSRContent, error) {
	var yesterdayLines []string

	for _, project := range data.RemoteGit {
		if len(project.Commits) == 0 {
			continue
		}
		yesterdayLines = append(yesterdayLines, fmt.Sprintf("%s (%s):", project.ProjectName, project.Provider))
		for _, commit := range project.Commits {
			firstLine := strings.Split(commit.Message, "\n")[0]
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- [remote] %s (%s)", firstLine, shortSHA(commit.SHA)))
		}
	}

	commitsByRepo := make(map[string][]activity.LocalCommit)
	for _, commit := range data.LocalCommits {
		commitsByRepo[commit.RepositoryName] = append(commitsByRepo[commit.RepositoryName], commit)
	}
	for repoName, commits := range commitsByRepo {
		yesterdayLines = append(yesterdayLines, fmt.Sprintf("%s (local):", repoName))
		for _, commit := range commits {
			firstLine := strings.Split(commit.Message, "\n")[0]
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- [%s] %s (%s)", commit.Branch, firstLine, shortSHA(commit.SHA)))
		}
	}

	for _, item := range data.PlanningActivities {
		prefix := fmt.Sprintf("[%s] ", item.Provider)
		switch item.ActivityType {
		case "assigned":
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- %sAssigned to %s: %s", prefix, item.ItemKey, item.ItemTitle))
		case "status_change":
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- %s%s moved %s → %s (%s)", prefix, item.ItemKey, item.PreviousStatus, item.Status, item.ItemTitle))
		case "comment_added":
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- %sCommented on %s: %s", prefix, item.ItemKey, truncate(item.Comment, 120)))
		case "completed":
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- %sCompleted %s: %s", prefix, item.ItemKey, item.ItemTitle))
		default:
			yesterdayLines = append(yesterdayLines, fmt.Sprintf("- %s%s: %s", prefix, item.ItemKey, item.ItemTitle))
		}
	}

	yesterdayWork := "No tracked activity found for this date."
	if len(yesterdayLines) > 0 {
		yesterdayWork = strings.Join(yesterdayLines, "\n")
	}

	todayPlan := "Continue ongoing development tasks."
	if data.ActiveRepository != nil {
		todayPlan = fmt.Sprintf("Continue work on %s.", data.ActiveRepository.RepositoryName)
	}

	summary := fmt.Sprintf("No tracked activity recorded on %s.", reportDate.Format("2006-01-02"))
	if data.HasAnyActivity() {
		summary = fmt.Sprintf(
			"Recorded %d remote commit(s), %d local commit(s), and %d planning tool event(s) on %s.",
			data.TotalRemoteCommits(),
			len(data.LocalCommits),
			len(data.PlanningActivities),
			reportDate.Format("2006-01-02"),
		)
	}

	return &DSRContent{
		YesterdayWork: yesterdayWork,
		TodayPlan:     todayPlan,
		Blockers:      "None reported.",
		Summary:       summary,
		Model:         ModelTemplate,
	}, nil
}

type openAISummarizer struct {
	apiKey string
	model  string
	client *http.Client
}

func (s *openAISummarizer) GenerateDSR(ctx context.Context, reportDate time.Time, data activity.Summary) (*DSRContent, error) {
	payload, err := json.Marshal(map[string]any{
		"model": s.model,
		"response_format": map[string]string{
			"type": "json_object",
		},
		"messages": []map[string]string{
			{
				"role": "system",
				"content": "You generate concise daily status reports from developer activity. " +
					"Input may include remote git commits (GitHub/GitLab), local git commits, planning tool events " +
					"(Jira, Trello, or other task trackers), and the developer's currently active repository. " +
					"Synthesize all sources into a cohesive narrative — do not ignore any section with data. " +
					"Return JSON with keys: yesterday_work, today_plan, blockers, summary. " +
					"Use complete sentences and bullet points where helpful.",
			},
			{
				"role":    "user",
				"content": buildPrompt(reportDate, data),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return nil, err
	}
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	var content struct {
		YesterdayWork string `json:"yesterday_work"`
		TodayPlan     string `json:"today_plan"`
		Blockers      string `json:"blockers"`
		Summary       string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &content); err != nil {
		return nil, err
	}

	return &DSRContent{
		YesterdayWork: strings.TrimSpace(content.YesterdayWork),
		TodayPlan:     strings.TrimSpace(content.TodayPlan),
		Blockers:      strings.TrimSpace(content.Blockers),
		Summary:       strings.TrimSpace(content.Summary),
		Model:         s.model,
	}, nil
}

func buildPrompt(reportDate time.Time, data activity.Summary) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Generate a DSR for work done on %s.\n\n", reportDate.Format("2006-01-02")))

	builder.WriteString("## Remote Git (GitHub/GitLab)\n")
	hasRemote := false
	for _, project := range data.RemoteGit {
		if len(project.Commits) == 0 {
			continue
		}
		hasRemote = true
		builder.WriteString(fmt.Sprintf("Project: %s (%s, %s)\n", project.ProjectName, project.PathWithNamespace, project.Provider))
		for _, commit := range project.Commits {
			builder.WriteString(fmt.Sprintf("- [%s] %s by %s\n", shortSHA(commit.SHA), commit.Message, commit.Author))
		}
		builder.WriteString("\n")
	}
	if !hasRemote {
		builder.WriteString("None\n\n")
	}

	builder.WriteString("## Local Git Commits\n")
	if len(data.LocalCommits) == 0 {
		builder.WriteString("None\n\n")
	} else {
		for _, commit := range data.LocalCommits {
			builder.WriteString(fmt.Sprintf(
				"- [%s] %s on branch %s in %s by %s\n",
				shortSHA(commit.SHA), commit.Message, commit.Branch, commit.RepositoryName, commit.Author,
			))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Planning Tool Activity (Jira, Trello, etc.)\n")
	if len(data.PlanningActivities) == 0 {
		builder.WriteString("None\n\n")
	} else {
		for _, item := range data.PlanningActivities {
			builder.WriteString(fmt.Sprintf("- [%s/%s] %s %s: %s", item.Provider, item.ActivityType, item.ItemKey, item.ItemTitle, item.Status))
			if item.PreviousStatus != "" {
				builder.WriteString(fmt.Sprintf(" (from %s)", item.PreviousStatus))
			}
			if item.Comment != "" {
				builder.WriteString(fmt.Sprintf(" — comment: %s", truncate(item.Comment, 200)))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Active Repository\n")
	if data.ActiveRepository == nil {
		builder.WriteString("Not reported\n")
	} else {
		builder.WriteString(fmt.Sprintf(
			"%s at %s (source: %s)\n",
			data.ActiveRepository.RepositoryName,
			data.ActiveRepository.RepositoryPath,
			data.ActiveRepository.Source,
		))
	}

	return builder.String()
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
