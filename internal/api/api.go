package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/goldie/clank/internal/models"
	"github.com/goldie/clank/internal/store"
)

type API struct {
	store *store.Store
}

func New(s *store.Store) *API {
	return &API{store: s}
}

func (a *API) RegisterRoutes(r chi.Router) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		// Projects
		r.Get("/projects", a.listProjects)
		r.Post("/projects", a.createProject)
		r.Get("/projects/{id}", a.getProject)
		r.Patch("/projects/{id}", a.updateProject)
		r.Delete("/projects/{id}", a.deleteProject)

		// Tasks
		r.Get("/tasks", a.listTasks)
		r.Post("/tasks", a.createTask)
		r.Get("/tasks/{id}", a.getTask)
		r.Patch("/tasks/{id}", a.updateTask)
		r.Delete("/tasks/{id}", a.deleteTask)
		r.Post("/tasks/{id}/move", a.moveTask)
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Project handlers

func (a *API) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := a.store.ListProjects()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []models.Project{}
	}
	respondJSON(w, http.StatusOK, projects)
}

func (a *API) createProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	project, err := a.store.CreateProject(req.Name, req.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, project)
}

func (a *API) getProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	project, err := a.store.GetProject(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}
	respondJSON(w, http.StatusOK, project)
}

func (a *API) updateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	project, err := a.store.UpdateProject(id, req.Name, req.Description)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if project == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}
	respondJSON(w, http.StatusOK, project)
}

func (a *API) deleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.DeleteProject(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Task handlers

func (a *API) listTasks(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	status := r.URL.Query().Get("status")

	tasks, err := a.store.ListAllTasks(projectID, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	respondJSON(w, http.StatusOK, tasks)
}

func (a *API) createTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID   string  `json:"project_id"`
		ParentID    *string `json:"parent_id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Spec        string  `json:"spec"`
		Status      string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ProjectID == "" {
		respondError(w, http.StatusBadRequest, "project_id is required")
		return
	}
	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Status == "" {
		req.Status = models.StatusBacklog
	}

	task, err := a.store.CreateTask(req.ProjectID, req.ParentID, req.Title, req.Description, req.Spec, req.Status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, task)
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := a.store.GetTask(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}
	respondJSON(w, http.StatusOK, task)
}

func (a *API) updateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Only allow specific fields to be updated
	allowed := map[string]bool{"title": true, "description": true, "spec": true, "status": true}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}

	task, err := a.store.UpdateTask(id, updates)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}
	respondJSON(w, http.StatusOK, task)
}

func (a *API) deleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := a.store.DeleteTask(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) moveTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Status   string  `json:"status"`
		Position float64 `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	task, err := a.store.MoveTask(id, req.Status, req.Position)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}
	respondJSON(w, http.StatusOK, task)
}
