package lightcode

import (
	"net"
	"net/http"

	"github.com/Kartik-2239/lightcode/internal/server"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/tui/views"
)

func Lightcode() {
	port := config.GetCustomization().ApiUrl
	if !isPortInUse(port) {
		_, err := http.Get("http://localhost:" + port)
		if err != nil {
			ready := make(chan struct{})
			go server.Initialise(ready, port)
			<-ready
		}
		// body, err := io.ReadAll(resp.Body)
		// if string(body) != "lightcode is running!" {
		// 	log.Fatal("port: " + port + " is not being used by lightcode!")
		// }
	}
	views.LauchHomePage()
}

func isPortInUse(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return true // port is in use
	}
	ln.Close()
	return false
}
