package main

import (
	"fmt"

	"github.com/Kartik-2239/lightcode/internal/server/tools"
)

func main() {
	// port := "8080"
	// ready := make(chan struct{})
	// go server.Initialise(ready, port)
	// <-ready
	response, err := tools.Skill(tools.ToolContext{WorkingDirectory: "/Users/kartik/Desktop/lightcode"}, map[string]any{"skillName": "frontend-skill"})
	if err != nil {
		fmt.Println("Error", err)
		return
	}
	fmt.Println("Response", response.Content)
}
