package taigainstance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var apiURLPattern = regexp.MustCompile(`^(?P<base>.*)/api/(?P<version>[^/]+)/?$`)

type confPayload struct {
	API string `json:"api"`
}

type Details struct {
	APIURL     string
	BaseURL    string
	APIVersion string
}

func Discover(frontendURL string, timeout time.Duration) (details Details, err error) {
	frontendURL, err = NormaliseURL(frontendURL)
	if err != nil {
		return Details{}, err
	}

	confURL := strings.TrimRight(frontendURL, "/") + "/conf.json"
	httpClient := &http.Client{Timeout: timeout}
	response, err := httpClient.Get(confURL)
	if err != nil {
		return Details{}, fmt.Errorf("fetch %s: %w", confURL, err)
	}
	defer func() {
		closeErr := response.Body.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close %s response body: %w", confURL, closeErr)
		}
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Details{}, fmt.Errorf("fetch %s: unexpected status %s", confURL, response.Status)
	}

	payload := &confPayload{}
	if err := json.NewDecoder(response.Body).Decode(payload); err != nil {
		return Details{}, fmt.Errorf("decode %s: %w", confURL, err)
	}
	if strings.TrimSpace(payload.API) == "" {
		return Details{}, errors.New("conf.json did not contain an api value")
	}

	baseURL, apiVersion, err := SplitAPIURL(payload.API)
	if err != nil {
		return Details{}, err
	}
	apiURL, err := NormaliseURL(payload.API)
	if err != nil {
		return Details{}, err
	}

	return Details{
		APIURL:     apiURL,
		BaseURL:    baseURL,
		APIVersion: apiVersion,
	}, nil
}

func NormaliseURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid url %q", raw)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

func SplitAPIURL(apiURL string) (string, string, error) {
	normalised, err := NormaliseURL(apiURL)
	if err != nil {
		return "", "", err
	}
	matches := apiURLPattern.FindStringSubmatch(normalised)
	if len(matches) == 0 {
		return "", "", fmt.Errorf("api url %q does not end with /api/<version>", apiURL)
	}
	baseIndex := apiURLPattern.SubexpIndex("base")
	versionIndex := apiURLPattern.SubexpIndex("version")
	return matches[baseIndex], matches[versionIndex], nil
}
