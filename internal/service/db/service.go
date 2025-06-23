package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"agregator/group/service/vector"
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

func (g *DB) UpdateParsed(id uint64, parsed bool) error {
	query := `
        UPDATE feed 
        SET parsed = $1 
        WHERE id = $2
    `
	_, err := g.conn.Exec(query, parsed, id)
	return err
}

func (g *DB) Insert(t time.Time, feed_id int64, is_rt bool, vec *vector.Vector) (uint64, error) {
	log.Default().Println("Inserting into DB", "time", t, "feed_id", feed_id, "is_rt", is_rt)
	var id uint64
	query := `INSERT INTO groups (time, feed_id, is_rt, embedding)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT(feed_id) DO NOTHING
			RETURNING id`

	err := g.conn.QueryRow(query, t, feed_id, is_rt, vec.ToPqString()).Scan(&id)
	// Проверяем, является ли ошибка sql.ErrNoRows
	if err == sql.ErrNoRows {
		return 0, nil
	}

	// Если это какая-то другая ошибка, или если все прошло успешно, возвращаем ее
	if err != nil {
		return 0, err
	}

	// Если ошибок нет, значит, строка была успешно вставлена, и 'id' содержит новый ID.
	return id, nil
}

func (g *DB) InsertCompares(groupID uint64, compareID uint64) error {
	query := `
        INSERT INTO compares (group_id, feed_id) 
        VALUES ($1, $2)
    `
	_, err := g.conn.Exec(query, groupID, compareID)
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
