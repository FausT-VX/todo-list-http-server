// database/database.go
package database

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/FausT-VX/todo-list-server/models"
	"github.com/jmoiron/sqlx"
)

// type Dbinstance struct {
// 	Db *gorm.DB
// }

// CreateDB - создает базу данных по указанному пути dbPath
func CreateDB(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("func CreateDB. Error: %v", err)
		return err
	}
	defer db.Close()

	// Создание таблицы scheduler и индекса по полю date
	queryCreate := `
	CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date CHAR(8) NOT NULL DEFAULT "",
		title VARCHAR(128) NOT NULL DEFAULT "",
		comment VARCHAR(1000) NOT NULL DEFAULT "",
		repeat VARCHAR(128) NOT NULL DEFAULT ""
	);     
	CREATE INDEX IF NOT EXISTS scheduler_date ON scheduler (date);
	`

	_, err = db.Exec(queryCreate)
	if err != nil {
		log.Printf("func CreateDB. Error creating table: %v", err)
		return err
	}

	return nil
}

// ConnectDB создает подключение к базе данных по указанному пути dbPath
func ConnectDB(dbPath string) (*sqlx.DB, error) {
	// если dbPath не существует, то создаём базу данных по указанному пути dbPath
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		appPath, err := os.Getwd()
		if err != nil {
			log.Fatalf("func ConnectDB. Error: %v", err)
		}
		dbFile = filepath.Join(appPath, dbPath)
	}
	_, err := os.Stat(dbFile)

	var install bool
	if err != nil {
		install = true
	}
	// если install равен true, после открытия БД требуется выполнить
	// sql-запрос с CREATE TABLE и CREATE INDEX
	if install {
		if CreateDB(dbFile) != nil {
			log.Println(err)
			return nil, err
		}
		log.Println("Database has been successfully created")
	} else {
		log.Println("Database already exists")
	}

	db, err := sqlx.Connect("sqlite", dbFile)
	if err != nil {
		return nil, err
	}
	//defer DB.Close()
	return db, nil
}

// GetTaskByID - получение задачи по id
func GetTaskByID(db *sqlx.DB, id int) (models.JsonTask, error) {
	task := models.JsonTask{}
	err := db.Get(&task, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err = errors.New("task not found")
		}
		return models.JsonTask{}, err
	}
	return task, nil
}

// DeleteTaskByID - удаление задачи по id
func DeleteTaskByID(db *sqlx.DB, id int) error {
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("task not found")
	}
	return nil
}
