package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Task struct {
	id    int    `json:"id"`
	title string `json:"title"`
	done  bool   `json:"done"`
}

var tasks []Task
var currentID int

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", GetTasks)  //get
	mux.HandleFunc("/task", CreateTask) //post
	err := http.ListenAndServe("8080", nil)
	if err != nil {
		log.Fatal("Проблема запуска сервера")
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
if r.Method != "GET"(
	http.Error(w, "не тот метод", http.Star)
)
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
