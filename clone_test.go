package main

import (
	"testing"

	taigo "github.com/theriverman/taigo/v2"
)

func TestPrepareEpicCloneSameProject(t *testing.T) {
	t.Parallel()

	source := &taigo.Epic{
		ID:         10,
		Ref:        22,
		Version:    3,
		Project:    7,
		Status:     5,
		AssignedTo: 9,
		Watchers:   []int{1, 2},
		Subject:    "Original epic",
	}

	clone := prepareEpicClone(source, 7, cloneOptions{SubjectPrefix: defaultCloneSubjectPrefix})

	if clone.ID != 0 || clone.Ref != 0 || clone.Version != 0 {
		t.Fatalf("expected clone identity fields to be reset, got %+v", clone)
	}
	if clone.Status != 5 || clone.AssignedTo != 9 {
		t.Fatalf("expected same-project clone to preserve status and assignee, got %+v", clone)
	}
	if clone.Subject != "Copy of Original epic" {
		t.Fatalf("unexpected clone subject %q", clone.Subject)
	}
}

func TestPrepareUserStoryCloneCrossProject(t *testing.T) {
	t.Parallel()

	source := &taigo.UserStory{
		ID:         11,
		Ref:        33,
		Version:    4,
		Project:    7,
		Status:     8,
		Milestone:  12,
		AssignedTo: 9,
		Watchers:   []int{1, 2},
		Points:     taigo.AgilePoints{"1": 3},
		Subject:    "Original user story",
	}

	clone := prepareUserStoryClone(source, 9, cloneOptions{SubjectPrefix: defaultCloneSubjectPrefix})

	if clone.Project != 9 {
		t.Fatalf("expected target project 9, got %d", clone.Project)
	}
	if clone.Status != 0 || clone.Milestone != 0 || clone.AssignedTo != 0 {
		t.Fatalf("expected cross-project user story fields to be cleared, got %+v", clone)
	}
	if len(clone.Watchers) != 0 {
		t.Fatalf("expected watchers to be cleared, got %+v", clone.Watchers)
	}
	if clone.Points != nil {
		t.Fatalf("expected points to be cleared, got %+v", clone.Points)
	}
}

func TestPrepareTaskCloneCrossProjectAndParentOverride(t *testing.T) {
	t.Parallel()

	source := &taigo.Task{
		ID:             12,
		Ref:            44,
		Version:        5,
		Project:        7,
		Status:         8,
		Milestone:      13,
		AssignedTo:     9,
		Watchers:       []int{1, 2},
		UserStory:      21,
		TaskboardOrder: 99,
		UsOrder:        100,
		Subject:        "Original task",
	}

	clone := prepareTaskClone(source, 9, 31, cloneOptions{Subject: "Task clone"})

	if clone.Project != 9 {
		t.Fatalf("expected target project 9, got %d", clone.Project)
	}
	if clone.UserStory != 31 {
		t.Fatalf("expected target user story 31, got %d", clone.UserStory)
	}
	if clone.Status != 0 || clone.Milestone != 0 || clone.AssignedTo != 0 {
		t.Fatalf("expected cross-project task fields to be cleared, got %+v", clone)
	}
	if clone.Subject != "Task clone" {
		t.Fatalf("unexpected clone subject %q", clone.Subject)
	}
}
