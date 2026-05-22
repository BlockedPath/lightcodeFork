package config

import (
	"os"
	"path/filepath"
)

func Dir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".lightcode")
	os.MkdirAll(dir, 0755)
	return dir
}

func SkillsPath() string {
	return GetCustomization().SkillsPath
}

func GetAuthPath() (string, error) {
	path := filepath.Join(Dir(), "auth.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", err
	}
	return path, nil

}

func DBPath() string {
	return filepath.Join(Dir(), "lightcode.db")
}
