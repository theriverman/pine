package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadRuntime() (*Runtime, error) {
	configPath, err := configPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	cfg := &Config{
		Version:   1,
		Instances: map[string]*Instance{},
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Instances == nil {
		cfg.Instances = map[string]*Instance{}
	}

	return &Runtime{
		ConfigPath: configPath,
		Config:     cfg,
		Secrets:    NewSecretStore(),
	}, nil
}

func configPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(envConfigDir)); override != "" {
		return filepath.Join(override, configFileName), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, appName, configFileName), nil
}

func (rt *Runtime) save() error {
	rt.Config.Version = 1
	data, err := yaml.Marshal(rt.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(rt.ConfigPath, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (rt *Runtime) listInstances() []*Instance {
	instances := make([]*Instance, 0, len(rt.Config.Instances))
	for _, instance := range rt.Config.Instances {
		instances = append(instances, instance)
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Alias < instances[j].Alias
	})
	return instances
}

func (rt *Runtime) getInstance(alias string) (*Instance, error) {
	if alias == "" {
		return nil, errors.New("instance alias is required")
	}
	instance, ok := rt.Config.Instances[alias]
	if !ok {
		return nil, fmt.Errorf("instance %q not found", alias)
	}
	return instance, nil
}

func (rt *Runtime) resolveInstanceAlias(requested string) (string, error) {
	if requested != "" {
		if _, ok := rt.Config.Instances[requested]; !ok {
			return "", fmt.Errorf("instance %q not found", requested)
		}
		return requested, nil
	}
	if envAlias := strings.TrimSpace(os.Getenv(envInstance)); envAlias != "" {
		if _, ok := rt.Config.Instances[envAlias]; ok {
			return envAlias, nil
		}
		return "", fmt.Errorf("instance %q from %s not found", envAlias, envInstance)
	}
	if rt.Config.CurrentInstance == "" {
		return "", errors.New("no current instance selected")
	}
	if _, ok := rt.Config.Instances[rt.Config.CurrentInstance]; !ok {
		return "", fmt.Errorf("current instance %q not found", rt.Config.CurrentInstance)
	}
	return rt.Config.CurrentInstance, nil
}

func (rt *Runtime) upsertInstance(instance *Instance) {
	if rt.Config.Instances == nil {
		rt.Config.Instances = map[string]*Instance{}
	}
	rt.Config.Instances[instance.Alias] = instance
}

func (rt *Runtime) deleteInstance(alias string) {
	delete(rt.Config.Instances, alias)
	if rt.Config.CurrentInstance == alias {
		rt.Config.CurrentInstance = ""
	}
}

func (rt *Runtime) upsertProject(instance *Instance, project *SavedProject) {
	for i, existing := range instance.SavedProjects {
		if existing.Slug == project.Slug {
			instance.SavedProjects[i] = project
			return
		}
	}
	instance.SavedProjects = append(instance.SavedProjects, project)
	sort.Slice(instance.SavedProjects, func(i, j int) bool {
		return instance.SavedProjects[i].Slug < instance.SavedProjects[j].Slug
	})
}

func (rt *Runtime) deleteProject(instance *Instance, slug string) {
	filtered := make([]*SavedProject, 0, len(instance.SavedProjects))
	for _, project := range instance.SavedProjects {
		if project.Slug != slug {
			filtered = append(filtered, project)
		}
	}
	instance.SavedProjects = filtered
	if instance.DefaultProject != nil && instance.DefaultProject.Slug == slug {
		instance.DefaultProject = nil
	}
}

func (rt *Runtime) findSavedProject(instance *Instance, slug string) *SavedProject {
	for _, project := range instance.SavedProjects {
		if project.Slug == slug {
			return project
		}
	}
	return nil
}
