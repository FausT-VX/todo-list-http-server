// handlers/handlers.go
package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/FausT-VX/todo-list-server/database"
	"github.com/FausT-VX/todo-list-server/models"
	"github.com/FausT-VX/todo-list-server/service/scheduler"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	Exp      int64  `json:"exp"`
	Checksum string `json:"checksum"`
	jwt.StandardClaims
}

var jwtSecretKey = []byte("very-secret-key")

// NextDateHandler получает следующую дату повторения задачи по переданным в http-запросе параметрам
// now, date, repeat
func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	//Получаем параметры из запроса
	now := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")

	//Вычисляем следующую дату
	nextDate, err := scheduler.NextDate(now, date, repeat)
	if err != nil {
		//http.Error(w, errorJSON(err), http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(""))
		return
	}
	log.Printf("NextDateHandler: %v %v %v %v\n", now, date, repeat, nextDate)

	//Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(nextDate) /*jsonResp*/)
}

// GetTasks обработчик возвращает все задачи из БД в формате списка JSON либо,
// при наличии параметра search, возвращает задачи по переданным параметрам
func GetTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		err := errors.New("method not supported")
		http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
		return
	}

	type params struct {
		Date   string `db:"date"`
		Search string `db:"search"`
		Limit  int    `db:"limit"`
	}
	var (
		err  error
		date time.Time
		args params
	)
	query := ""

	// в зависимости от наличия и значения параметра search задаем соответствующий запрос и определяем его параметры
	search := r.URL.Query().Get("search")
	if search != "" {
		if date, err = time.Parse("02.01.2006", search); err == nil {
			query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = :date LIMIT :limit"
			args = params{Date: date.Format("20060102"), Limit: 50}
		} else {
			query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE :search OR comment LIKE :search ORDER BY date LIMIT :limit"
			args = params{Search: "%" + search + "%", Limit: 50}
		}
	} else {
		query = "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT 50"
		args = params{Limit: 50}
	}

	log.Printf("Handler GetTasks: search = %v; args = %v\n", search, args)
	tasks := []models.JsonTask{}
	task := models.JsonTask{}
	// Выполняем подготовленный запрос
	rows, err := database.DB.NamedQuery(query, args)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		err = rows.StructScan(&task)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	response := map[string][]models.JsonTask{"tasks": tasks}
	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusInternalServerError)
		return
	}
	//log.Printf("Handler GetTasks: result = %v\n", response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// GetTaskByID обработчик возвращает задачу по переданному ID
func GetTaskByID(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if strings.TrimSpace(idParam) == "" {
		err := errors.New("task ID not specified")
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	task, err := database.GetTaskByID(id)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusInternalServerError)
		return
	}
	log.Printf("Handler GetTaskByID: id = %v; task = %v\n", id, task)

	resp, err := json.MarshalIndent(&task, "", "  ")
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// PostTask обработчик создает новую задачу по переданным в http-запросе параметрам,
// записывая в БД с переданными параметрами и записывает в БД
func PostTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		err := errors.New("method not supported")
		http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
		return
	}

	task := new(models.Task)
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	log.Printf("Handler PostTask: task = %v\n", task)

	// проверяем корректность переданных параметров title, date, repeat, и корректируем при необходимости
	if task.Title == "" {
		err := errors.New("task title not specified")
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		log.Println(err)
		return
	}

	date := strings.TrimSpace(task.Date)
	now := time.Now().Format("20060102")
	nextDate := ""

	if len(date) == 0 {
		task.Date = now
	} else {
		begDate, err := time.Parse("20060102", date)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		if begDate.Before(time.Now()) {
			if repeat := strings.TrimSpace(task.Repeat); repeat == "" {
				task.Date = now
			} else {
				nextDate, err = scheduler.NextDate(now, date, task.Repeat)
				if err != nil {
					http.Error(w, errorJSON(err), http.StatusBadRequest)
					return
				}
				task.Date = nextDate
			}
		}
	}

	resultDB, err := database.DB.NamedExec("INSERT INTO scheduler (date, title, comment, repeat) VALUES (:date, :title, :comment, :repeat)", &task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Получаем ID последней вставленной записи, формируем JSON, в формате {"id":"186"} и отправляем ответ
	lastID, err := resultDB.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type TaskID struct {
		ID int64 `json:"id"`
	}
	taskID := TaskID{ID: lastID}
	//log.Printf("Handler PostTaskDone: result = %v\n", taskID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(taskID)
}

// PostTaskDone обработчик удаляет задачу по переданному ID если не задано правило повторения
// либо обновляет дату следующего повторения по правилу указанному в задаче
func PostTaskDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		err := errors.New("method not supported")
		http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("id")
	if strings.TrimSpace(idParam) == "" {
		err := errors.New("task ID not specified")
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	// получаем задачу из БД по ID
	task, err := database.GetTaskByID(id)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusInternalServerError)
		return
	}
	log.Printf("Handler PostTaskDone: id = %v; task = %v\n", id, task)

	if strings.TrimSpace(task.Repeat) == "" {
		if err := database.DeleteTaskByID(id); err != nil {
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
			return
		}
	}
	// получаем новую дату повторения задачи и записываем в базу
	now := time.Now().Add(time.Hour * 25).Format("20060102")
	task.Date, err = scheduler.NextDate(now, task.Date, task.Repeat)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusInternalServerError)
		return

	}

	_, err = database.DB.NamedExec("UPDATE scheduler SET date = :date, title = :title, comment = :comment, repeat = :repeat WHERE id = :id", &task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//log.Printf("Handler PutTask: result = %v\n", task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("{}"))
}

// PutTask обработчик обновляет задачу переданными в json данными, получая ее из базы по ID
func PutTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		err := errors.New("method not supported")
		http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
		return
	}

	task := new(models.JsonTask)
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}
	log.Printf("Handler PutTask: task = %v\n", task)

	idTask := strings.TrimSpace(task.ID)
	if idTask == "" {
		err := errors.New("task ID not specified")
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idTask)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}
	// Проверяем, существует ли задача с указанным ID в базе
	if _, err := database.GetTaskByID(id); err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}
	// проверяем корректность переданных параметров title, date, repeat, и корректируем при необходимости
	if task.Title == "" {
		err := errors.New("task title not specified")
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		log.Println(err)
		return
	}

	date := strings.TrimSpace(task.Date)
	now := time.Now().Format("20060102")
	nextDate := ""
	if repeat := strings.TrimSpace(task.Repeat); repeat != "" {
		nextDate, err = scheduler.NextDate(now, date, task.Repeat)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
	}

	if len(date) == 0 {
		task.Date = now
	} else {
		begDate, err := time.Parse("20060102", date)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		if begDate.Before(time.Now()) {
			if repeat := strings.TrimSpace(task.Repeat); repeat == "" {
				task.Date = now
			} else {
				task.Date = nextDate
			}
		}
	}

	_, err = database.DB.NamedExec("UPDATE scheduler SET date = :date, title = :title, comment = :comment, repeat = :repeat WHERE id = :id", &task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("{}"))
}

// DeleteTask обработчик удаляет задачу из базы по ID
func DeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		err := errors.New("method not supported")
		http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("id")
	if strings.TrimSpace(idParam) == "" {
		err := errors.New("task ID not specified")
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, errorJSON(err), http.StatusBadRequest)
		return
	}
	log.Printf("Handler DeleteTask: id = %v\n", id)
	if err := database.DeleteTaskByID(id); err != nil {
		http.Error(w, errorJSON(err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("{}"))
}

type Credentials struct {
	Password string `json:"password"`
}

type Response struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

// AuthHandler обработчик аутентификации пользователя по паролю
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "Invalid request payload"})
		return
	}

	expectedPassword := os.Getenv("TODO_PASSWORD")
	if creds.Password != expectedPassword {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{Error: "Invalid password"})
		return
	}

	hash := sha256.Sum256([]byte(creds.Password))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		//"exp":      time.Now().Add(time.Hour * 8).Unix(),
		"checksum": fmt.Sprintf("%x", hash),
	})

	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: "Failed to generate token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{Token: tokenString})
}

// AuthMiddleware обработчик аутентификации пользователя по токену из куки
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// смотрим наличие пароля
		pass := os.Getenv("TODO_PASSWORD")
		if len(pass) > 0 {
			var tokenString string // JWT-токен из куки
			// получаем куку
			cookie, err := r.Cookie("token")
			if err == nil {
				tokenString = cookie.Value
			}
			// здесь код для валидации и проверки JWT-токена
			jwtToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return jwtSecretKey, nil
			})
			if err != nil || !jwtToken.Valid {
				http.Error(w, "Authentification required", http.StatusUnauthorized)
				return
			}
			// проверяем payload
			claims, ok := jwtToken.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Authentification required", http.StatusUnauthorized)
				return
			}
			// проверяем контрольную сумму пароля
			hash := sha256.Sum256([]byte(pass))
			checksum := fmt.Sprintf("%x", hash)
			if claims["checksum"] != checksum {
				http.Error(w, "Authentification required", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// errorJSON возвращает json-строку с ошибкой
func errorJSON(err error) string {
	jsonError, err := json.Marshal(map[string]string{"error": err.Error()})
	if err != nil {
		println(err)
		return ""
	}
	return string(jsonError)
}
