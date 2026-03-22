package main

import (
	"time"

	taigo "github.com/theriverman/taigo/v2"
)

type contextKey string

const runtimeContextKey contextKey = "pine.runtime"

const (
	appName            = "pine"
	configFileName     = "config.yaml"
	keyringService     = "pine"
	envInstance        = "PINE_INSTANCE"
	envAuthType        = "PINE_AUTH_TYPE"
	envUsername        = "PINE_USERNAME"
	envPassword        = "PINE_PASSWORD"
	envToken           = "PINE_TOKEN"
	envConfigDir       = "PINE_CONFIG_DIR"
	defaultHTTPTimeout = 15 * time.Second
)

type Config struct {
	Version         int                  `yaml:"version" json:"version"`
	CurrentInstance string               `yaml:"current_instance,omitempty" json:"current_instance,omitempty"`
	Instances       map[string]*Instance `yaml:"instances,omitempty" json:"instances,omitempty"`
}

type Instance struct {
	Alias          string          `yaml:"alias" json:"alias"`
	FrontendURL    string          `yaml:"frontend_url" json:"frontend_url"`
	APIURL         string          `yaml:"api_url" json:"api_url"`
	BaseURL        string          `yaml:"base_url" json:"base_url"`
	APIVersion     string          `yaml:"api_version" json:"api_version"`
	AuthType       string          `yaml:"auth_type" json:"auth_type"`
	Username       string          `yaml:"username,omitempty" json:"username,omitempty"`
	DefaultProject *SavedProject   `yaml:"default_project,omitempty" json:"default_project,omitempty"`
	SavedProjects  []*SavedProject `yaml:"saved_projects,omitempty" json:"saved_projects,omitempty"`
}

type SavedProject struct {
	ID   int    `yaml:"id" json:"id"`
	Slug string `yaml:"slug" json:"slug"`
	Name string `yaml:"name" json:"name"`
}

type Runtime struct {
	ConfigPath string
	Config     *Config
	Secrets    SecretStore
}

type Secret struct {
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

type Credentials struct {
	AuthType string
	Username string
	Password string
	Token    string
}

type Session struct {
	Alias    string
	Instance *Instance
	Client   *taigo.Client
}

type PaginationEnvelope struct {
	Items      any            `json:"items" yaml:"items"`
	Pagination PaginationView `json:"pagination,omitempty" yaml:"pagination,omitempty"`
	Extra      map[string]any `json:"extra,omitempty" yaml:"extra,omitempty"`
}

type PaginationView struct {
	Paginated bool   `json:"paginated" yaml:"paginated"`
	Page      int    `json:"page,omitempty" yaml:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty" yaml:"page_size,omitempty"`
	Count     int    `json:"count,omitempty" yaml:"count,omitempty"`
	Next      string `json:"next,omitempty" yaml:"next,omitempty"`
	Prev      string `json:"prev,omitempty" yaml:"prev,omitempty"`
}
