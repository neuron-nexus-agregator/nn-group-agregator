package db

import (
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"agregator/group/internal/model/db/feed"
	"agregator/group/service/group"
)

type DB struct {
	conn *sqlx.DB
}

func New(maxConnections int) (*DB, error) {
	connectionData := fmt.Sprintf(
		"user=%s dbname=%s sslmode=disable password=%s host=%s port=%s",
		os.Getenv("DB_LOGIN"),
		"newagregator",
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)

	conn, err := sqlx.Connect("postgres", connectionData)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(maxConnections)
	conn.SetMaxIdleConns(maxConnections / 2)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return &DB{conn: conn}, nil
}

func (g *DB) UpdateParsedBatch(ids []uint64, parsed bool) error {
	query := `
        UPDATE feed 
        SET parsed = $1 
        WHERE id = ANY($2)
    `
	_, err := g.conn.Exec(query, parsed, ids)
	return err
}

func (g *DB) Get() ([]feed.Model, error) {
	var feeds []feed.Model
	query := `
        SELECT * 
        FROM feed 
        WHERE parsed = false 
        ORDER BY id DESC
    `
	err := g.conn.Select(&feeds, query)
	return feeds, err
}

func (g *DB) UpdateParsed(id uint64, parsed bool) error {
	query := `
        UPDATE feed 
        SET parsed = $1 
        WHERE id = $2
    `
	_, err := g.conn.Exec(query, parsed, id)
	return err
}

func (g *DB) Insert(group *group.Group) (uint64, error) {
	var id uint64
	query := `
        INSERT INTO groups (time, feed_id, is_rt, embedding) 
        VALUES ($1, $2, $3, $4) 
        RETURNING id
    `

	err := g.conn.QueryRow(query, group.Date, group.Content_id, group.IsTatarstan(), group.Centroid().ToPqString()).Scan(&id)
	return id, err
}

func (g *DB) InsertCompares(groupID uint64, compareID uint64) error {
	query := `
        INSERT INTO compares (group_id, feed_id) 
        VALUES ($1, $2)
    `
	_, err := g.conn.Exec(query, groupID, compareID)
	return err
}

func (g *DB) UpdateDate(groupID uint64, date time.Time, feedID uint64) error {
	query := `
        UPDATE groups 
        SET feed_id = $1 
        WHERE id = $2
    `
	_, err := g.conn.Exec(query, feedID, groupID)
	return err
}

func (g *DB) UpdateRT(groupID uint64, isRT bool) error {
	query := `
        UPDATE groups 
        SET is_rt = $1 
        WHERE id = $2
    `
	_, err := g.conn.Exec(query, isRT, groupID)
	return err
}

func (g *DB) GetRTWords() ([]string, error) {
	var words []string
	query := `
        SELECT word 
        FROM rt_words
    `
	err := g.conn.Select(&words, query)
	return words, err
}

func (g *DB) UpdateEmbedding(id uint64, pqVec string) error {
	query := `
        UPDATE groups 
        SET embedding = $1 
        WHERE id = $2
    `
	_, err := g.conn.Exec(query, pqVec, id)
	return err
}
