package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	taigo "github.com/theriverman/taigo/v2"
	"pine/internal/taigainstance"
)

func discoverInstance(frontendURL string) (apiURL, baseURL, apiVersion string, err error) {
	details, err := taigainstance.Discover(frontendURL, defaultHTTPTimeout)
	if err != nil {
		return "", "", "", err
	}
	return details.APIURL, details.BaseURL, details.APIVersion, nil
}

func (rt *Runtime) openSession(alias string) (*Session, error) {
	resolvedAlias, err := rt.resolveInstanceAlias(alias)
	if err != nil {
		return nil, err
	}

	instance, err := rt.getInstance(resolvedAlias)
	if err != nil {
		return nil, err
	}

	credentials, err := resolveCredentials(instance, rt.Secrets)
	if err != nil {
		return nil, err
	}

	client := &taigo.Client{
		BaseURL:    instance.BaseURL,
		APIversion: instance.APIVersion,
		HTTPClient: &http.Client{Timeout: defaultHTTPTimeout},
	}

	switch credentials.AuthType {
	case "token":
		if err := client.AuthByToken(taigo.TokenBearer, credentials.Token, ""); err != nil {
			return nil, err
		}
	case "normal", "ldap":
		if err := client.AuthByCredentials(&taigo.Credentials{
			Type:     credentials.AuthType,
			Username: credentials.Username,
			Password: credentials.Password,
		}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported auth type %q", credentials.AuthType)
	}

	client.DisablePagination(false)
	if instance.DefaultProject != nil && instance.DefaultProject.ID > 0 {
		client.Project.ConfigureMappedServices(instance.DefaultProject.ID)
	}

	return &Session{
		Alias:    resolvedAlias,
		Instance: instance,
		Client:   client,
	}, nil
}

func (s *Session) close() {
	if s != nil && s.Client != nil {
		s.Client.Close()
	}
}

func (s *Session) activeProjectID() int {
	if s == nil || s.Instance == nil || s.Instance.DefaultProject == nil {
		return 0
	}
	return s.Instance.DefaultProject.ID
}

func (s *Session) resolveProjectID(projectID int, projectSlug string) (int, error) {
	switch {
	case projectID > 0:
		return projectID, nil
	case strings.TrimSpace(projectSlug) != "":
		project, err := s.Client.Project.GetBySlug(projectSlug)
		if err != nil {
			return 0, err
		}
		return project.ID, nil
	case s.activeProjectID() > 0:
		return s.activeProjectID(), nil
	default:
		return 0, errors.New("a project is required and no default project is selected")
	}
}

func paginationFromClient(client *taigo.Client) PaginationView {
	pagination := client.GetPagination()
	view := PaginationView{
		Paginated: pagination.Paginated,
		Page:      pagination.PaginationCurrent,
		PageSize:  pagination.PaginatedBy,
		Count:     pagination.PaginationCount,
	}
	if pagination.PaginationNext != nil {
		view.Next = pagination.PaginationNext.String()
	}
	if pagination.PaginationPrev != nil {
		view.Prev = pagination.PaginationPrev.String()
	}
	return view
}

func requestURL(client *taigo.Client, endpoint string, values url.Values) string {
	return appendQuery(client.MakeURL(endpoint), values)
}
