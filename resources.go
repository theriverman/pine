package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	cli "github.com/urfave/cli/v3"
)

type fieldKind string

const (
	fieldString    fieldKind = "string"
	fieldInt       fieldKind = "int"
	fieldInt64     fieldKind = "int64"
	fieldBool      fieldKind = "bool"
	fieldCSVString fieldKind = "csv-string"
	fieldCSVInt    fieldKind = "csv-int"
	fieldJSON      fieldKind = "json"
)

type fieldSpec struct {
	Name    string
	APIName string
	Kind    fieldKind
	Usage   string
	Aliases []string
}

type resourceSpec struct {
	Name                  string
	Aliases               []string
	Endpoint              string
	Usage                 string
	QueryFields           []fieldSpec
	WriteFields           []fieldSpec
	SupportsList          bool
	SupportsGet           bool
	SupportsCreate        bool
	SupportsEdit          bool
	SupportsDelete        bool
	DefaultProjectInQuery bool
	DefaultProjectInBody  bool
	ListExtras            func(*http.Response) map[string]any
	GetURL                func(*Session, string) (string, error)
}

func (s resourceSpec) findField(name string) *fieldSpec {
	for _, field := range s.WriteFields {
		if field.Name == name || field.APIName == name {
			copyField := field
			return &copyField
		}
	}
	return nil
}

func resourceSpecs() []resourceSpec {
	projectQuery := []fieldSpec{
		boolField("is-looking-for-people", "is_looking_for_people", "Filter by looking-for-people status", "is_looking_for_people"),
		boolField("is-featured", "is_featured", "Filter by featured status", "is_featured"),
		boolField("is-backlog-activated", "is_backlog_activated", "Filter by backlog status", "is_backlog_activated"),
		boolField("is-kanban-activated", "is_kanban_activated", "Filter by kanban status", "is_kanban_activated"),
		intCSVField("members", "members", "Filter by member IDs"),
		intField("member", "member", "Filter by member ID"),
		stringField("order-by", "order_by", "Order by field", "order_by"),
	}

	projectWrite := []fieldSpec{
		stringField("name", "name", "Project name"),
		stringField("description", "description", "Project description"),
		intField("creation-template", "creation_template", "Creation template ID", "creation_template"),
		boolField("is-backlog-activated", "is_backlog_activated", "Enable backlog", "is_backlog_activated"),
		boolField("is-issues-activated", "is_issues_activated", "Enable issues", "is_issues_activated"),
		boolField("is-kanban-activated", "is_kanban_activated", "Enable kanban", "is_kanban_activated"),
		boolField("is-private", "is_private", "Make project private", "is_private"),
		boolField("is-wiki-activated", "is_wiki_activated", "Enable wiki", "is_wiki_activated"),
		stringField("videoconferences", "videoconferences", "Videoconference provider"),
		stringField("videoconferences-extra-data", "videoconferences_extra_data", "Videoconference extra data", "videoconferences_extra_data"),
	}

	projectScopedQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project__slug", "Project slug", "project__slug"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		boolField("status-is-closed", "status__is_closed", "Filter by closed status", "status__is_closed"),
		boolField("include-attachments", "include_attachments", "Include attachments", "include_attachments"),
	}

	epicWrite := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project_slug", "Project slug override"),
		stringField("subject", "subject", "Epic subject"),
		stringField("description", "description", "Epic description"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		intField("status", "status", "Status ID"),
		stringField("color", "color", "Epic colour"),
		boolField("client-requirement", "client_requirement", "Mark as client requirement", "client_requirement"),
		boolField("team-requirement", "team_requirement", "Mark as team requirement", "team_requirement"),
		boolField("is-blocked", "is_blocked", "Blocked flag", "is_blocked"),
		stringField("blocked-note", "blocked_note", "Blocked note", "blocked_note"),
		int64Field("epics-order", "epics_order", "Epic order", "epics_order"),
		stringCSVField("tags", "tags", "Comma-separated tags"),
		intCSVField("watchers", "watchers", "Comma-separated watcher IDs"),
	}

	userStoryQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		intField("milestone", "milestone", "Milestone ID"),
		boolField("milestone-isnull", "milestone__isnull", "Filter by null milestone", "milestone__isnull"),
		intField("status", "status", "Status ID"),
		boolField("status-is-archived", "status__is_archived", "Filter by archived status", "status__is_archived"),
		stringField("tags", "tags", "Comma-separated tags"),
		intField("watchers", "watchers", "Watcher ID"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		intField("epic", "epic", "Epic ID"),
		intField("role", "role", "Role ID"),
		boolField("status-is-closed", "status__is_closed", "Filter by closed status", "status__is_closed"),
		boolField("include-attachments", "include_attachments", "Include attachments", "include_attachments"),
		boolField("include-tasks", "include_tasks", "Include tasks", "include_tasks"),
		intField("exclude-status", "exclude_status", "Excluded status ID", "exclude_status"),
		stringField("exclude-tags", "exclude_tags", "Comma-separated excluded tags", "exclude_tags"),
		intField("exclude-assigned-to", "exclude_assigned_to", "Excluded assigned user ID", "exclude_assigned_to"),
		intField("exclude-role", "exclude_role", "Excluded role ID", "exclude_role"),
		intField("exclude-epic", "exclude_epic", "Excluded epic ID", "exclude_epic"),
	}

	userStoryWrite := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project_slug", "Project slug override"),
		stringField("subject", "subject", "User story subject"),
		stringField("description", "description", "User story description"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		intField("status", "status", "Status ID"),
		intField("milestone", "milestone", "Milestone ID"),
		intField("epic", "epic", "Epic ID"),
		boolField("client-requirement", "client_requirement", "Mark as client requirement", "client_requirement"),
		boolField("team-requirement", "team_requirement", "Mark as team requirement", "team_requirement"),
		boolField("is-blocked", "is_blocked", "Blocked flag", "is_blocked"),
		stringField("blocked-note", "blocked_note", "Blocked note", "blocked_note"),
		int64Field("backlog-order", "backlog_order", "Backlog order", "backlog_order"),
		int64Field("kanban-order", "kanban_order", "Kanban order", "kanban_order"),
		intField("sprint-order", "sprint_order", "Sprint order", "sprint_order"),
		stringCSVField("tags", "tags", "Comma-separated tags"),
		intCSVField("watchers", "watchers", "Comma-separated watcher IDs"),
		stringCSVField("external-reference", "external_reference", "Comma-separated external references", "external_reference"),
		jsonField("points-json", "points", "JSON object of agile points"),
	}

	taskQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		intField("status", "status", "Status ID"),
		stringField("tags", "tags", "Comma-separated tags"),
		intField("user-story", "user_story", "User story ID", "user_story"),
		intField("role", "role", "Role ID"),
		intField("owner", "owner", "Owner ID"),
		intField("milestone", "milestone", "Milestone ID"),
		intField("watchers", "watchers", "Watcher ID"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		boolField("status-is-closed", "status__is_closed", "Filter by closed status", "status__is_closed"),
		intField("exclude-status", "exclude_status", "Excluded status ID", "exclude_status"),
		stringField("exclude-tags", "exclude_tags", "Comma-separated excluded tags", "exclude_tags"),
		intField("exclude-role", "exclude_role", "Excluded role ID", "exclude_role"),
		intField("exclude-owner", "exclude_owner", "Excluded owner ID", "exclude_owner"),
		intField("exclude-assigned-to", "exclude_assigned_to", "Excluded assigned user ID", "exclude_assigned_to"),
		boolField("include-attachments", "include_attachments", "Include attachments", "include_attachments"),
	}

	taskWrite := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project_slug", "Project slug override"),
		stringField("subject", "subject", "Task subject"),
		stringField("description", "description", "Task description"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		intField("status", "status", "Status ID"),
		intField("milestone", "milestone", "Milestone ID"),
		intField("user-story", "user_story", "User story ID", "user_story"),
		intField("owner", "owner", "Owner ID"),
		boolField("is-blocked", "is_blocked", "Blocked flag", "is_blocked"),
		stringField("blocked-note", "blocked_note", "Blocked note", "blocked_note"),
		int64Field("kanban-order", "kanban_order", "Kanban order", "kanban_order"),
		intField("taskboard-order", "taskboard_order", "Taskboard order", "taskboard_order"),
		intField("us-order", "us_order", "User story order", "us_order"),
		stringCSVField("tags", "tags", "Comma-separated tags"),
		intCSVField("watchers", "watchers", "Comma-separated watcher IDs"),
	}

	issueQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		intField("milestone", "milestone", "Milestone ID"),
		boolField("milestone-isnull", "milestone__isnull", "Filter by null milestone", "milestone__isnull"),
		intField("status", "status", "Status ID"),
		boolField("status-is-archived", "status__is_archived", "Filter by archived status", "status__is_archived"),
		stringField("tags", "tags", "Comma-separated tags"),
		intField("watchers", "watchers", "Watcher ID"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		intField("epic", "epic", "Epic ID"),
		intField("role", "role", "Role ID"),
		boolField("status-is-closed", "status__is_closed", "Filter by closed status", "status__is_closed"),
		intField("type", "type", "Issue type ID"),
		intField("severity", "severity", "Severity ID"),
		intField("priority", "priority", "Priority ID"),
		intField("owner", "owner", "Owner ID"),
		intField("exclude-status", "exclude_status", "Excluded status ID", "exclude_status"),
		stringField("exclude-tags", "exclude_tags", "Comma-separated excluded tags", "exclude_tags"),
		intField("exclude-assigned-to", "exclude_assigned_to", "Excluded assigned user ID", "exclude_assigned_to"),
		intField("exclude-role", "exclude_role", "Excluded role ID", "exclude_role"),
		intField("exclude-epic", "exclude_epic", "Excluded epic ID", "exclude_epic"),
		intField("exclude-severity", "exclude_severity", "Excluded severity ID", "exclude_severity"),
		intField("exclude-priority", "exclude_priority", "Excluded priority ID", "exclude_priority"),
		intField("exclude-owner", "exclude_owner", "Excluded owner ID", "exclude_owner"),
		intField("exclude-type", "exclude_type", "Excluded type ID", "exclude_type"),
		boolField("include-attachments", "include_attachments", "Include attachments", "include_attachments"),
	}

	issueWrite := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project_slug", "Project slug override"),
		stringField("subject", "subject", "Issue subject"),
		stringField("description", "description", "Issue description"),
		intField("assigned-to", "assigned_to", "Assigned user ID", "assigned_to"),
		intField("status", "status", "Status ID"),
		intField("milestone", "milestone", "Milestone ID"),
		intField("owner", "owner", "Owner ID"),
		intField("priority", "priority", "Priority ID"),
		intField("severity", "severity", "Severity ID"),
		intField("type", "type", "Issue type ID"),
		boolField("is-blocked", "is_blocked", "Blocked flag", "is_blocked"),
		stringField("blocked-note", "blocked_note", "Blocked note", "blocked_note"),
		stringCSVField("tags", "tags", "Comma-separated tags"),
		intCSVField("watchers", "watchers", "Comma-separated watcher IDs"),
		stringField("due-date", "due_date", "Due date", "due_date"),
		stringField("due-date-reason", "due_date_reason", "Due date reason", "due_date_reason"),
		stringField("due-date-status", "due_date_status", "Due date status", "due_date_status"),
	}

	milestoneQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		boolField("closed", "closed", "Filter by closed status"),
	}

	milestoneWrite := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project_slug", "Project slug override"),
		stringField("name", "name", "Milestone name"),
		stringField("estimated-start", "estimated_start", "Estimated start date", "estimated_start"),
		stringField("estimated-finish", "estimated_finish", "Estimated finish date", "estimated_finish"),
		boolField("closed", "closed", "Closed flag"),
	}

	wikiQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("slug", "slug", "Wiki slug"),
	}

	wikiWrite := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("project-slug", "project_slug", "Project slug override"),
		stringField("slug", "slug", "Wiki slug"),
		stringField("content", "content", "Wiki content"),
	}

	userQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
	}

	userWrite := []fieldSpec{
		stringField("bio", "bio", "User biography"),
		stringField("color", "color", "User colour"),
		stringField("email", "email", "Email address"),
		stringField("full-name", "full_name", "Full name", "full_name"),
		stringField("lang", "lang", "Language"),
		boolField("read-new-terms", "read_new_terms", "Read new terms flag", "read_new_terms"),
		stringField("theme", "theme", "Theme"),
		stringField("timezone", "timezone", "Timezone"),
		stringField("username", "username", "Username"),
	}

	searchQuery := []fieldSpec{
		intField("project", "project", "Project ID"),
		stringField("text", "text", "Search text"),
	}

	return []resourceSpec{
		{
			Name:           "projects",
			Endpoint:       "projects",
			Usage:          "Manage Taiga projects",
			QueryFields:    projectQuery,
			WriteFields:    projectWrite,
			SupportsList:   true,
			SupportsGet:    true,
			SupportsCreate: true,
			SupportsEdit:   true,
			SupportsDelete: true,
			GetURL: func(session *Session, identifier string) (string, error) {
				if id, ok := parseIdentifier(identifier); ok {
					return session.Client.MakeURL("projects", strconv.Itoa(id)), nil
				}
				values := url.Values{}
				values.Set("slug", identifier)
				return appendQuery(session.Client.MakeURL("projects", "by_slug"), values), nil
			},
		},
		{
			Name:                  "epics",
			Endpoint:              "epics",
			Usage:                 "Manage Taiga epics",
			QueryFields:           projectScopedQuery,
			WriteFields:           epicWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        true,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  true,
		},
		{
			Name:                  "user-stories",
			Aliases:               []string{"us"},
			Endpoint:              "userstories",
			Usage:                 "Manage Taiga user stories",
			QueryFields:           userStoryQuery,
			WriteFields:           userStoryWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        true,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  true,
		},
		{
			Name:                  "tasks",
			Endpoint:              "tasks",
			Usage:                 "Manage Taiga tasks",
			QueryFields:           taskQuery,
			WriteFields:           taskWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        true,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  true,
		},
		{
			Name:                  "issues",
			Endpoint:              "issues",
			Usage:                 "Manage Taiga issues",
			QueryFields:           issueQuery,
			WriteFields:           issueWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        true,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  true,
		},
		{
			Name:                  "milestones",
			Endpoint:              "milestones",
			Usage:                 "Manage Taiga milestones",
			QueryFields:           milestoneQuery,
			WriteFields:           milestoneWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        true,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  true,
			ListExtras: func(response *http.Response) map[string]any {
				if response == nil {
					return nil
				}
				return map[string]any{
					"opened_milestones": response.Header.Get("Taiga-Info-Total-Opened-Milestones"),
					"closed_milestones": response.Header.Get("Taiga-Info-Total-Closed-Milestones"),
				}
			},
		},
		{
			Name:                  "wiki",
			Endpoint:              "wiki",
			Usage:                 "Manage Taiga wiki pages",
			QueryFields:           wikiQuery,
			WriteFields:           wikiWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        true,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  true,
			GetURL: func(session *Session, identifier string) (string, error) {
				if id, ok := parseIdentifier(identifier); ok {
					return session.Client.MakeURL("wiki", strconv.Itoa(id)), nil
				}
				projectID, err := session.resolveProjectID(0, "")
				if err != nil {
					return "", err
				}
				values := url.Values{}
				values.Set("slug", identifier)
				values.Set("project", strconv.Itoa(projectID))
				return appendQuery(session.Client.MakeURL("wiki", "by_slug"), values), nil
			},
		},
		{
			Name:                  "users",
			Endpoint:              "users",
			Usage:                 "Manage Taiga users",
			QueryFields:           userQuery,
			WriteFields:           userWrite,
			SupportsList:          true,
			SupportsGet:           true,
			SupportsCreate:        false,
			SupportsEdit:          true,
			SupportsDelete:        true,
			DefaultProjectInQuery: false,
			DefaultProjectInBody:  false,
		},
		{
			Name:                  "search",
			Endpoint:              "search",
			Usage:                 "Search Taiga resources",
			QueryFields:           searchQuery,
			SupportsList:          true,
			SupportsGet:           false,
			SupportsCreate:        false,
			SupportsEdit:          false,
			SupportsDelete:        false,
			DefaultProjectInQuery: true,
			DefaultProjectInBody:  false,
		},
	}
}

func resourceCommand(spec resourceSpec) *cli.Command {
	command := &cli.Command{
		Name:    spec.Name,
		Aliases: spec.Aliases,
		Usage:   spec.Usage,
	}

	if spec.SupportsList {
		command.Commands = append(command.Commands, &cli.Command{
			Name:  "list",
			Usage: "List resources",
			Flags: append(resourceQueryFlags(spec.QueryFields), commonListFlags()...),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runList(ctx, cmd, spec)
			},
		})
	}
	if spec.SupportsGet {
		command.Commands = append(command.Commands, &cli.Command{
			Name:      "get",
			Usage:     "Get a resource by identifier",
			ArgsUsage: "<id-or-slug>",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runGet(ctx, cmd, spec)
			},
		})
	}
	if spec.SupportsCreate {
		command.Commands = append(command.Commands, &cli.Command{
			Name:  "create",
			Usage: "Create a resource",
			Flags: append(resourceWriteFlags(spec.WriteFields), jsonFlags()...),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runCreate(ctx, cmd, spec)
			},
		})
	}
	if spec.SupportsEdit {
		command.Commands = append(command.Commands, &cli.Command{
			Name:      "edit",
			Usage:     "Edit a resource",
			ArgsUsage: "<id>",
			Flags:     append(resourceWriteFlags(spec.WriteFields), editFlags()...),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runEdit(ctx, cmd, spec)
			},
		})
	}
	if spec.SupportsDelete {
		command.Commands = append(command.Commands, &cli.Command{
			Name:      "delete",
			Usage:     "Delete a resource",
			ArgsUsage: "<id>",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "yes", Usage: "Skip the confirmation prompt"},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runDelete(ctx, cmd, spec)
			},
		})
	}
	if spec.Name == "epics" {
		command.Commands = append(command.Commands, &cli.Command{
			Name:    "bulk-creation",
			Aliases: []string{"bulk_creation"},
			Usage:   "Bulk-create epics from a JSON file",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "json", Usage: "Path to a JSON file", Required: true},
				&cli.IntFlag{Name: "project", Usage: "Project ID"},
				&cli.StringFlag{Name: "project-slug", Usage: "Project slug", Aliases: []string{"project_slug"}},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runEpicBulkCreate(ctx, cmd)
			},
		})
	}
	if clone := cloneCommand(spec); clone != nil {
		command.Commands = append(command.Commands, clone)
	}
	return command
}

func commonListFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{Name: "page", Usage: "Page number"},
		&cli.IntFlag{Name: "page-size", Usage: "Page size", Aliases: []string{"page_size"}},
	}
}

func jsonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "from-json", Usage: "Path to a JSON file to merge into the request", Aliases: []string{"json"}},
	}
}

func editFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "from-json", Usage: "Path to a JSON file to merge into the request", Aliases: []string{"json"}},
		&cli.StringSliceFlag{Name: "clear", Usage: "Fields to clear explicitly"},
	}
}

func resourceQueryFlags(specs []fieldSpec) []cli.Flag {
	flags := make([]cli.Flag, 0, len(specs))
	for _, spec := range specs {
		flags = append(flags, buildFlag(spec))
	}
	return flags
}

func resourceWriteFlags(specs []fieldSpec) []cli.Flag {
	flags := make([]cli.Flag, 0, len(specs))
	for _, spec := range specs {
		flags = append(flags, buildFlag(spec))
	}
	return flags
}

func buildFlag(spec fieldSpec) cli.Flag {
	switch spec.Kind {
	case fieldString, fieldCSVString, fieldCSVInt, fieldJSON:
		return &cli.StringFlag{Name: spec.Name, Usage: spec.Usage, Aliases: spec.Aliases}
	case fieldInt:
		return &cli.IntFlag{Name: spec.Name, Usage: spec.Usage, Aliases: spec.Aliases}
	case fieldInt64:
		return &cli.Int64Flag{Name: spec.Name, Usage: spec.Usage, Aliases: spec.Aliases}
	case fieldBool:
		return &cli.BoolFlag{Name: spec.Name, Usage: spec.Usage, Aliases: spec.Aliases}
	default:
		return &cli.StringFlag{Name: spec.Name, Usage: spec.Usage, Aliases: spec.Aliases}
	}
}

func runList(ctx context.Context, cmd *cli.Command, spec resourceSpec) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := rt.openSession(cmd.String("instance"))
	if err != nil {
		return err
	}
	defer session.close()

	queryValues, err := collectQueryValues(cmd, spec.QueryFields)
	if err != nil {
		return err
	}
	if spec.DefaultProjectInQuery && queryValues.Get("project") == "" && queryValues.Get("project__slug") == "" && session.activeProjectID() > 0 {
		queryValues.Set("project", strconv.Itoa(session.activeProjectID()))
	}
	if cmd.IsSet("page") {
		queryValues.Set("page", strconv.Itoa(cmd.Int("page")))
	}
	if cmd.IsSet("page-size") {
		queryValues.Set("page_size", strconv.Itoa(cmd.Int("page-size")))
	}

	url := requestURL(session.Client, spec.Endpoint, queryValues)
	var response any
	httpResponse, err := session.Client.Request.Get(url, &response)
	if err != nil {
		return err
	}

	envelope := PaginationEnvelope{
		Items:      response,
		Pagination: paginationFromClient(session.Client),
	}
	if envelope.Pagination.Page == 0 && cmd.Int("page") > 0 {
		envelope.Pagination.Page = cmd.Int("page")
	}
	if envelope.Pagination.PageSize == 0 && cmd.Int("page-size") > 0 {
		envelope.Pagination.PageSize = cmd.Int("page-size")
	}
	if spec.ListExtras != nil {
		envelope.Extra = spec.ListExtras(httpResponse)
	}
	return renderView(resolveOutput(cmd), envelope, spec.Name)
}

func runGet(ctx context.Context, cmd *cli.Command, spec resourceSpec) error {
	if cmd.Args().Len() == 0 {
		return errors.New("an identifier is required")
	}
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := rt.openSession(cmd.String("instance"))
	if err != nil {
		return err
	}
	defer session.close()

	identifier := cmd.Args().First()
	url := session.Client.MakeURL(spec.Endpoint, identifier)
	if spec.GetURL != nil {
		url, err = spec.GetURL(session, identifier)
		if err != nil {
			return err
		}
	} else if _, ok := parseIdentifier(identifier); !ok {
		return fmt.Errorf("%s get requires a numeric ID", spec.Name)
	}

	response := map[string]any{}
	if _, err := session.Client.Request.Get(url, &response); err != nil {
		return err
	}
	return renderView(resolveOutput(cmd), response, spec.Name)
}

func runCreate(ctx context.Context, cmd *cli.Command, spec resourceSpec) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := rt.openSession(cmd.String("instance"))
	if err != nil {
		return err
	}
	defer session.close()

	payload := map[string]any{}
	if path := cmd.String("from-json"); path != "" {
		payload, err = readJSONMap(path)
		if err != nil {
			return err
		}
	}

	flagValues, err := collectPayloadValues(cmd, spec.WriteFields)
	if err != nil {
		return err
	}
	payload = mergeMaps(payload, flagValues)
	if err := applyDefaultProject(session, payload, spec.DefaultProjectInBody); err != nil {
		return err
	}

	response := map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL(spec.Endpoint), payload, &response); err != nil {
		return err
	}
	return renderView(resolveOutput(cmd), response, spec.Name)
}

func runEdit(ctx context.Context, cmd *cli.Command, spec resourceSpec) error {
	if cmd.Args().Len() == 0 {
		return errors.New("an identifier is required")
	}
	identifier := cmd.Args().First()
	if _, ok := parseIdentifier(identifier); !ok {
		return errors.New("edit requires a numeric ID")
	}

	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := rt.openSession(cmd.String("instance"))
	if err != nil {
		return err
	}
	defer session.close()

	url := session.Client.MakeURL(spec.Endpoint, identifier)
	current := map[string]any{}
	if _, err := session.Client.Request.Get(url, &current); err != nil {
		return err
	}

	desired := cloneMap(current)
	if path := cmd.String("from-json"); path != "" {
		jsonValues, err := readJSONMap(path)
		if err != nil {
			return err
		}
		desired = mergeMaps(desired, jsonValues)
	}
	flagValues, err := collectPayloadValues(cmd, spec.WriteFields)
	if err != nil {
		return err
	}
	desired = mergeMaps(desired, flagValues)
	clearValues := cmd.StringSlice("clear")
	if len(clearValues) > 0 {
		if err := applyClearValues(spec, desired, clearValues); err != nil {
			return err
		}
	}
	if err := applyDefaultProject(session, desired, spec.DefaultProjectInBody); err != nil {
		return err
	}

	patch := deriveChangedFields(current, desired)
	if version, ok := current["version"]; ok {
		patch["version"] = version
	}
	if len(patch) == 0 {
		return errors.New("no changes were supplied")
	}
	if len(patch) == 1 {
		if _, hasVersion := patch["version"]; hasVersion {
			return errors.New("no changes were supplied")
		}
	}

	response := map[string]any{}
	if _, err := session.Client.Request.Patch(url, patch, &response); err != nil {
		return err
	}
	return renderView(resolveOutput(cmd), response, spec.Name)
}

func runDelete(ctx context.Context, cmd *cli.Command, spec resourceSpec) error {
	if cmd.Args().Len() == 0 {
		return errors.New("an identifier is required")
	}
	identifier := cmd.Args().First()
	if _, ok := parseIdentifier(identifier); !ok {
		return errors.New("delete requires a numeric ID")
	}

	if !cmd.Bool("yes") {
		ok, err := promptConfirm(fmt.Sprintf("Delete %s %s?", spec.Name, identifier), false)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := rt.openSession(cmd.String("instance"))
	if err != nil {
		return err
	}
	defer session.close()

	response, err := session.Client.Request.Delete(session.Client.MakeURL(spec.Endpoint, identifier))
	if err != nil {
		return err
	}
	return render(resolveOutput(cmd), map[string]any{
		"deleted": true,
		"status":  response.StatusCode,
	})
}

func runEpicBulkCreate(ctx context.Context, cmd *cli.Command) error {
	rt, err := runtimeFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := rt.openSession(cmd.String("instance"))
	if err != nil {
		return err
	}
	defer session.close()

	data, err := readJSONArray(cmd.String("json"))
	if err != nil {
		return err
	}
	projectID, err := session.resolveProjectID(cmd.Int("project"), cmd.String("project-slug"))
	if err != nil {
		return err
	}

	lines := make([]string, 0, len(data))
	for _, item := range data {
		switch typed := item.(type) {
		case string:
			lines = append(lines, typed)
		case map[string]any:
			if subject, ok := typed["subject"].(string); ok && strings.TrimSpace(subject) != "" {
				lines = append(lines, subject)
			}
		}
	}
	response, err := postEpicBulkCreate(session, projectID, lines)
	if err != nil {
		return err
	}
	return renderView(resolveOutput(cmd), response, "epics")
}

func postEpicBulkCreate(session *Session, projectID int, lines []string) ([]map[string]any, error) {
	payload := map[string]any{
		"project":    projectID,
		"bulk_epics": strings.Join(lines, "\n"),
	}
	response := []map[string]any{}
	if _, err := session.Client.Request.Post(session.Client.MakeURL("epics", "bulk_create"), payload, &response); err != nil {
		if strings.Contains(err.Error(), "project_id") {
			payload["project_id"] = projectID
			delete(payload, "project")
			if _, retryErr := session.Client.Request.Post(session.Client.MakeURL("epics", "bulk_create"), payload, &response); retryErr != nil {
				return nil, retryErr
			}
			return response, nil
		}
		return nil, err
	}
	return response, nil
}

func collectQueryValues(cmd *cli.Command, fields []fieldSpec) (url.Values, error) {
	values := url.Values{}
	for _, field := range fields {
		if !cmd.IsSet(field.Name) {
			continue
		}
		switch field.Kind {
		case fieldString, fieldCSVString:
			values.Set(field.APIName, cmd.String(field.Name))
		case fieldInt:
			values.Set(field.APIName, strconv.Itoa(cmd.Int(field.Name)))
		case fieldInt64:
			values.Set(field.APIName, strconv.FormatInt(cmd.Int64(field.Name), 10))
		case fieldBool:
			values.Set(field.APIName, strconv.FormatBool(cmd.Bool(field.Name)))
		case fieldCSVInt:
			values.Set(field.APIName, cmd.String(field.Name))
		default:
			values.Set(field.APIName, cmd.String(field.Name))
		}
	}
	return values, nil
}

func collectPayloadValues(cmd *cli.Command, fields []fieldSpec) (map[string]any, error) {
	values := map[string]any{}
	for _, field := range fields {
		if !cmd.IsSet(field.Name) {
			continue
		}
		switch field.Kind {
		case fieldString:
			values[field.APIName] = cmd.String(field.Name)
		case fieldInt:
			values[field.APIName] = cmd.Int(field.Name)
		case fieldInt64:
			values[field.APIName] = cmd.Int64(field.Name)
		case fieldBool:
			values[field.APIName] = cmd.Bool(field.Name)
		case fieldCSVString:
			values[field.APIName] = asStringSlice(cmd.String(field.Name))
		case fieldCSVInt:
			parsed, err := asIntSlice(cmd.String(field.Name))
			if err != nil {
				return nil, err
			}
			values[field.APIName] = parsed
		case fieldJSON:
			var raw any
			if err := json.Unmarshal([]byte(cmd.String(field.Name)), &raw); err != nil {
				return nil, fmt.Errorf("parse %s: %w", field.Name, err)
			}
			values[field.APIName] = raw
		}
	}
	return values, nil
}

func applyDefaultProject(session *Session, payload map[string]any, required bool) error {
	if !required {
		delete(payload, "project_slug")
		return nil
	}

	projectID := 0
	projectSlug := ""
	if raw, ok := payload["project"]; ok {
		switch typed := raw.(type) {
		case int:
			projectID = typed
		case int64:
			projectID = int(typed)
		case float64:
			projectID = int(typed)
		}
	}
	if raw, ok := payload["project_slug"].(string); ok {
		projectSlug = raw
	}
	resolvedProjectID, err := session.resolveProjectID(projectID, projectSlug)
	if err != nil {
		return err
	}
	payload["project"] = resolvedProjectID
	delete(payload, "project_slug")
	return nil
}

func applyClearValues(spec resourceSpec, payload map[string]any, clearValues []string) error {
	for _, clearField := range clearValues {
		field := spec.findField(clearField)
		if field == nil {
			return fmt.Errorf("cannot clear unknown field %q", clearField)
		}
		payload[field.APIName] = field.zeroValue()
	}
	return nil
}

func (f fieldSpec) zeroValue() any {
	switch f.Kind {
	case fieldString:
		return ""
	case fieldInt:
		return 0
	case fieldInt64:
		return int64(0)
	case fieldBool:
		return false
	case fieldCSVString:
		return []string{}
	case fieldCSVInt:
		return []int{}
	case fieldJSON:
		return map[string]any{}
	default:
		return nil
	}
}

func stringField(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldString, Usage: usage, Aliases: aliases}
}

func intField(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldInt, Usage: usage, Aliases: aliases}
}

func int64Field(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldInt64, Usage: usage, Aliases: aliases}
}

func boolField(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldBool, Usage: usage, Aliases: aliases}
}

func stringCSVField(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldCSVString, Usage: usage, Aliases: aliases}
}

func intCSVField(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldCSVInt, Usage: usage, Aliases: aliases}
}

func jsonField(name, apiName, usage string, aliases ...string) fieldSpec {
	return fieldSpec{Name: name, APIName: apiName, Kind: fieldJSON, Usage: usage, Aliases: aliases}
}
