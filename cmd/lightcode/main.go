package main

import (
	"flag"
	"net"
	"net/http"

	"github.com/Kartik-2239/lightcode/internal/server"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/tui/views"
)

func main() {
	isServer := flag.Bool("server", false, "to run the server only")
	isTui := flag.Bool("tui", false, "to run the tui only")
	flag.Parse()
	Lightcode(!*isTui, !*isServer)

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
			if isServer {
				ready := make(chan struct{})
				go server.Initialise(ready, port)
				<-ready
			}
		}
	}
	if isTui {
		views.LauchHomePage()
	}
}
