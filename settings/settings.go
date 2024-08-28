package settings

import "os"

// Настройки по умолчанию
const (
	DateFormat = "20060102"
	DBPath     = "./scheduler.db" // Путь к базе данных
	Port       = ":7540"          // Порт сервера
	WebDir     = "./web"          // Директория для web файлов
)

// Лимиты на получение строк в SQL-запросах
const (
	Limit50 int = 50
)

var EnvDBFile = os.Getenv("TODO_DBFILE") // Файл БД из переменной окружения TODO_DBFILE
var EnvPort = os.Getenv("TODO_PORT")     // Порт из переменной окружения TODO_PORT
var EnvPass = os.Getenv("TODO_PASSWORD") // Пароль из переменной окружения TODO_PASSWORD

var JwtSecretKey = []byte("very-secret-key")
