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
	"strconv"
	"strings"
	"time"

	"github.com/FausT-VX/todo-list-server/database"
	"github.com/FausT-VX/todo-list-server/models"
	"github.com/FausT-VX/todo-list-server/service/scheduler"
	"github.com/FausT-VX/todo-list-server/settings"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	//Exp      int64  `json:"exp"`
	Checksum string `json:"checksum"`
	jwt.StandardClaims
}

type TaskID struct {
	ID int64 `json:"id"`
}

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
		log.Printf("NextDateHandler: %v %v %v %v; error: %v\n", now, date, repeat, nextDate, err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(""))
		return
	}

	//Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(nextDate) /*jsonResp*/)
}

// GetTasks обработчик возвращает все задачи из БД в формате списка JSON либо,
// при наличии параметра search, возвращает задачи по переданным параметрам
func GetTasks(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			err := errors.New("method not supported")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		search := r.URL.Query().Get("search")
		tasks, err := store.GetTasks(search)
		if err != nil {
			log.Printf("Handler GetTasks: search = %v; err = %v\n", search, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
		}

		response := map[string][]models.Task{"tasks": tasks}
		jsonResponse, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonResponse)
	}
}

// GetTaskByID обработчик возвращает задачу по переданному ID
func GetTaskByID(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idParam := r.URL.Query().Get("id")
		if strings.TrimSpace(idParam) == "" {
			err := errors.New("task ID not specified")
			log.Printf("Handler GetTaskByID: id = %v; error = %v\n", idParam, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}
		task, err := store.GetTaskByID(id)
		if err != nil {
			log.Printf("Handler GetTaskByID: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		resp, err := json.MarshalIndent(&task, "", "  ")
		if err != nil {
			log.Printf("Handler GetTaskByID: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}
}

// PostTask обработчик создает новую задачу по переданным в http-запросе параметрам,
// записывая в БД с переданными параметрами и записывает в БД
func PostTask(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			err := errors.New("method not supported")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		task := models.Task{}
		var buf bytes.Buffer

		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			log.Printf("Handler PostTask: buf = %v; error = %v\n", buf, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
			log.Printf("Handler PostTask: task = %v; error = %v\n", task, err)
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
		now := time.Now().Format(settings.DateFormat)
		nextDate := ""

		if len(date) == 0 {
			task.Date = now
		} else {
			begDate, err := time.Parse(settings.DateFormat, date)
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

		lastID, err := store.InsertTask(task)
		if err != nil {
			log.Printf("Handler PostTask: task = %v; error = %v\n", task, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Формируем JSON, в формате {"id":"186"} и отправляем ответ
		taskID := TaskID{ID: lastID}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(taskID)
	}
}

// PostTaskDone обработчик удаляет задачу по переданному ID если не задано правило повторения
// либо обновляет дату следующего повторения по правилу указанному в задаче
func PostTaskDone(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		task, err := store.GetTaskByID(id)
		if err != nil {
			log.Printf("Handler PostTaskDone: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		if strings.TrimSpace(task.Repeat) == "" {

			if err := store.DeleteTaskByID(id); err != nil {
				log.Printf("Handler PostTaskDone: id = %v; task = %v; error = %v\n", id, task, err)
				http.Error(w, errorJSON(err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}
		// получаем новую дату повторения задачи и записываем в базу
		now := time.Now().Add(time.Hour * 25).Format(settings.DateFormat)
		task.Date, err = scheduler.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return

		}

		err = store.UpdateTask(task)
		if err != nil {
			log.Printf("Handler PostTaskDone: id = %v; task = %v; error = %v\n", id, task, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}
}

// PutTask обработчик обновляет задачу переданными в json данными, получая ее из базы по ID
func PutTask(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			err := errors.New("method not supported")
			http.Error(w, errorJSON(err), http.StatusMethodNotAllowed)
			return
		}

		task := models.Task{}
		var buf bytes.Buffer

		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

		if err = json.Unmarshal(buf.Bytes(), &task); err != nil {
			log.Printf("Handler PutTask: task = %v; error = %v\n", task, err)
			http.Error(w, errorJSON(err), http.StatusBadRequest)
			return
		}

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
		if _, err := store.GetTaskByID(id); err != nil {
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
		now := time.Now().Format(settings.DateFormat)
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
			begDate, err := time.Parse(settings.DateFormat, date)
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

		err = store.UpdateTask(task)
		if err != nil {
			log.Printf("Handler PutTask: task = %v; error = %v\n", task, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}
}

// DeleteTask обработчик удаляет задачу из базы по ID
func DeleteTask(store database.TasksStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if err := store.DeleteTaskByID(id); err != nil {
			log.Printf("Handler DeleteTask: id = %v\n, error = %v\n", id, err)
			http.Error(w, errorJSON(err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}
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

	if creds.Password != settings.EnvPass {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{Error: "Invalid password"})
		return
	}

	hash := sha256.Sum256([]byte(creds.Password))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		//"exp":      time.Now().Add(time.Hour * 8).Unix(),
		"checksum": fmt.Sprintf("%x", hash),
	})

	tokenString, err := token.SignedString(settings.JwtSecretKey)
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
		if len(settings.EnvPass) > 0 {
			var tokenString string // JWT-токен из куки
			// получаем куку
			cookie, err := r.Cookie("token")
			if err == nil {
				tokenString = cookie.Value
			}
			// здесь код для валидации и проверки JWT-токена
			jwtToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return settings.JwtSecretKey, nil
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
			hash := sha256.Sum256([]byte(settings.EnvPass))
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
