package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-chi/chi/v5"
	"github.com/goldie/clank/internal/api"
	"github.com/goldie/clank/internal/db"
	"github.com/goldie/clank/internal/models"
	"github.com/goldie/clank/internal/store"
	"github.com/spf13/cobra"
)

//go:embed web
var webFS embed.FS

var (
	apiURL string
	dbPath string
	port   int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "clank",
		Short: "Minimal task/project management",
	}

	// Server command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		RunE:  runServe,
	}
	serveCmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	serveCmd.Flags().StringVar(&dbPath, "db", "clank.db", "Path to SQLite database")
	rootCmd.AddCommand(serveCmd)

	// Project commands
	projectCmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"p"},
		Short:   "Manage projects",
	}
	projectCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE:  runProjectList,
	})
	projectCmd.AddCommand(&cobra.Command{
		Use:   "new [name]",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectNew,
	})
	projectCmd.AddCommand(&cobra.Command{
		Use:   "show [id]",
		Short: "Show project details",
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectShow,
	})
	projectCmd.AddCommand(&cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectDelete,
	})
	rootCmd.AddCommand(projectCmd)

	// Task commands
	addCmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add a new task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskAdd,
	}
	addCmd.Flags().StringP("project", "p", "", "Project ID")
	addCmd.Flags().String("parent", "", "Parent task ID")
	addCmd.Flags().StringP("status", "s", "backlog", "Initial status")
	rootCmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE:  runTaskList,
	}
	listCmd.Flags().StringP("project", "p", "", "Filter by project ID")
	listCmd.Flags().StringP("status", "s", "", "Filter by status")
	rootCmd.AddCommand(listCmd)

	showCmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Show task details",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskShow,
	}
	rootCmd.AddCommand(showCmd)

	editCmd := &cobra.Command{
		Use:   "edit [id]",
		Short: "Edit a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskEdit,
	}
	editCmd.Flags().String("title", "", "New title")
	editCmd.Flags().String("description", "", "New description")
	editCmd.Flags().String("spec", "", "New spec (markdown)")
	editCmd.Flags().StringP("status", "s", "", "New status")
	rootCmd.AddCommand(editCmd)

	moveCmd := &cobra.Command{
		Use:   "move [id]",
		Short: "Move a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskMove,
	}
	moveCmd.Flags().StringP("status", "s", "", "New status")
	moveCmd.Flags().Float64("position", 0, "New position")
	rootCmd.AddCommand(moveCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskDelete,
	}
	rootCmd.AddCommand(deleteCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&apiURL, "api", os.Getenv("CLANK_API"), "API URL (or set CLANK_API)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	s := store.New(database)
	a := api.New(s)

	r := chi.NewRouter()

	// Register API routes first
	a.RegisterRoutes(r)

	// Serve embedded web files for non-API routes
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("failed to get web fs: %w", err)
	}
	fileServer := http.FileServer(http.FS(webContent))
	r.NotFound(fileServer.ServeHTTP)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, r)
}

// API client helpers

func apiRequest(method, path string, body interface{}) (*http.Response, error) {
	if apiURL == "" {
		return nil, fmt.Errorf("API URL not set. Use --api flag or set CLANK_API environment variable")
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, apiURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

func apiGet(path string, result interface{}) error {
	resp, err := apiRequest("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", body)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func apiPost(path string, body, result interface{}) error {
	resp, err := apiRequest("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", respBody)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func apiPatch(path string, body, result interface{}) error {
	resp, err := apiRequest("PATCH", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", respBody)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func apiDelete(path string) error {
	resp, err := apiRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", body)
	}
	return nil
}

// Project CLI handlers

func runProjectList(cmd *cobra.Command, args []string) error {
	var projects []models.Project
	if err := apiGet("/api/projects", &projects); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
	for _, p := range projects {
		desc := p.Description
		if len(desc) > 40 {
			desc = desc[:40] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.ID[:8], p.Name, desc)
	}
	return w.Flush()
}

func runProjectNew(cmd *cobra.Command, args []string) error {
	var project models.Project
	if err := apiPost("/api/projects", map[string]string{"name": args[0]}, &project); err != nil {
		return err
	}
	fmt.Printf("Created project: %s (%s)\n", project.Name, project.ID)
	return nil
}

func runProjectShow(cmd *cobra.Command, args []string) error {
	var project models.Project
	if err := apiGet("/api/projects/"+args[0], &project); err != nil {
		return err
	}
	fmt.Printf("ID:          %s\n", project.ID)
	fmt.Printf("Name:        %s\n", project.Name)
	fmt.Printf("Description: %s\n", project.Description)
	fmt.Printf("Created:     %s\n", project.CreatedAt.Format("2006-01-02 15:04"))
	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	if err := apiDelete("/api/projects/" + args[0]); err != nil {
		return err
	}
	fmt.Println("Project deleted")
	return nil
}

// Task CLI handlers

func runTaskAdd(cmd *cobra.Command, args []string) error {
	projectID, _ := cmd.Flags().GetString("project")
	parentID, _ := cmd.Flags().GetString("parent")
	status, _ := cmd.Flags().GetString("status")

	if projectID == "" {
		return fmt.Errorf("project ID is required (use -p flag)")
	}

	body := map[string]interface{}{
		"project_id": projectID,
		"title":      args[0],
		"status":     status,
	}
	if parentID != "" {
		body["parent_id"] = parentID
	}

	var task models.Task
	if err := apiPost("/api/tasks", body, &task); err != nil {
		return err
	}
	fmt.Printf("Created task: %s (%s)\n", task.Title, task.ID)
	return nil
}

func runTaskList(cmd *cobra.Command, args []string) error {
	projectID, _ := cmd.Flags().GetString("project")
	status, _ := cmd.Flags().GetString("status")

	path := "/api/tasks?"
	if projectID != "" {
		path += "project_id=" + projectID + "&"
	}
	if status != "" {
		path += "status=" + status + "&"
	}

	var tasks []models.Task
	if err := apiGet(path, &tasks); err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tTITLE")
	for _, t := range tasks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", t.ID[:8], t.Status, t.Title)
	}
	return w.Flush()
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	var task models.Task
	if err := apiGet("/api/tasks/"+args[0], &task); err != nil {
		return err
	}
	fmt.Printf("ID:          %s\n", task.ID)
	fmt.Printf("Title:       %s\n", task.Title)
	fmt.Printf("Status:      %s\n", task.Status)
	fmt.Printf("Project:     %s\n", task.ProjectID)
	if task.ParentID != nil {
		fmt.Printf("Parent:      %s\n", *task.ParentID)
	}
	fmt.Printf("Position:    %.2f\n", task.Position)
	fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04"))
	if task.Description != "" {
		fmt.Printf("\nDescription:\n%s\n", task.Description)
	}
	if task.Spec != "" {
		fmt.Printf("\nSpec:\n%s\n", task.Spec)
	}
	if len(task.Subtasks) > 0 {
		fmt.Printf("\nSubtasks:\n")
		for _, st := range task.Subtasks {
			fmt.Printf("  - [%s] %s (%s)\n", st.Status, st.Title, st.ID[:8])
		}
	}
	return nil
}

func runTaskEdit(cmd *cobra.Command, args []string) error {
	updates := make(map[string]interface{})

	if title, _ := cmd.Flags().GetString("title"); title != "" {
		updates["title"] = title
	}
	if desc, _ := cmd.Flags().GetString("description"); desc != "" {
		updates["description"] = desc
	}
	if spec, _ := cmd.Flags().GetString("spec"); spec != "" {
		updates["spec"] = spec
	}
	if status, _ := cmd.Flags().GetString("status"); status != "" {
		updates["status"] = status
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified")
	}

	var task models.Task
	if err := apiPatch("/api/tasks/"+args[0], updates, &task); err != nil {
		return err
	}
	fmt.Printf("Updated task: %s\n", task.ID)
	return nil
}

func runTaskMove(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	position, _ := cmd.Flags().GetFloat64("position")

	if status == "" {
		return fmt.Errorf("status is required")
	}

	body := map[string]interface{}{
		"status":   status,
		"position": position,
	}

	var task models.Task
	if err := apiPost("/api/tasks/"+args[0]+"/move", body, &task); err != nil {
		return err
	}
	fmt.Printf("Moved task to %s at position %.2f\n", task.Status, task.Position)
	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	if err := apiDelete("/api/tasks/" + args[0]); err != nil {
		return err
	}
	fmt.Println("Task deleted")
	return nil
}

func init() {
	// Suppress unused import error
	_ = strings.Contains
}
