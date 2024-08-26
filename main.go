package main

import (
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	"github.com/FausT-VX/todo-list-server/database"
	"github.com/FausT-VX/todo-list-server/handlers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

// Порт сервера
const Port = ":7540"

// Директория для web файлов
const WebDir = "./web"

// Путь к базе данных
const DbPath = "./scheduler.db"

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("todo-server: ")
	log.Println("Starting application...")

	// Соединение с базой данных
	db, err := database.ConnectDB(DbPath)
	if err != nil {
		log.Println(err)
		return
	}
	handlers.DBinit(db)
	// инициализация маршрутизатора
	router := chi.NewRouter()

	// файловый сервер
	fs := http.FileServer(http.Dir(WebDir))
	router.Handle("/*", http.StripPrefix("/", fs))
	// обработчики

	// Обработчики API
	apiRouter := chi.NewRouter()
	// Middleware для проверки аутентификации
	//apiRouter.Use(middleware.Logger)
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(handlers.AuthMiddleware)
	apiRouter.Get("/tasks", handlers.GetTasks)
	apiRouter.Route("/task", func(r chi.Router) {
		r.Get("/", handlers.GetTaskByID)
		r.Post("/", handlers.PostTask)
		r.Post("/done", handlers.PostTaskDone)
		r.Put("/", handlers.PutTask)
		r.Delete("/", handlers.DeleteTask)
	})
	router.Mount("/api", apiRouter)
	router.Post("/api/signin", handlers.AuthHandler)
	router.Get("/api/nextdate", handlers.NextDateHandler)
	// router := gin.Default()

	// // файловый сервер
	// fileServer := http.FileServer(http.Dir(WebDir))

	// // Добавляем файловый сервер к роутеру
	// router.GET("/", func(c *gin.Context) {
	// 	c.Request = c.Request.WithContext(c)
	// 	fileServer.ServeHTTP(c.Writer, c.Request)
	// })

	// // обработчики
	// router.POST("/api/signin", handlers.AuthHandler)
	// router.GET("/api/nextdate", handlers.NextDateHandler)

	// // Обработчики API
	// api := router.Group("/api")
	// // Middleware для проверки аутентификации
	// api.Use(gin.Recovery(), handlers.AuthMiddleware())
	// api.GET("/tasks", handlers.GetTasks)
	// task := api.Group("/task")
	// {
	// 	task.GET("/", handlers.GetTaskByID)
	// 	task.POST("/", handlers.PostTask)
	// 	task.POST("/done", handlers.PostTaskDone)
	// 	task.PUT("/", handlers.PutTask)
	// 	task.DELETE("/", handlers.DeleteTask)
	// }

	// // Обработка статических файлов
	// router.Static("/web", WebDir)
	// router.Static("/js", WebDir+"/js")
	// router.Static("/css", WebDir+"/css")
	// router.StaticFile("/favicon.ico", WebDir+"/favicon.ico") // 06служивание index.html по корневому пути
	// router.StaticFile("/login.html", WebDir+"/login.html")
	// router.NoRoute(func(c *gin.Context) {
	// 	c.File(WebDir + "/index.html")
	// })

	port := ":" + os.Getenv("TODO_PORT")
	if port == ":" {
		port = Port
	}

	log.Printf("Starting server on port %s...\n", port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Printf("Start server error: %s", err.Error())
		handlers.DBclose()
		return
	}
	handlers.DBclose()
}
