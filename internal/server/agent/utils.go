package agent

import (
	"os"
	"path"
)

func ReadAgentsMd(dir string) string {
	data, error := os.ReadFile(path.Join(dir, "AGENTS.md"))
	if os.IsExist(error) {
		return ""
	}

	return string(data)
}
