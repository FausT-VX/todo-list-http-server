package main

import (
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	"github.com/FausT-VX/todo-list-server/database"
	"github.com/FausT-VX/todo-list-server/handlers"
	"github.com/FausT-VX/todo-list-server/settings"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	infLog := log.New(os.Stdout, "todo-server INF: ", log.Ldate|log.Ltime)
	errLog := log.New(os.Stderr, "todo-server ERR: ", log.Ldate|log.Ltime)
	infLog.Println("Starting application...")

	// Соединение с базой данных
	db, err := database.ConnectDB(settings.DBPath)
	if err != nil {
		errLog.Println(err)
		return
	}
	defer db.Close()
	// инициализация маршрутизатора
	router := chi.NewRouter()

	// файловый сервер
	fs := http.FileServer(http.Dir(settings.WebDir))
	router.Handle("/*", http.StripPrefix("/", fs))
	// обработчики

	// Обработчики API
	apiRouter := chi.NewRouter()
	// Middleware для проверки аутентификации
	//apiRouter.Use(middleware.Logger)
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(handlers.AuthMiddleware)
	apiRouter.Get("/tasks", handlers.GetTasks(db))
	apiRouter.Route("/task", func(r chi.Router) {
		r.Get("/", handlers.GetTaskByID(db))
		r.Post("/", handlers.PostTask(db))
		r.Post("/done", handlers.PostTaskDone(db))
		r.Put("/", handlers.PutTask(db))
		r.Delete("/", handlers.DeleteTask(db))
	})
	router.Mount("/api", apiRouter)
	router.Post("/api/signin", handlers.AuthHandler)
	router.Get("/api/nextdate", handlers.NextDateHandler)

	port := ":" + settings.EnvPort
	if port == ":" {
		port = settings.Port
	}

	infLog.Printf("Starting server on port %s...\n", port)
	if err := http.ListenAndServe(port, router); err != nil {
		errLog.Printf("Start server error: %s", err.Error())
		return
	}
}
