package main

import (
	"net"
	"net/http"

	"github.com/Kartik-2239/lightcode/internal/server"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/tui/views"
)

func main() {
	Lightcode()

}
func isPortInUse(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return true // port is in use
	}
	ln.Close()
	return false
}
func Lightcode() {
	port := config.GetCustomization().Port
	if !isPortInUse(port) {
		_, err := http.Get("http://localhost:" + port)
		if err != nil {
			ready := make(chan struct{})
			go server.Initialise(ready, port)
			<-ready
		}
	}
	views.LauchHomePage()
}
