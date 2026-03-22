package main

import (
	"context"
	"fmt"
	"strings"

	taigo "github.com/theriverman/taigo/v2"
	cli "github.com/urfave/cli/v3"
)

func newApp() (*cli.Command, error) {
	cli.VersionPrinter = printVersion

	commands := []*cli.Command{
		newContextCommand(),
		newMeCommand(),
	}

	for _, spec := range resourceSpecs() {
		commands = append(commands, resourceCommand(spec))
	}

	return &cli.Command{
		Name:                  appName,
		Usage:                 "A Taiga CLI built on Taigo v2",
		Version:               currentVersionInfo().Version,
		EnableShellCompletion: true,
		ConfigureShellCompletionCommand: func(command *cli.Command) {
			command.Hidden = false
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "instance", Usage: "Override the selected instance alias"},
			&cli.StringFlag{Name: "output", Usage: "Output format", Value: "json"},
		},
		Commands: commands,
	}, nil
}

func runtimeFromContext(ctx context.Context) (*Runtime, error) {
	if ctx != nil {
		if rt, _ := ctx.Value(runtimeContextKey).(*Runtime); rt != nil {
			return rt, nil
		}
	}
	return loadRuntime()
}

func resolveOutput(cmd *cli.Command) string {
	value := strings.TrimSpace(cmd.String("output"))
	if value == "" {
		return "json"
	}
	return value
}

func newContextCommand() *cli.Command {
	return &cli.Command{
		Name:  "ctx",
		Usage: "Manage saved Taiga instances and projects",
		Commands: []*cli.Command{
			{
				Name:  "show",
				Usage: "Show the current instance and project context",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rt, err := runtimeFromContext(ctx)
					if err != nil {
						return err
					}
					var currentInstance *Instance
					if rt.Config.CurrentInstance != "" {
						currentInstance = rt.Config.Instances[rt.Config.CurrentInstance]
					}
					var defaultProject *SavedProject
					if currentInstance != nil {
						defaultProject = currentInstance.DefaultProject
					}
					return render(resolveOutput(cmd), map[string]any{
						"current_instance": rt.Config.CurrentInstance,
						"instance":         currentInstance,
						"default_project":  defaultProject,
					})
				},
			},
			{
				Name:  "instance",
				Usage: "Manage saved instances",
				Commands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List saved instances",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							rt, err := runtimeFromContext(ctx)
							if err != nil {
								return err
							}
							summaries := make([]map[string]any, 0, len(rt.Config.Instances))
							for _, instance := range rt.listInstances() {
								summaries = append(summaries, map[string]any{
									"alias":           instance.Alias,
									"frontend_url":    instance.FrontendURL,
									"api_url":         instance.APIURL,
									"auth_type":       instance.AuthType,
									"username":        instance.Username,
									"is_current":      rt.Config.CurrentInstance == instance.Alias,
									"default_project": projectSlug(instance.DefaultProject),
								})
							}
							return render(resolveOutput(cmd), summaries)
						},
					},
					{
						Name:  "add",
						Usage: "Add and validate a Taiga instance",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextInstanceAdd(ctx, cmd)
						},
					},
					{
						Name:      "use",
						Usage:     "Select the current instance",
						ArgsUsage: "[alias]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextInstanceUse(ctx, cmd)
						},
					},
					{
						Name:      "remove",
						Usage:     "Remove a saved instance",
						ArgsUsage: "[alias]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextInstanceRemove(ctx, cmd)
						},
					},
				},
			},
			{
				Name:  "project",
				Usage: "Manage saved projects on the current instance",
				Commands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List saved projects for the current instance",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextProjectList(ctx, cmd)
						},
					},
					{
						Name:  "add",
						Usage: "Add a project from the current instance",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextProjectAdd(ctx, cmd)
						},
					},
					{
						Name:      "use",
						Usage:     "Select the default project",
						ArgsUsage: "[slug]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextProjectUse(ctx, cmd)
						},
					},
					{
						Name:      "remove",
						Usage:     "Remove a saved project",
						ArgsUsage: "[slug]",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return runContextProjectRemove(ctx, cmd)
						},
					},
				},
			},
		},
	}
}

func newMeCommand() *cli.Command {
	return &cli.Command{
		Name:  "me",
		Usage: "Inspect the current authenticated user",
		Commands: []*cli.Command{
			{
				Name:  "get",
				Usage: "Return the current user profile",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rt, err := runtimeFromContext(ctx)
					if err != nil {
						return err
					}
					session, err := rt.openSession(cmd.String("instance"))
					if err != nil {
						return err
					}
					defer session.close()

					user, err := session.Client.User.Me()
					if err != nil {
						return err
					}
					return render(resolveOutput(cmd), user)
				},
			},
		},
	}
}

func runContextInstanceAdd(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	defaultFrontend := "https://tree.taiga.io"
	alias, err := promptRequired("Instance alias", "")
	if err != nil {
		return err
	}
	if _, exists := rt.Config.Instances[alias]; exists {
		return fmt.Errorf("instance %q already exists", alias)
	}
	frontendURL, err := promptRequired("Taiga frontend URL", defaultFrontend)
	if err != nil {
		return err
	}
	authType, err := promptSelect("Select auth type", []string{"normal", "ldap", "token"})
	if err != nil {
		return err
	}

	username := ""
	if authType != "token" {
		username, err = promptRequired("Username", "")
		if err != nil {
			return err
		}
	}

	secret := &Secret{}
	creds := &Credentials{AuthType: authType, Username: username}
	if authType == "token" {
		token, err := promptPassword("Bearer token")
		if err != nil {
			return err
		}
		secret.Token = token
		creds.Token = token
	} else {
		password, err := promptPassword("Password")
		if err != nil {
			return err
		}
		secret.Password = password
		creds.Password = password
	}

	apiURL, baseURL, apiVersion, err := discoverInstance(frontendURL)
	if err != nil {
		return err
	}

	instance := &Instance{
		Alias:       alias,
		FrontendURL: frontendURL,
		APIURL:      apiURL,
		BaseURL:     baseURL,
		APIVersion:  apiVersion,
		AuthType:    authType,
		Username:    username,
	}

	if err := validateInstanceCredentials(instance, creds); err != nil {
		return err
	}

	if rt.Secrets.Supported() {
		if err := rt.Secrets.Set(alias, secret); err != nil {
			return err
		}
	}

	rt.upsertInstance(instance)
	if rt.Config.CurrentInstance == "" {
		rt.Config.CurrentInstance = alias
	} else {
		useAsCurrent, err := promptConfirm("Set this as the current instance?", false)
		if err != nil {
			return err
		}
		if useAsCurrent {
			rt.Config.CurrentInstance = alias
		}
	}
	if err := rt.save(); err != nil {
		return err
	}

	response := map[string]any{
		"alias":        alias,
		"frontend_url": frontendURL,
		"api_url":      apiURL,
		"auth_type":    authType,
		"username":     username,
		"is_current":   rt.Config.CurrentInstance == alias,
	}
	if !rt.Secrets.Supported() {
		response["secret_storage"] = "disabled_on_this_platform"
	}
	return render(resolveOutput(cmd), response)
}

func runContextInstanceUse(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	alias := cmd.Args().First()
	if alias == "" {
		aliases := make([]string, 0, len(rt.Config.Instances))
		for _, instance := range rt.listInstances() {
			aliases = append(aliases, instance.Alias)
		}
		selected, err := promptSelect("Select an instance", aliases)
		if err != nil {
			return err
		}
		alias = selected
	}
	if _, err := rt.getInstance(alias); err != nil {
		return err
	}
	rt.Config.CurrentInstance = alias
	if err := rt.save(); err != nil {
		return err
	}
	return render(resolveOutput(cmd), map[string]any{"current_instance": alias})
}

func runContextInstanceRemove(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	alias := cmd.Args().First()
	if alias == "" {
		aliases := make([]string, 0, len(rt.Config.Instances))
		for _, instance := range rt.listInstances() {
			aliases = append(aliases, instance.Alias)
		}
		selected, err := promptSelect("Select an instance to remove", aliases)
		if err != nil {
			return err
		}
		alias = selected
	}
	if _, err := rt.getInstance(alias); err != nil {
		return err
	}
	confirmed, err := promptConfirm(fmt.Sprintf("Remove instance %q?", alias), false)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}
	rt.deleteInstance(alias)
	if err := rt.Secrets.Delete(alias); err != nil {
		return err
	}
	if err := rt.save(); err != nil {
		return err
	}
	return render(resolveOutput(cmd), map[string]any{"removed": alias})
}

func runContextProjectList(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	alias, err := rt.resolveInstanceAlias(cmd.String("instance"))
	if err != nil {
		return err
	}
	instance, err := rt.getInstance(alias)
	if err != nil {
		return err
	}
	projects := make([]map[string]any, 0, len(instance.SavedProjects))
	for _, project := range instance.SavedProjects {
		projects = append(projects, map[string]any{
			"id":         project.ID,
			"slug":       project.Slug,
			"name":       project.Name,
			"is_default": instance.DefaultProject != nil && instance.DefaultProject.Slug == project.Slug,
		})
	}
	return render(resolveOutput(cmd), projects)
}

func runContextProjectAdd(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	alias, err := rt.resolveInstanceAlias(cmd.String("instance"))
	if err != nil {
		return err
	}
	instance, err := rt.getInstance(alias)
	if err != nil {
		return err
	}

	session, err := rt.openSession(alias)
	if err != nil {
		return err
	}
	defer session.close()

	projectList, err := session.Client.Project.List(nil)
	if err != nil {
		return err
	}
	projects, err := projectList.AsProjects()
	if err != nil {
		return err
	}
	options := make([]string, 0, len(projects))
	projectIndex := map[string]*SavedProject{}
	for _, project := range projects {
		label := fmt.Sprintf("%s (%s)", project.Slug, project.Name)
		options = append(options, label)
		projectIndex[label] = &SavedProject{
			ID:   project.ID,
			Slug: project.Slug,
			Name: project.Name,
		}
	}
	selected, err := promptSelect("Select a project", options)
	if err != nil {
		return err
	}
	project := projectIndex[selected]
	rt.upsertProject(instance, project)
	if instance.DefaultProject == nil {
		instance.DefaultProject = project
	} else {
		useAsDefault, err := promptConfirm("Set this as the default project?", false)
		if err != nil {
			return err
		}
		if useAsDefault {
			instance.DefaultProject = project
		}
	}
	if err := rt.save(); err != nil {
		return err
	}
	return render(resolveOutput(cmd), project)
}

func runContextProjectUse(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	alias, err := rt.resolveInstanceAlias(cmd.String("instance"))
	if err != nil {
		return err
	}
	instance, err := rt.getInstance(alias)
	if err != nil {
		return err
	}
	slug := cmd.Args().First()
	if slug == "" {
		options := make([]string, 0, len(instance.SavedProjects))
		for _, project := range instance.SavedProjects {
			options = append(options, project.Slug)
		}
		selected, err := promptSelect("Select a project", options)
		if err != nil {
			return err
		}
		slug = selected
	}
	project := rt.findSavedProject(instance, slug)
	if project == nil {
		return fmt.Errorf("project %q is not saved on instance %q", slug, alias)
	}
	instance.DefaultProject = project
	if err := rt.save(); err != nil {
		return err
	}
	return render(resolveOutput(cmd), map[string]any{
		"instance":        alias,
		"default_project": project,
	})
}

func runContextProjectRemove(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	alias, err := rt.resolveInstanceAlias(cmd.String("instance"))
	if err != nil {
		return err
	}
	instance, err := rt.getInstance(alias)
	if err != nil {
		return err
	}
	slug := cmd.Args().First()
	if slug == "" {
		options := make([]string, 0, len(instance.SavedProjects))
		for _, project := range instance.SavedProjects {
			options = append(options, project.Slug)
		}
		selected, err := promptSelect("Select a project to remove", options)
		if err != nil {
			return err
		}
		slug = selected
	}
	confirmed, err := promptConfirm(fmt.Sprintf("Remove project %q?", slug), false)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}
	rt.deleteProject(instance, slug)
	if err := rt.save(); err != nil {
		return err
	}
	return render(resolveOutput(cmd), map[string]any{"removed": slug})
}

func validateInstanceCredentials(instance *Instance, credentials *Credentials) error {
	client := &taigo.Client{
		BaseURL:    instance.BaseURL,
		APIversion: instance.APIVersion,
	}
	switch credentials.AuthType {
	case "token":
		return client.AuthByToken(taigo.TokenBearer, credentials.Token, "")
	case "normal", "ldap":
		return client.AuthByCredentials(&taigo.Credentials{
			Type:     credentials.AuthType,
			Username: credentials.Username,
			Password: credentials.Password,
		})
	default:
		return fmt.Errorf("unsupported auth type %q", credentials.AuthType)
	}
}

func projectSlug(project *SavedProject) string {
	if project == nil {
		return ""
	}
	return project.Slug
}
