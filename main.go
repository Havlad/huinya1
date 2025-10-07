package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Task struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

var db *sql.DB

// Конфигурация базы данных
const (
	dbHost     = "localhost"
	dbPort     = "5432"
	dbUser     = "postgres"
	dbPassword = "123"
	dbName     = "postgres"
	sslMode    = "disable"
)

func main() {
	// Подключение к БД
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	// Проверка подключения
	err = db.Ping()
	if err != nil {
		log.Fatal("Нет подключения к БД:", err)
	}

	// Обработка команд миграции
	if len(os.Args) > 1 {
		handleMigrationCommand(os.Args[1])
		return
	}

	// Запуск HTTP-сервера
	startServer()
}

// Обработка команд миграции
func handleMigrationCommand(command string) {
	migrationDir := "./migrations"

	switch command {
	case "migrate-up":
		runGooseCommand("up", migrationDir)
	case "migrate-down":
		runGooseCommand("down", migrationDir)
	case "migrate-status":
		runGooseCommand("status", migrationDir)
	case "migrate-create":
		if len(os.Args) < 3 {
			log.Fatal("Для создания миграции укажите название: go run main.go migrate-create <name>")
		}
		createMigration(os.Args[2])
	default:
		log.Printf("Неизвестная команда: %s", command)
		log.Printf("Доступные команды:")
		log.Printf("  migrate-up     - Применить все миграции")
		log.Printf("  migrate-down   - Откатить последнюю миграцию")
		log.Printf("  migrate-status - Показать статус миграций")
		log.Printf("  migrate-create <name> - Создать новую миграцию")
	}
}

// Запуск команд Goose
func runGooseCommand(command string, migrationDir string) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	cmd := exec.Command("goose", "-dir", migrationDir, "postgres", connStr, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Выполнение команды: goose %s", command)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Ошибка выполнения миграции: %v", err)
	}
}

// Создание новой миграции
func createMigration(name string) {
	cmd := exec.Command("goose", "-dir", "migrations", "create", name, "sql")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Создание миграции: %s", name)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Ошибка создания миграции: %v", err)
	}
}

// Запуск HTTP-сервера
func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", tasksHandler)
	mux.HandleFunc("/tasks/", taskByIDHandler)

	fmt.Println("Сервер запущен на порту 8080")
	fmt.Println("Доступные endpoints:")
	fmt.Println("  GET    /tasks     - Получить все задачи")
	fmt.Println("  POST   /tasks     - Создать новую задачу")
	fmt.Println("  GET    /tasks/{id} - Получить задачу по ID")
	fmt.Println("  PUT    /tasks/{id} - Обновить задачу")
	fmt.Println("  DELETE /tasks/{id} - Удалить задачу")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

// Обработчик для /tasks
func tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getTasks(w, r)
	case http.MethodPost:
		createTask(w, r)
	default:
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
	}
}

// Обработчик для /tasks/{id}
func taskByIDHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Неверный URL", http.StatusBadRequest)
		return
	}

	id := parts[2]
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "Неверный UUID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getTaskByID(w, r, id)
	case http.MethodPut:
		updateTask(w, r, id)
	case http.MethodDelete:
		deleteTask(w, r, id)
	default:
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
	}
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, done FROM tasks")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Done); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, "Title не может быть пустым", http.StatusBadRequest)
		return
	}

	task.ID = uuid.New().String()
	task.Done = false // Всегда создаем как невыполненную

	_, err := db.Exec("INSERT INTO tasks (id, title, done) VALUES ($1, $2, $3)",
		task.ID, task.Title, task.Done)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func getTaskByID(w http.ResponseWriter, r *http.Request, id string) {
	var task Task
	err := db.QueryRow("SELECT id, title, done FROM tasks WHERE id = $1", id).
		Scan(&task.ID, &task.Title, &task.Done)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Задача не найдена", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func updateTask(w http.ResponseWriter, r *http.Request, id string) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, "Title не может быть пустым", http.StatusBadRequest)
		return
	}

	task.ID = id

	result, err := db.Exec("UPDATE tasks SET title = $1, done = $2 WHERE id = $3",
		task.Title, task.Done, task.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func deleteTask(w http.ResponseWriter, r *http.Request, id string) {
	result, err := db.Exec("DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
