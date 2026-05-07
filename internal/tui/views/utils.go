package views

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func FormatK(n int64) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return strconv.FormatInt(n, 10)
}

func shortenDir(curDir string) string {
	home, _ := os.UserHomeDir()
	return strings.Replace(curDir, home, "~", 1)
}
