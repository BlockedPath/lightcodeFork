package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server"
	"github.com/Kartik-2239/lightcode/internal/server/agent"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/views"
)

func main() {
	var prompt string
	flag.StringVar(&prompt, "prompt", "", "prompt for the agent")
	flag.StringVar(&prompt, "p", "", "prompt for the agent (shorthand)")

	isServer := flag.Bool("server", false, "")
	isTui := flag.Bool("tui", false, "")

	flag.Parse()
	if *isServer {
		Lightcode(true, false)
		return
	}
	if *isTui {
		Lightcode(false, true)
		return
	}
	if prompt == "" {
		Lightcode(true, true)
		return
	}
	// fmt.Println(prompt)
	runAgent(prompt)
}

func isPortInUse(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return true // port is in use
	}
	ln.Close()
	return false
}
func Lightcode(isServer bool, isTui bool) {
	port := config.GetCustomization().Port
	if !isPortInUse(port) {
		_, err := http.Get("http://localhost:" + port)
		if err != nil {
			if isServer && isTui {
				ready := make(chan struct{})
				go server.Initialise(ready, port)
				<-ready
			}
			if isServer && !isTui {
				ready := make(chan struct{})
				server.Initialise(ready, port)
			}
		}
	}
	if isTui {
		views.LauchHomePage()
	}
}

func runAgent(prompt string) {
	ctx := context.Background()
	database, _ := db.Connect()
	session_id := randomSessionID()
	path, err := os.Getwd()
	if err != nil {
		fmt.Println("couldn't create a session")
	}
	session := models.Session{ID: session_id, Title: prompt, Directory: path}
	database.Create(&session)

	var messages []models.Message
	database.Table("messages").Select("*").Where("session_id = ?", session_id).Find(&messages)
	newMessage := models.Message{SessionID: session_id, Data: models.EncodeMessageData(models.StoredMessageData{Role: "user", Content: prompt})}

	database.Create(&newMessage)
	for result := range agent.New().Run(ctx, prompt, [][]byte{}, session_id, "chat") {
		fmt.Println(result.Content)
		for _, tool := range result.ToolCalls {
			fmt.Printf("%s({%s})", tool.Name, tool.Arguments)
		}
	}
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
