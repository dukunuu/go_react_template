CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   IF row(NEW.*) IS DISTINCT FROM row(OLD.*) THEN
      NEW.updated_at = CURRENT_TIMESTAMP;
      RETURN NEW;
   ELSE
      RETURN NULL;
   END IF;
END;
$$ language 'plpgsql';

CREATE TABLE courses (
    course_id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    course_name VARCHAR(255) NOT NULL,
    total_points_possible INTEGER NOT NULL CHECK (total_points_possible >= 1 AND total_points_possible <= 100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_courses_user_id ON courses(user_id);

CREATE TRIGGER trigger_courses_updated_at
BEFORE UPDATE ON courses
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

