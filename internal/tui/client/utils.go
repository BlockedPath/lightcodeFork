package client

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/config"
)

func isPortInUse_(port string) bool {
	// ln, err := net.Listen("tcp", ":"+port)
	// if err != nil {
	// 	return true // port is in use
	// }
	data, err := http.Get("http://localhost:" + port)
	if err != nil {
		return true
	}
	defer data.Body.Close()
	d, err := io.ReadAll(data.Body)
	if err != nil {
		return true
	}
	if strings.Contains(string(d), "lightcode") {
		return true
	}
	// ln.Close()
	return false
}

func getPortRunningServer() string {
	port := config.GetCustomization().Port
	portInt, _ := strconv.Atoi(port)
	for {
		// fmt.Println("port", portInt)
		if isPortInUse_(strconv.Itoa(portInt)) {
			return strconv.Itoa(portInt)
		} else {
			portInt++
		}
	}
}
