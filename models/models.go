// models/models.go
package models

type Task struct {
	ID      string `json:"id"      db:"id,omitempty"`
	Date    string `json:"date"    db:"date"`
	Title   string `json:"title"   db:"title"`
	Comment string `json:"comment" db:"comment"`
	Repeat  string `json:"repeat"  db:"repeat"`
}
