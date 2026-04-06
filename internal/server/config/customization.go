package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ```bash
// OPENAI_API_KEY=sk-...
// OPENAI_BASE_URL=https://...
// SKILL_PATH=path/to/skill/folder
// API_URL=http://localhost:8080
// ```
type Customization struct {
	SkillsPath   string  `json:"skills_path"`
	ApiUrl       string  `json:"api_url"`
	Theme        string  `json:"theme"`
	Models       []Model `json:"models"`
	CurrentModel Model   `json:"current_model"`
}

type Model struct {
	Model   string `json:"model"`
	BaseUrl string `json:"base_url"`
	ApiKey  string `json:"api_key"`
}

func CustomizationPath() (string, error) {
	path := filepath.Join(Dir(), "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		bare := Customization{
			Theme:      "light",
			SkillsPath: filepath.Join(Dir(), "skills"),
			ApiUrl:     "http://localhost:8080",
			Models:     []Model{},
		}
		d, err := json.Marshal(bare)
		if err != nil {
			return "", err
		}
		_ = os.WriteFile(path, d, 0644)
	}
	return path, nil
}

func GetCustomization() Customization {
	path, err := CustomizationPath()
	if err != nil {
		return Customization{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Customization{}
	}
	var customization Customization
	json.Unmarshal(data, &customization)
	return customization
}

func GetTheme() string {
	return GetCustomization().Theme
}

func SetTheme(theme string) error {
	customization := GetCustomization()
	customization.Theme = theme
	d, err := json.Marshal(customization)
	if err != nil {
		return errors.New("Error Setting theme")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error Setting theme")
	}
	err = os.WriteFile(path, d, 0644)
	if err != nil {
		return errors.New("Error Setting theme")
	}
	return nil
}

func GetModels() ([]Model, error) {
	models := GetCustomization().Models
	if len(models) == 0 {
		return []Model{}, errors.New("no models found")
	}
	return models, nil
}

func AddModel(model Model) error {
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error Adding model")
	}
	customization := GetCustomization()
	customization.Models = append(customization.Models, model)
	d, err := json.Marshal(customization)
	if err != nil {
		return errors.New("Error Adding model")
	}
	err = os.WriteFile(path, d, 0644)
	if err != nil {
		return errors.New("Error Adding model")
	}
	return nil
}

func SetCurrentModel(model Model) error {
	customization := GetCustomization()
	customization.CurrentModel = model
	d, err := json.Marshal(customization)
	if err != nil {
		return errors.New("Error Setting current model")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error Setting current model")
	}
	err = os.WriteFile(path, d, 0644)
	if err != nil {
		return errors.New("Error Setting current model")
	}
	return nil
}

func GetCurrentModel() Model {
	if GetCustomization().CurrentModel.Model == "" {
		return GetCustomization().Models[0]
	}
	return GetCustomization().CurrentModel
}
