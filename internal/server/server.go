package server

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/agent"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"gorm.io/gorm"
)

var DEBUG = false
var DB *gorm.DB

type Request struct {
	Images [][]byte `json:"images"`
}

func Initialise(ready chan<- struct{}, port string, isDebug bool) {
	DEBUG = isDebug
	database, err := db.Connect()
	if err != nil {
		fmt.Println("Database error")
		return
	}
	DB = database
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "lightcode is running!")
	})
	http.HandleFunc("GET /list-sessions", listSessions)
	http.HandleFunc("GET /get-session-data", getSessionData)
	http.HandleFunc("GET /chat-completion", chatcompletion)
	http.HandleFunc("POST /send-message", sendMessage)
	http.HandleFunc("POST /create-session", createSession)
	http.HandleFunc("POST /delete-session", deleteSession)
	http.HandleFunc("GET /get-current-todo-list", getCurrentTodoList)
	http.HandleFunc("GET /get-context-size", getContextSize)
	http.HandleFunc("GET /get-available-skills", getAvailableSkills)
	http.HandleFunc("GET /get-current-model", getCurrentModel)
	http.HandleFunc("GET /get-models", getModels)
	// http.ListenAndServe(":8080", nil)

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "address already in use") {
			fmt.Println("Running only tui")
			return
			// close(ready)
		}
		log.Fatal(err)
	}
	close(ready)

	http.Serve(ln, nil)
}

func listSessions(w http.ResponseWriter, r *http.Request) {
	var sessions []models.Session
	DB.Table("sessions").Select("*").Find(&sessions)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func getSessionData(w http.ResponseWriter, r *http.Request) {
	var messages []models.Message
	session_id := r.URL.Query().Get("session_id")
	DB.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	// fmt.Println(messages)
	json.NewEncoder(w).Encode(messages)
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	session_id := r.URL.Query().Get("session_id")
	message := r.URL.Query().Get("message")

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {

	}
	var messages []models.Message
	DB.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	newMessage := models.Message{SessionID: session_id, Data: models.EncodeMessageData(models.StoredMessageData{Role: "user", Content: message})}
	DB.Create(&newMessage)
	json.NewEncoder(w).Encode(newMessage)
}

func deleteSession(w http.ResponseWriter, r *http.Request) {
	session_id := r.URL.Query().Get("session_id")
	DB.Table("messages").Where("session_id = ?", session_id).Delete(&models.Message{})
	DB.Table("sessions").Where("id = ?", session_id).Delete(&models.Session{})
	fmt.Fprint(w, "Session deleted successfully")
}

func createSession(w http.ResponseWriter, r *http.Request) {
	prompt := r.URL.Query().Get("prompt")
	workingDirectory := r.URL.Query().Get("working_directory")
	session_id := randomSessionID()
	session := models.Session{ID: session_id, Title: prompt, Directory: workingDirectory}
	DB.Create(&session)
	fmt.Fprint(w, session_id)
}

func chatcompletion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	session_id := r.URL.Query().Get("session_id")
	prompt := r.URL.Query().Get("prompt")
	mode := r.URL.Query().Get("mode")

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {

	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	for result := range agent.New(DB).Run(r.Context(), prompt, req.Images, session_id, mode, DEBUG) {
		if r.Context().Err() != nil {
			return
		}
		fmt.Fprintf(w, "%s\n", models.EncodeMessageData(result))
		flusher.Flush()
	}
	fmt.Fprintf(w, "[DONE]\n")
	flusher.Flush()
}

func getCurrentTodoList(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, `{"error":"session_id required"}`, http.StatusBadRequest)
		return
	}
	// database, err := db.Connect()
	// if err != nil {
	// 	http.Error(w, `{"error":"database unavailable"}`, http.StatusInternalServerError)
	// 	return
	// }
	var session models.Session
	if err := DB.Where("id = ?", sessionID).First(&session).Error; err != nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}
	todos := models.DecodeToDoList(session.ToDoList)
	if todos == nil {
		todos = []models.ToDo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func getContextSize(w http.ResponseWriter, r *http.Request) {
	// database, _ := db.Connect()
	var messages []models.Message
	session_id := r.URL.Query().Get("session_id")
	DB.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	slices.Reverse(messages)
	var context_size int64
	var afterLastAssistant []models.Message
	for _, m := range messages {
		afterLastAssistant = append(afterLastAssistant, m)
		if models.DecodeMessageData(m.Data).Role == "assistant" {
			context_size = models.DecodeMessageData(m.Data).Usage.PromptTokens + int64(len(models.DecodeMessageData(m.Data).Content)/4)

			break
		}
	}
	for _, m := range afterLastAssistant {
		context_size += int64(len(models.DecodeMessageData(m.Data).Content) / 4)
	}
	fmt.Fprint(w, context_size)
}

func getAvailableSkills(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(config.SkillsPath())
	if err != nil {
		json.NewEncoder(w).Encode([]string{})
		fmt.Println("gone")
		return
	}
	skills := []string{}
	for _, f := range files {
		subFiles, _ := os.ReadDir(config.SkillsPath() + "/" + f.Name())
		for _, subf := range subFiles {
			if strings.Contains(subf.Name(), "SKILL.md") {
				skills = append(skills, f.Name())
				break
			}
		}
	}
	json.NewEncoder(w).Encode(skills)
}

func getCurrentModel(w http.ResponseWriter, r *http.Request) {
	cur_model := config.GetCustomization().CurrentModel
	if cur_model.Model != "" {
		json.NewEncoder(w).Encode(cur_model)
	}
}

func getModels(w http.ResponseWriter, r *http.Request) {
	list_models, recent_models, err := config.GetModels()
	if err != nil {

	}
	payload := config.AllModels{
		Models:       list_models,
		RecentModels: recent_models,
	}
	json.NewEncoder(w).Encode(payload)
}

func randomSessionID() string {
	var chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890-_"
	length := 10
	var result strings.Builder
	for range length {
		result.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	return result.String()
}
