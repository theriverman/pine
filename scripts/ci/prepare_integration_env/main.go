package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	taigo "github.com/theriverman/taigo/v2"
	"pine/internal/taigainstance"
)

type options struct {
	frontendURL        string
	username           string
	password           string
	projectName        string
	projectSlug        string
	projectDescription string
	githubEnvPath      string
	timeout            time.Duration
	pollInterval       time.Duration
}

func main() {
	opts := parseFlags()

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	project, err := waitForProject(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare integration env: %v\n", err)
		os.Exit(1)
	}

	lines := []string{
		"PINE_RUN_INTEGRATION=1",
		fmt.Sprintf("PINE_INTEGRATION_FRONTEND=%s", opts.frontendURL),
		fmt.Sprintf("PINE_INTEGRATION_USERNAME=%s", opts.username),
		fmt.Sprintf("PINE_INTEGRATION_PASSWORD=%s", opts.password),
		fmt.Sprintf("PINE_INTEGRATION_PROJECT_ID=%d", project.ID),
		fmt.Sprintf("PINE_INTEGRATION_PROJECT_SLUG=%s", project.Slug),
	}

	if err := emitLines(lines, opts.githubEnvPath); err != nil {
		fmt.Fprintf(os.Stderr, "write integration env: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() options {
	var opts options

	flag.StringVar(&opts.frontendURL, "frontend", "http://localhost:9000", "Taiga frontend URL")
	flag.StringVar(&opts.username, "username", "admin", "Taiga username")
	flag.StringVar(&opts.password, "password", "123123", "Taiga password")
	flag.StringVar(&opts.projectName, "project-name", "pine-ci", "Project name to create or reuse")
	flag.StringVar(&opts.projectSlug, "project-slug", "pine-ci", "Project slug to look up")
	flag.StringVar(&opts.projectDescription, "project-description", "Ephemeral Taiga project for pine integration tests", "Project description")
	flag.StringVar(&opts.githubEnvPath, "github-env", "", "Optional path to append GitHub Actions environment variables to")
	flag.DurationVar(&opts.timeout, "timeout", 10*time.Minute, "Overall wait timeout")
	flag.DurationVar(&opts.pollInterval, "poll-interval", 5*time.Second, "Retry interval while waiting for Taiga")
	flag.Parse()

	return opts
}

func waitForProject(ctx context.Context, opts options) (*taigo.Project, error) {
	var lastErr error

	for {
		project, err := prepareProject(opts)
		if err == nil {
			return project, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for Taiga at %s: %w", opts.frontendURL, lastErr)
		case <-time.After(opts.pollInterval):
		}
	}
}

func prepareProject(opts options) (*taigo.Project, error) {
	details, err := taigainstance.Discover(opts.frontendURL, 10*time.Second)
	if err != nil {
		return nil, err
	}

	client := &taigo.Client{
		BaseURL:    details.BaseURL,
		APIversion: details.APIVersion,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	defer client.Close()

	if err := client.AuthByCredentials(&taigo.Credentials{
		Type:     "normal",
		Username: opts.username,
		Password: opts.password,
	}); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	projectsList, err := client.Project.List(nil)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	projects, err := projectsList.AsProjects()
	if err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}
	for i := range projects {
		if projects[i].Slug == opts.projectSlug || projects[i].Name == opts.projectName {
			return &projects[i], nil
		}
	}

	project, err := client.Project.Create(&taigo.Project{
		Name:        opts.projectName,
		Description: opts.projectDescription,
	})
	if err != nil {
		return nil, fmt.Errorf("create project %q: %w", opts.projectName, err)
	}
	return project, nil
}

func emitLines(lines []string, githubEnvPath string) (err error) {
	if githubEnvPath == "" {
		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	}

	file, err := os.OpenFile(githubEnvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close %s: %w", githubEnvPath, closeErr)
		}
	}()

	for _, line := range lines {
		if _, err := fmt.Fprintln(file, line); err != nil {
			return err
		}
	}
	return nil
}
