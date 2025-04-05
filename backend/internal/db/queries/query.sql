-- query.sql

-- =============================================
-- Users
-- =============================================

-- name: CreateUser :one
INSERT INTO users (
    email,
    password_hash
) VALUES (
    sqlc.arg(email),
    sqlc.arg(password_hash)
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE user_id = sqlc.arg(user_id)
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = sqlc.arg(email)
LIMIT 1;


-- =============================================
-- Courses
-- =============================================

-- name: CreateCourse :one
INSERT INTO courses (
    user_id,
    course_name,
    total_points_possible
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(course_name),
    sqlc.arg(total_points_possible)
)
RETURNING *;

-- name: GetCourseByID :one
-- Fetch a specific course only if it belongs to the specified user
SELECT * FROM courses
WHERE course_id = sqlc.arg(course_id) AND user_id = sqlc.arg(user_id)
LIMIT 1;

-- name: ListCoursesByUser :many
-- Lists all courses belonging to a specific user
SELECT * FROM courses
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC;

-- name: DeleteCourse :exec
-- Deletes a course only if it belongs to the specified user
DELETE FROM courses
WHERE course_id = sqlc.arg(course_id) AND user_id = sqlc.arg(user_id);

-- name: UpdateCourse :one
-- Updates course details, ensuring ownership
UPDATE courses
SET
    course_name = sqlc.arg(course_name),
    total_points_possible = sqlc.arg(total_points_possible)
WHERE
    course_id = sqlc.arg(course_id) AND user_id = sqlc.arg(user_id)
RETURNING *;


-- =============================================
-- Tasks
-- =============================================

-- name: CreateTask :one
INSERT INTO tasks (
    course_id,
    title,
    due_date,
    points_value
    -- status defaults to 'in_progress'
) VALUES (
    sqlc.arg(course_id),
    sqlc.arg(title),
    sqlc.arg(due_date),
    sqlc.arg(points_value)
)
RETURNING *;

-- name: GetTaskByID :one
-- Fetches a single task by its ID. Ownership should be checked in application logic
-- by verifying the associated course_id belongs to the user.
SELECT * FROM tasks
WHERE task_id = sqlc.arg(task_id)
LIMIT 1;

-- name: ListTasksByCourse :many
-- Lists all tasks for a specific course, ordered by due date
-- Ownership must be verified in application logic before calling this (e.g., ensure the user owns the course_id)
SELECT * FROM tasks
WHERE course_id = sqlc.arg(course_id)
ORDER BY due_date ASC;

-- name: UpdateTaskStatus :one
-- Updates the status of a specific task. Ownership should be checked first.
UPDATE tasks
SET status = sqlc.arg(status)
WHERE task_id = sqlc.arg(task_id)
RETURNING *;

-- name: UpdateTaskDetails :one
-- Updates details of a specific task. Ownership should be checked first.
UPDATE tasks
SET
    title = sqlc.arg(title),
    due_date = sqlc.arg(due_date),
    points_value = sqlc.arg(points_value),
    status = sqlc.arg(status)
WHERE
    task_id = sqlc.arg(task_id)
RETURNING *;


-- name: DeleteTask :exec
-- Deletes a specific task. Ownership should be checked first.
DELETE FROM tasks
WHERE task_id = sqlc.arg(task_id);


-- =============================================
-- Progress & Reminders
-- =============================================

-- name: GetEarnedPointsForCourse :one
-- Calculates the sum of points for tasks marked 'done' for a specific course.
-- Ownership must be verified in application logic before calling this.
SELECT COALESCE(SUM(points_value), 0)::INTEGER AS earned_points
FROM tasks
WHERE course_id = sqlc.arg(course_id) AND status = 'done';

-- name: ListUpcomingTasksByUser :many
-- Lists tasks that are 'in_progress' and due between now and a specified future date for a given user.
-- Includes course name for context.
SELECT
    t.task_id,
    t.course_id,
    t.title,
    t.due_date,
    t.points_value,
    t.status,
    t.created_at,
    t.updated_at,
    c.course_name
FROM tasks t
JOIN courses c ON t.course_id = c.course_id
WHERE
    c.user_id = sqlc.arg(user_id)
    AND t.status = 'in_progress'
    AND t.due_date >= NOW()
    AND t.due_date <= sqlc.arg(upcoming_limit_date)
ORDER BY t.due_date ASC;

-- name: ListCoursesWithEarnedPoints :many
-- Lists all courses for a user along with the total points earned from completed tasks in each course.
SELECT
    c.course_id,
    c.user_id,
    c.course_name,
    c.total_points_possible,
    c.created_at,
    c.updated_at,
    COALESCE(SUM(CASE WHEN t.status = 'done' THEN t.points_value ELSE 0 END), 0)::INTEGER AS earned_points
FROM courses c
LEFT JOIN tasks t ON c.course_id = t.course_id
WHERE c.user_id = sqlc.arg(user_id)
GROUP BY c.course_id
ORDER BY c.created_at DESC;


