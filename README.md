# http-сервер списка задач

В директории `tests` находятся тесты для проверки API, которое должно быть реализовано в веб-сервере.
Директория `web` содержит файлы фронтенда.

Сервер работает на url http://localhost:7540/
Все необходимые переменные среды настроены в докер-файле
TODO_PORT=7540 - порт на котором работает сервис
TODO_DBFILE=../scheduler.db - путь к файлу БД
TODO_PASSWORD=123 - пароль

Сборка образа: docker build -t faustvx/todo_server:v1 . 
Запуск контейнера: docker run -p 7540:7540 faustvx/todo_server:v1

Выполнены все задания со звёздочкой.

tests/settings.go настроен для полного тестирования всех заданий, включая задания со звездочкой.
