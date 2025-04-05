package handlers

import (
	"encoding/json"
	"go-react-template/backend/internal/auth"
	"go-react-template/backend/internal/db"
	"go-react-template/backend/pkg/sqlcqueries"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CourseHandler struct {
	dbService *db.DBService
}

func NewCourseHandler(dbService *db.DBService) *CourseHandler {
	return &CourseHandler{dbService: dbService}
}

// POST /api/v1/courses
func (h *CourseHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(firebaseUIDStr) // Convert Firebase UID string to UUID
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	var params sqlcqueries.CreateCourseParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// Basic validation (add more as needed)
	if params.CourseName == "" || params.TotalPointsPossible <= 0 || params.TotalPointsPossible > 100 {
		respondWithError(w, http.StatusBadRequest, "Invalid input: course name required, points must be between 1 and 100")
		return
	}

	params.UserID = userID // Set the user ID from the authenticated user

	course, err := h.dbService.CreateCourse(r.Context(), params)
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusCreated, course)
}

// GET /api/v1/courses
func (h *CourseHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(firebaseUIDStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	// Use the query that includes earned points
	coursesWithPoints, err := h.dbService.Queries.ListCoursesWithEarnedPoints(r.Context(), sqlcqueries.ListCoursesWithEarnedPointsParams{UserID: userID})
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	// If the list is empty, return an empty array, not an error
	if coursesWithPoints == nil {
		coursesWithPoints = []*sqlcqueries.ListCoursesWithEarnedPointsRow{}
	}

	respondWithJSON(w, http.StatusOK, coursesWithPoints)
}

// GET /api/v1/courses/{courseID}
func (h *CourseHandler) GetCourse(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(firebaseUIDStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	courseIDStr := chi.URLParam(r, "courseID")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid course ID format")
		return
	}

	params := sqlcqueries.GetCourseByIDParams{
		CourseID: int32(courseID),
		UserID:   userID, // Ensure the user owns this course
	}

	course, err := h.dbService.Queries.GetCourseByID(r.Context(), params)
	if err != nil {
		code, msg := mapDBError(err) // Handles ErrNoRows as 404 Not Found
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusOK, course)
}

// PUT /api/v1/courses/{courseID}
func (h *CourseHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(firebaseUIDStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	courseIDStr := chi.URLParam(r, "courseID")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid course ID format")
		return
	}

	var params sqlcqueries.UpdateCourseParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// Basic validation
	if params.CourseName == "" || params.TotalPointsPossible <= 0 || params.TotalPointsPossible > 100 {
		respondWithError(w, http.StatusBadRequest, "Invalid input: course name required, points must be between 1 and 100")
		return
	}

	params.CourseID = int32(courseID)
	params.UserID = userID // Set user ID for ownership check in the query

	updatedCourse, err := h.dbService.Queries.UpdateCourse(r.Context(), params)
	if err != nil {
		code, msg := mapDBError(err) // Handles ErrNoRows if course doesn't exist or user doesn't own it
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedCourse)
}

// DELETE /api/v1/courses/{courseID}
func (h *CourseHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	firebaseUIDStr, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(firebaseUIDStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	courseIDStr := chi.URLParam(r, "courseID")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 32)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid course ID format")
		return
	}

	params := sqlcqueries.DeleteCourseParams{
		CourseID: int32(courseID),
		UserID:   userID, // Ensure the user owns this course for deletion
	}

	err = h.dbService.Queries.DeleteCourse(r.Context(), params)
	// Note: sqlc DeleteCourse returns :exec, so it doesn't return ErrNoRows
	// if the course wasn't found or didn't belong to the user.
	// The operation simply does nothing in that case.
	// If you need to confirm deletion happened, you might need a different query or check affected rows.
	if err != nil {
		code, msg := mapDBError(err)
		respondWithError(w, code, msg)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Course deleted successfully"})
}

