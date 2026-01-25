package store

import (
	"database/sql"
	"time"

	"github.com/goldie/clank/internal/models"
	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Projects

func (s *Store) ListProjects() ([]models.Project, error) {
	rows, err := s.db.Query(`SELECT id, name, description, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(id string) (*models.Project, error) {
	var p models.Project
	err := s.db.QueryRow(`SELECT id, name, description, created_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) CreateProject(name, description string) (*models.Project, error) {
	p := models.Project{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
	_, err := s.db.Exec(`INSERT INTO projects (id, name, description, created_at) VALUES (?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) UpdateProject(id, name, description string) (*models.Project, error) {
	_, err := s.db.Exec(`UPDATE projects SET name = ?, description = ? WHERE id = ?`, name, description, id)
	if err != nil {
		return nil, err
	}
	return s.GetProject(id)
}

func (s *Store) DeleteProject(id string) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

// Tasks

func (s *Store) ListTasks(projectID, status string, parentID *string) ([]models.Task, error) {
	query := `SELECT id, project_id, parent_id, title, description, spec, status, position, created_at FROM tasks WHERE 1=1`
	args := []interface{}{}

	if projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if parentID != nil {
		query += ` AND parent_id = ?`
		args = append(args, *parentID)
	} else {
		query += ` AND parent_id IS NULL`
	}

	query += ` ORDER BY position ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Title, &t.Description, &t.Spec, &t.Status, &t.Position, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *Store) ListAllTasks(projectID, status string) ([]models.Task, error) {
	query := `SELECT id, project_id, parent_id, title, description, spec, status, position, created_at FROM tasks WHERE 1=1`
	args := []interface{}{}

	if projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	query += ` ORDER BY position ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Title, &t.Description, &t.Spec, &t.Status, &t.Position, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *Store) GetTask(id string) (*models.Task, error) {
	var t models.Task
	err := s.db.QueryRow(`SELECT id, project_id, parent_id, title, description, spec, status, position, created_at FROM tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Title, &t.Description, &t.Spec, &t.Status, &t.Position, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Load subtasks
	subtasks, err := s.ListTasks("", "", &t.ID)
	if err != nil {
		return nil, err
	}
	t.Subtasks = subtasks

	return &t, nil
}

func (s *Store) GetNextPosition(projectID, status string) (float64, error) {
	var maxPos sql.NullFloat64
	err := s.db.QueryRow(`SELECT MAX(position) FROM tasks WHERE project_id = ? AND status = ? AND parent_id IS NULL`, projectID, status).Scan(&maxPos)
	if err != nil {
		return 0, err
	}
	if !maxPos.Valid {
		return 1.0, nil
	}
	return maxPos.Float64 + 1.0, nil
}

func (s *Store) CreateTask(projectID string, parentID *string, title, description, spec, status string) (*models.Task, error) {
	pos, err := s.GetNextPosition(projectID, status)
	if err != nil {
		return nil, err
	}

	t := models.Task{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		ParentID:    parentID,
		Title:       title,
		Description: description,
		Spec:        spec,
		Status:      status,
		Position:    pos,
		CreatedAt:   time.Now(),
	}

	_, err = s.db.Exec(`INSERT INTO tasks (id, project_id, parent_id, title, description, spec, status, position, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.ParentID, t.Title, t.Description, t.Spec, t.Status, t.Position, t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) UpdateTask(id string, updates map[string]interface{}) (*models.Task, error) {
	if len(updates) == 0 {
		return s.GetTask(id)
	}

	query := `UPDATE tasks SET `
	args := []interface{}{}
	first := true

	for key, val := range updates {
		if !first {
			query += ", "
		}
		query += key + " = ?"
		args = append(args, val)
		first = false
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return s.GetTask(id)
}

func (s *Store) DeleteTask(id string) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

func (s *Store) MoveTask(id, status string, position float64) (*models.Task, error) {
	_, err := s.db.Exec(`UPDATE tasks SET status = ?, position = ? WHERE id = ?`, status, position, id)
	if err != nil {
		return nil, err
	}
	return s.GetTask(id)
}
