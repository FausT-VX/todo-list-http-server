// database/database.go
package database

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FausT-VX/todo-list-server/models"
	"github.com/FausT-VX/todo-list-server/settings"
	"github.com/jmoiron/sqlx"
)

type TasksStore struct {
	db *sqlx.DB
}

func NewTasksStore(db *sqlx.DB) TasksStore {
	return TasksStore{db: db}
}

// параметры для запросов
type params struct {
	Date   string `db:"date"`
	Search string `db:"search"`
	Limit  int    `db:"limit"`
}

var info = log.New(os.Stdout, "todo-server INF: ", log.Ldate|log.Ltime)

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
	dbFile := settings.EnvDBFile
	dbFile = strings.TrimPrefix(dbFile, ".")
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
			return nil, err
		}
		info.Println("Database has been successfully created")
	} else {
		info.Println("Database already exists")
	}

	db, err := sqlx.Connect("sqlite", dbFile)
	if err != nil {
		return nil, err
	}
	//defer DB.Close()
	return db, nil
}

// GetTaskByID - получение задачи по id
func (s TasksStore) GetTaskByID(id int) (models.Task, error) {
	task := models.Task{}
	err := s.db.Get(&task, "SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			err = errors.New("task not found")
		}
		return models.Task{}, err
	}
	return task, nil
}

// DeleteTaskByID - удаление задачи по id
func (s TasksStore) DeleteTaskByID(id int) error {
	result, err := s.db.Exec("DELETE FROM scheduler WHERE id = ?", id)
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

// GetTasks - получение всех задач если search = "";
// если search равен строке в формате "02.01.2006", задачи на указанную дату;
// иначе ищет задачи содержащие подстроку search в полях title и comment с учетом регистра;
// во всех случая учитывается ограничение на количество возвращаемых строк database.RowsLimit
func (s TasksStore) GetTasks(search string) ([]models.Task, error) {
	// хотел реализовать бе учета регистра но обнаружил, а потом и нагуглил, что sqlite не поддерживает LOWER() для кириллицы
	var args params
	query := ""

	// в зависимости от наличия и значения параметра search задаем соответствующий запрос и определяем его параметры
	if search != "" {
		if date, err := time.Parse("02.01.2006", search); err == nil {
			query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = :date LIMIT :limit"
			args = params{Date: date.Format(settings.DateFormat), Limit: settings.Limit50}
		} else {
			query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE :search OR comment LIKE :search ORDER BY date LIMIT :limit"
			args = params{Search: "%" + search + "%", Limit: settings.Limit50}
		}
	} else {
		query = "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT :limit"
		args = params{Limit: settings.Limit50}
	}

	tasks := []models.Task{}
	task := models.Task{}
	// Выполняем подготовленный запрос
	rows, err := s.db.NamedQuery(query, args)
	if err != nil {
		return []models.Task{}, err
	}
	for rows.Next() {
		err = rows.StructScan(&task)
		if err != nil {
			return []models.Task{}, err
		}
		tasks = append(tasks, task)
	}
	if err = rows.Err(); err != nil {
		return []models.Task{}, err
	}

	return tasks, nil
}

// UpdateTask - обновление задачи по id
func (s TasksStore) UpdateTask(task models.Task) error {
	_, err := s.db.NamedExec("UPDATE scheduler SET date = :date, title = :title, comment = :comment, repeat = :repeat WHERE id = :id", &task)
	if err != nil {
		return err
	}
	return nil
}

// DeleteTask - удаление задачи по id
func (s TasksStore) InsertTask(task models.Task) (lastInsertId int64, err error) {
	resultDB, err := s.db.NamedExec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (:date, :title, :comment, :repeat)", &task)
	if err != nil {
		return 0, err
	}
	// Получаем ID последней вставленной записи
	lastInsertId, err = resultDB.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastInsertId, nil
}
