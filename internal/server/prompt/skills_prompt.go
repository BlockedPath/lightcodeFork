package prompt

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/config"
)

func AvailableSkills() string {
	skillPath := config.SkillsPath()
	skill_data := []string{}
	files, err := os.ReadDir(skillPath)
	if err != nil {
		return "Error: " + err.Error()
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(skillPath, file.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`(?s)---.*?---`)
		skill_data1 := re.FindAllString(string(data), -1)[0]
		skill_data1 = strings.TrimSpace(strings.ReplaceAll(skill_data1, "---", ""))
		lines := strings.Split(skill_data1, "\n")
		name := strings.ReplaceAll(lines[0], "name: ", "") + "\n"
		description := strings.ReplaceAll(lines[1], "description: ", "")
		skill_data = append(skill_data, "<skill>\n<name>"+name+"</name>\n<description>"+description+"</description>\n</skill>")
	}
	AvailableSkills := "<available_skills>\n" + strings.Join(skill_data, "\n") + "\n</available_skills>"
	return AvailableSkills
}
