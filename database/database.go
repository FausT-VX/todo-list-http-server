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

var DB *sqlx.DB //Dbinstance

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
func ConnectDB(dbPath string) error {
	// если dbPath не существует, то создаём базу данных по указанному пути dbPath
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		appPath, err := os.Executable()
		if err != nil {
			log.Fatalf("func ConnectDB. Error: %v", err)
		}
		dbFile = filepath.Join(filepath.Dir(appPath), dbPath)
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
			return err
		}
		log.Println("Database has been successfully created")
	} else {
		log.Println("Database already exists")
	}

	DB, err = sqlx.Connect("sqlite", dbFile)
	if err != nil {
		return err
	}
	//defer DB.Close()
	return nil
}

// GetTaskByID - получение задачи по id
func GetTaskByID(id int) (models.JsonTask, error) {
	task := models.JsonTask{}
	err := DB.Get(&task, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err = errors.New("task not found")
		}
		return models.JsonTask{}, err
	}
	return task, nil
}

// DeleteTaskByID - удаление задачи по id
func DeleteTaskByID(id int) error {
	_, err := GetTaskByID(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

//
// func ConnectDB(dbPath string) {
// 	// создаём подключение к базе данных. В &gorm.Config настраивается логер,
// 	// который будет сохранять информацию обо всех активностях с базой данных.
// 	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
// 		Logger: logger.Default.LogMode(logger.Info),
// 	})

// 	if err != nil {
// 		log.Fatal("Failed to connect to database.\n", err)
// 		os.Exit(1)
// 	}

// 	log.Println("connected")
// 	db.Logger = logger.Default.LogMode(logger.Info)

// 	log.Println("running migration")
// 	db.AutoMigrate(&models.Task{})

// 	DB = Dbinstance{
// 		Db: db,
// 	}
// }
