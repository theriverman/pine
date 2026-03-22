package main

import (
	"context"
	"errors"
	"strings"

	taigo "github.com/theriverman/taigo/v2"
	cli "github.com/urfave/cli/v3"
)

const defaultCloneSubjectPrefix = "Copy of "

type cloneOptions struct {
	Subject                string
	SubjectPrefix          string
	TargetProjectID        int
	TargetProjectSlug      string
	TargetUserStoryID      int
	WithRelatedUserStories bool
	WithSubtasks           bool
}

func cloneCommand(spec resourceSpec) *cli.Command {
	switch spec.Name {
	case "epics":
		return &cli.Command{
			Name:      "clone",
			Usage:     "Clone an epic",
			ArgsUsage: "<id>",
			Flags: append(commonCloneFlags(), &cli.BoolFlag{
				Name:    "with-related-user-stories",
				Aliases: []string{"with-related-us"},
				Usage:   "Clone and re-link the epic's related user stories",
			}),
			Action: runEpicCloneCommand,
		}
	case "user-stories":
		return &cli.Command{
			Name:      "clone",
			Usage:     "Clone a user story",
			ArgsUsage: "<id>",
			Flags: append(commonCloneFlags(), &cli.BoolFlag{
				Name:    "with-subtasks",
				Aliases: []string{"with-tasks"},
				Usage:   "Clone tasks related to the user story",
			}),
			Action: runUserStoryCloneCommand,
		}
	case "tasks":
		return &cli.Command{
			Name:      "clone",
			Usage:     "Clone a task",
			ArgsUsage: "<id>",
			Flags: append(commonCloneFlags(), &cli.IntFlag{
				Name:    "user-story",
				Aliases: []string{"user_story"},
				Usage:   "Target user story ID for the cloned task",
			}),
			Action: runTaskCloneCommand,
		}
	default:
		return nil
	}
}

func commonCloneFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "subject", Usage: "Override the cloned subject"},
		&cli.StringFlag{Name: "subject-prefix", Usage: "Prefix to prepend to the cloned subject", Value: defaultCloneSubjectPrefix},
		&cli.IntFlag{Name: "project", Usage: "Target project ID"},
		&cli.StringFlag{Name: "project-slug", Usage: "Target project slug", Aliases: []string{"project_slug"}},
	}
}

func runEpicCloneCommand(ctx context.Context, cmd *cli.Command) error {
	sourceID, err := requireCloneID(cmd)
	if err != nil {
		return err
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

	result, err := cloneEpic(session, sourceID, cloneOptions{
		Subject:                cmd.String("subject"),
		SubjectPrefix:          cmd.String("subject-prefix"),
		TargetProjectID:        cmd.Int("project"),
		TargetProjectSlug:      cmd.String("project-slug"),
		WithRelatedUserStories: cmd.Bool("with-related-user-stories"),
	})
	if err != nil {
		return err
	}
	return render(resolveOutput(cmd), result)
}

func runUserStoryCloneCommand(ctx context.Context, cmd *cli.Command) error {
	sourceID, err := requireCloneID(cmd)
	if err != nil {
		return err
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

	result, err := cloneUserStory(session, sourceID, cloneOptions{
		Subject:           cmd.String("subject"),
		SubjectPrefix:     cmd.String("subject-prefix"),
		TargetProjectID:   cmd.Int("project"),
		TargetProjectSlug: cmd.String("project-slug"),
		WithSubtasks:      cmd.Bool("with-subtasks"),
	})
	if err != nil {
		return err
	}
	return render(resolveOutput(cmd), result)
}

func runTaskCloneCommand(ctx context.Context, cmd *cli.Command) error {
	sourceID, err := requireCloneID(cmd)
	if err != nil {
		return err
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

	result, err := cloneTask(session, sourceID, cloneOptions{
		Subject:           cmd.String("subject"),
		SubjectPrefix:     cmd.String("subject-prefix"),
		TargetProjectID:   cmd.Int("project"),
		TargetProjectSlug: cmd.String("project-slug"),
		TargetUserStoryID: cmd.Int("user-story"),
	})
	if err != nil {
		return err
	}
	return render(resolveOutput(cmd), result)
}

func requireCloneID(cmd *cli.Command) (int, error) {
	if cmd.Args().Len() == 0 {
		return 0, errors.New("an identifier is required")
	}
	id, ok := parseIdentifier(cmd.Args().First())
	if !ok {
		return 0, errors.New("clone requires a numeric ID")
	}
	return id, nil
}

func cloneEpic(session *Session, sourceID int, options cloneOptions) (map[string]any, error) {
	source, err := session.Client.Epic.Get(sourceID)
	if err != nil {
		return nil, err
	}
	targetProjectID, err := resolveCloneProject(session, source.Project, options.TargetProjectID, options.TargetProjectSlug)
	if err != nil {
		return nil, err
	}

	clone := prepareEpicClone(source, targetProjectID, options)
	created, err := session.Client.Epic.Create(clone)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"source": cloneSummary(source.ID, source.Ref, source.Subject, source.Project),
		"clone":  cloneSummary(created.ID, created.Ref, created.Subject, created.Project),
	}

	if options.WithRelatedUserStories {
		related, err := session.Client.Epic.ListRelatedUserStories(source.ID)
		if err != nil {
			return nil, err
		}
		clonedUserStories := make([]map[string]any, 0, len(related))
		for _, relation := range related {
			sourceUS, err := relation.GetUserStory(session.Client)
			if err != nil {
				return nil, err
			}
			createdUS, err := cloneUserStoryObject(session, sourceUS, cloneOptions{
				SubjectPrefix: childCloneSubjectPrefix(options),
			}, created.Project)
			if err != nil {
				return nil, err
			}
			if _, err := session.Client.Epic.CreateRelatedUserStory(created.ID, createdUS.ID); err != nil {
				return nil, err
			}
			clonedUserStories = append(clonedUserStories, cloneSummary(createdUS.ID, createdUS.Ref, createdUS.Subject, createdUS.Project))
		}
		result["cloned_related_user_stories"] = clonedUserStories
	}

	return result, nil
}

func cloneUserStory(session *Session, sourceID int, options cloneOptions) (map[string]any, error) {
	source, err := session.Client.UserStory.Get(sourceID)
	if err != nil {
		return nil, err
	}
	targetProjectID, err := resolveCloneProject(session, source.Project, options.TargetProjectID, options.TargetProjectSlug)
	if err != nil {
		return nil, err
	}

	created, err := cloneUserStoryObject(session, source, options, targetProjectID)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"source": cloneSummary(source.ID, source.Ref, source.Subject, source.Project),
		"clone":  cloneSummary(created.ID, created.Ref, created.Subject, created.Project),
	}

	if options.WithSubtasks {
		tasks, err := source.ListRelatedTasks(session.Client, source.ID)
		if err != nil {
			return nil, err
		}
		clonedTasks := make([]map[string]any, 0, len(tasks))
		for i := range tasks {
			createdTask, err := cloneTaskObject(session, &tasks[i], cloneOptions{
				SubjectPrefix: childCloneSubjectPrefix(options),
			}, created.Project, created.ID)
			if err != nil {
				return nil, err
			}
			clonedTasks = append(clonedTasks, cloneSummary(createdTask.ID, createdTask.Ref, createdTask.Subject, createdTask.Project))
		}
		result["cloned_subtasks"] = clonedTasks
	}

	return result, nil
}

func cloneTask(session *Session, sourceID int, options cloneOptions) (map[string]any, error) {
	source, err := session.Client.Task.Get(sourceID)
	if err != nil {
		return nil, err
	}
	targetProjectID, err := resolveCloneProject(session, source.Project, options.TargetProjectID, options.TargetProjectSlug)
	if err != nil {
		return nil, err
	}
	if options.TargetUserStoryID > 0 {
		targetUserStory, err := session.Client.UserStory.Get(options.TargetUserStoryID)
		if err != nil {
			return nil, err
		}
		if targetProjectID != 0 && targetProjectID != targetUserStory.Project {
			return nil, errors.New("target user story belongs to a different project than the requested target project")
		}
		targetProjectID = targetUserStory.Project
	}
	created, err := cloneTaskObject(session, source, options, targetProjectID, options.TargetUserStoryID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"source": cloneSummary(source.ID, source.Ref, source.Subject, source.Project),
		"clone":  cloneSummary(created.ID, created.Ref, created.Subject, created.Project),
	}, nil
}

func cloneUserStoryObject(session *Session, source *taigo.UserStory, options cloneOptions, targetProjectID int) (*taigo.UserStory, error) {
	if source == nil {
		return nil, errors.New("source user story is required")
	}
	clone := prepareUserStoryClone(source, targetProjectID, options)
	created, err := session.Client.UserStory.Create(clone)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func cloneTaskObject(session *Session, source *taigo.Task, options cloneOptions, targetProjectID int, targetUserStoryID int) (*taigo.Task, error) {
	if source == nil {
		return nil, errors.New("source task is required")
	}
	clone := prepareTaskClone(source, targetProjectID, targetUserStoryID, options)
	return session.Client.Task.Create(clone)
}

func prepareEpicClone(source *taigo.Epic, targetProjectID int, options cloneOptions) *taigo.Epic {
	clone := *source
	clone.ID = 0
	clone.Ref = 0
	clone.Version = 0
	clone.Project = targetProjectID
	clone.Subject = resolveCloneSubject(source.Subject, options)
	if targetProjectID != source.Project {
		clone.Status = 0
		clone.AssignedTo = 0
		clone.Watchers = nil
		clone.EpicsOrder = 0
	}
	return &clone
}

func prepareUserStoryClone(source *taigo.UserStory, targetProjectID int, options cloneOptions) *taigo.UserStory {
	clone := *source
	clone.ID = 0
	clone.Ref = 0
	clone.Version = 0
	clone.Project = targetProjectID
	clone.Subject = resolveCloneSubject(source.Subject, options)
	if targetProjectID != source.Project {
		clone.Status = 0
		clone.Milestone = 0
		clone.AssignedTo = 0
		clone.Watchers = nil
		clone.Points = nil
		clone.BacklogOrder = 0
		clone.KanbanOrder = 0
		clone.SprintOrder = 0
	}
	return &clone
}

func prepareTaskClone(source *taigo.Task, targetProjectID int, targetUserStoryID int, options cloneOptions) *taigo.Task {
	clone := *source
	clone.ID = 0
	clone.Ref = 0
	clone.Version = 0
	clone.Project = targetProjectID
	clone.Subject = resolveCloneSubject(source.Subject, options)

	if targetUserStoryID > 0 {
		clone.UserStory = targetUserStoryID
	} else if targetProjectID != source.Project {
		clone.UserStory = 0
	}

	if targetProjectID != source.Project {
		clone.Status = 0
		clone.Milestone = 0
		clone.AssignedTo = 0
		clone.Watchers = nil
		clone.TaskboardOrder = 0
		clone.UsOrder = 0
	}
	return &clone
}

func resolveCloneProject(session *Session, sourceProjectID int, targetProjectID int, targetProjectSlug string) (int, error) {
	switch {
	case targetProjectID > 0:
		return targetProjectID, nil
	case strings.TrimSpace(targetProjectSlug) != "":
		return session.resolveProjectID(0, targetProjectSlug)
	default:
		return sourceProjectID, nil
	}
}

func resolveCloneSubject(sourceSubject string, options cloneOptions) string {
	if strings.TrimSpace(options.Subject) != "" {
		return options.Subject
	}
	return options.SubjectPrefix + sourceSubject
}

func childCloneSubjectPrefix(options cloneOptions) string {
	if options.SubjectPrefix != "" {
		return options.SubjectPrefix
	}
	return defaultCloneSubjectPrefix
}

func cloneSummary(id int, ref int, subject string, project int) map[string]any {
	return map[string]any{
		"id":      id,
		"ref":     ref,
		"subject": subject,
		"project": project,
	}
}
