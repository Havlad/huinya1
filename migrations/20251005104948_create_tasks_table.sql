-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
"CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    done BOOLEAN DEFAULT FALSE
);"

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
"DROP TABLE IF EXISTS tasks;"