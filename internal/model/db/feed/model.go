package feed

import "time"

type Model struct {
	ID          uint64    `db:"id"`
	MD5         string    `db:"md5"`
	Time        time.Time `db:"time"`
	Source_Name string    `db:"source_name"`
	Parsed      bool      `db:"parsed"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	FullText    string    `db:"full_text"`
	Link        string    `db:"link"`
	Enclosure   string    `db:"enclosure"`
	Category    *string   `db:"category"`
}
