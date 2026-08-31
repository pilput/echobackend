-- +goose Up
ALTER TABLE corporate_actions
    ADD COLUMN IF NOT EXISTS cum_date DATE,
    ADD COLUMN IF NOT EXISTS rec_date DATE;

-- +goose Down
ALTER TABLE corporate_actions
    DROP COLUMN IF EXISTS cum_date,
    DROP COLUMN IF EXISTS rec_date;
