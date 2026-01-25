package models

import "time"

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Task struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Spec        string    `json:"spec,omitempty"`
	Status      string    `json:"status"`
	Position    float64   `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	Subtasks    []Task    `json:"subtasks,omitempty"`
}

const (
	StatusBacklog    = "backlog"
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
)
