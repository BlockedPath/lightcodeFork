package agent

import (
	"os"
	"path"
)

func ReadAgentsMd(dir string) (string, error) {
	data, error := os.ReadFile(path.Join(dir, "AGENTS.md"))
	if os.IsExist(error) {
		return "", error
	}

	return string(data), nil
}
