package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Postgres struct {
	conn *pgx.Conn
}

func NewConnection(url string) (*Postgres, error) {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}
	return &Postgres{conn: conn}, nil
}

func (p *Postgres) UpdateStatus(id int, status string, metadataJSON string) error {
	query := `
		UPDATE products 
		SET processing_status = $1, 
            title = CASE WHEN $2::json->>'title' IS NOT NULL THEN $2::json->>'title' ELSE title END,
            description = CASE WHEN $2::json->>'description' IS NOT NULL THEN $2::json->>'description' ELSE description END,
            taxonomy = CASE WHEN $2::json->>'taxonomy' IS NOT NULL THEN $2::json->>'taxonomy' ELSE taxonomy END,
            violations = CASE WHEN $2::json->>'violations' IS NOT NULL THEN ($2::json->'violations')::jsonb ELSE violations END,
            error_message = CASE WHEN $2::json->>'error_message' IS NOT NULL THEN $2::json->>'error_message' ELSE error_message END,
			updated_at = NOW()
		WHERE id = $3
	`

	_, err := p.conn.Exec(context.Background(), query, status, metadataJSON, id)
	return err
}
