// models/models.go
package models

// type Task struct { //сначала сделал через gorm, но не смог подогнать под тесты, потому что gorm добавляет в таблицу свои поля
// 	gorm.Model
// 	ID      uint   `json:"id"      gorm:"primaryKey"`
// 	Date    string `json:"date"    gorm:"type:varchar(8);not null;default:'';index"`
// 	Title   string `json:"title"   gorm:"type:varchar(128);not null;default:''"`
// 	Comment string `json:"comment" gorm:"type:varchar(1000);not null;default:''"`
// 	Repeat  string `json:"repeat"  gorm:"type:varchar(128);not null;default:''"`
// }
type Task struct {
	ID      int64  `json:"id"      db:"id"`
	Date    string `json:"date"    db:"date"`
	Title   string `json:"title"   db:"title"`
	Comment string `json:"comment" db:"comment"`
	Repeat  string `json:"repeat"  db:"repeat"`
}

// TableName задает имя таблицы для структуры Task.
func (Task) TableName() string {
	return "scheduler"
}

// структура задачи для возврата в формате json
type JsonTask struct {
	ID      string `json:"id"      db:"id"`
	Date    string `json:"date"    db:"date"`
	Title   string `json:"title"   db:"title"`
	Comment string `json:"comment" db:"comment"`
	Repeat  string `json:"repeat"  db:"repeat"`
}
