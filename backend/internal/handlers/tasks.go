package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"go-react-template/backend/internal/auth"
	"go-react-template/backend/internal/db"
	"go-react-template/backend/pkg/sqlcqueries"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TaskHandler struct {
	dbService *db.DBService
}

func NewTaskHandler(dbService *db.DBService) *TaskHandler {
	return &TaskHandler{dbService: dbService}
}

// Helper function to check if the authenticated user owns the course associated with a task/courseID
func (h *TaskHandler) checkCourseOwnership(ctx context.Context, firebaseUIDStr string, courseID int32) (bool, error) {
	userID, err := uuid.Parse(firebaseUIDStr)
	if err != nil {
		return false, err // Invalid user ID format
	}
	_, err = h.dbService.GetCourseByID(ctx, sqlcqueries.GetCourseByIDParams{
		CourseID: courseID,
		UserID:   userID,
	})
	if err != nil {
		// If ErrNoRows, user doesn't own the course or it doesn't exist
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		// Other database error
		return false, err
	}
	// Course found and belongs to the user
	return true, nil
}

// POST /api/v1/courses/{courseID}/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	courseIDStr := chi.URLParam(r, "courseID")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid course ID format")
		return
	}

	// --- Ownership Check ---
	ownsCourse, err := h.checkCourseOwnership(r.Context(), firebaseUIDStr, int32(courseID))
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}
	if !ownsCourse {
		respondWithError(w, http.StatusForbidden, "User does not own the specified course")
		return
	}
	// --- End Ownership Check ---

	var params sqlcqueries.CreateTaskParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// Basic validation
	if params.Title == "" || params.PointsValue < 0 || params.DueDate.IsZero() {
		respondWithError(w, http.StatusBadRequest, "Invalid input: title, non-negative points, and due date required")
		return
	}

	params.CourseID = int32(courseID) // Set course ID from URL param

	task, err := h.dbService.CreateTask(r.Context(), params)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusCreated, task)
}

// GET /api/v1/courses/{courseID}/tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	courseIDStr := chi.URLParam(r, "courseID")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid course ID format")
		return
	}

	// --- Ownership Check ---
	ownsCourse, err := h.checkCourseOwnership(r.Context(), firebaseUIDStr, int32(courseID))
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}
	if !ownsCourse {
		respondWithError(w, http.StatusForbidden, "User does not own the specified course")
		return
	}
	// --- End Ownership Check ---

	tasks, err := h.dbService.ListTasksByCourse(r.Context(), sqlcqueries.ListTasksByCourseParams{CourseID: int32(courseID)})
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	if tasks == nil {
		tasks = []*sqlcqueries.Task{}
	}

	respondWithJSON(w, http.StatusOK, tasks)
}

// GET /api/v1/tasks/{taskID}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID format")
		return
	}

	task, err := h.dbService.GetTaskByID(r.Context(), sqlcqueries.GetTaskByIDParams{TaskID: int32(taskID)})
	if err != nil {
		code, msg := mapDBError(err) // Handles ErrNoRows as 404
		respondWithError(w, code, msg)
		return
	}

	// --- Ownership Check ---
	ownsCourse, err := h.checkCourseOwnership(r.Context(), firebaseUIDStr, task.CourseID)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}
	if !ownsCourse {
		// Even if task exists, if user doesn't own the course, treat as not found for security
		respondWithError(w, http.StatusNotFound, "Resource not found")
		return
	}
	// --- End Ownership Check ---

	respondWithJSON(w, http.StatusOK, task)
}

// PUT /api/v1/tasks/{taskID}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID format")
		return
	}

	// First, get the task to check ownership via its course
	existingTask, err := h.dbService.GetTaskByID(r.Context(), sqlcqueries.GetTaskByIDParams{TaskID: int32(taskID)})
	if err != nil {
		code, msg := mapDBError(err) // Handles ErrNoRows as 404
		respondWithError(w, code, msg)
		return
	}

	// --- Ownership Check ---
	ownsCourse, err := h.checkCourseOwnership(r.Context(), firebaseUIDStr, existingTask.CourseID)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}
	if !ownsCourse {
		respondWithError(w, http.StatusNotFound, "Resource not found") // Treat as not found
		return
	}
	// --- End Ownership Check ---

	var params sqlcqueries.UpdateTaskDetailsParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// Basic validation
	if params.Title == "" || params.PointsValue < 0 || params.DueDate.IsZero() || (params.Status != sqlcqueries.TaskStatusInProgress && params.Status != sqlcqueries.TaskStatusDone) {
		respondWithError(w, http.StatusBadRequest, "Invalid input: title, non-negative points, due date, and valid status ('in_progress' or 'done') required")
		return
	}

	params.TaskID = int32(taskID) // Set task ID from URL param

	updatedTask, err := h.dbService.UpdateTaskDetails(r.Context(), params)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedTask)
}

// DELETE /api/v1/tasks/{taskID}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID format")
		return
	}

	// First, get the task to check ownership via its course
	existingTask, err := h.dbService.GetTaskByID(r.Context(), sqlcqueries.GetTaskByIDParams{TaskID: int32(taskID)})
	if err != nil {
		code, msg := mapDBError(err) // Handles ErrNoRows as 404
		respondWithError(w, code, msg)
		return
	}

	// --- Ownership Check ---
	ownsCourse, err := h.checkCourseOwnership(r.Context(), firebaseUIDStr, existingTask.CourseID)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}
	if !ownsCourse {
		respondWithError(w, http.StatusNotFound, "Resource not found") // Treat as not found
		return
	}
	// --- End Ownership Check ---

	err = h.dbService.DeleteTask(r.Context(), sqlcqueries.DeleteTaskParams{TaskID: int32(taskID)})
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Task deleted successfully"})
}

// PATCH /api/v1/tasks/{taskID}/status - Example for partial update (only status)
func (h *TaskHandler) UpdateTaskStatusOnly(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	taskIDStr := chi.URLParam(r, "taskID")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID format")
		return
	}

	// First, get the task to check ownership via its course
	existingTask, err := h.dbService.GetTaskByID(r.Context(), sqlcqueries.GetTaskByIDParams{TaskID: int32(taskID)})
	if err != nil {
		code, msg := mapDBError(err) // Handles ErrNoRows as 404
		respondWithError(w, code, msg)
		return
	}

	// --- Ownership Check ---
	ownsCourse, err := h.checkCourseOwnership(r.Context(), firebaseUIDStr, existingTask.CourseID)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}
	if !ownsCourse {
		respondWithError(w, http.StatusNotFound, "Resource not found") // Treat as not found
		return
	}
	// --- End Ownership Check ---

	var payload struct {
		Status sqlcqueries.TaskStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// Validate status
	if payload.Status != sqlcqueries.TaskStatusInProgress && payload.Status != sqlcqueries.TaskStatusDone {
		respondWithError(w, http.StatusBadRequest, "Invalid status value: must be 'in_progress' or 'done'")
		return
	}

	params := sqlcqueries.UpdateTaskStatusParams{
		TaskID: int32(taskID),
		Status: payload.Status,
	}

	updatedTask, err := h.dbService.UpdateTaskStatus(r.Context(), params)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedTask)
}



