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
	SkillsPath   string     `json:"skills_path"`
	Port         string     `json:"port"`
	Theme        string     `json:"theme"`
	Providers    []Provider `json:"providers"`
	CurrentModel ResModel   `json:"current_model"`
}

type Provider struct {
	BaseUrl string   `json:"base_url"`
	ApiKey  string   `json:"api_key"`
	Models  []string `json:"models"`
}

// type Model struct {
// 	Model string `json:"model"`
// }

type ResModel struct {
	Model   string `json:"model"`
	ApiKey  string `json:"api_key"`
	BaseUrl string `json:"base_url"`
}

func CustomizationPath() (string, error) {
	path := filepath.Join(Dir(), "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		bare := Customization{
			Theme:      "light",
			SkillsPath: filepath.Join(Dir(), "skills"),
			Port:       "8080",
			Providers:  []Provider{},
		}
		d, err := json.MarshalIndent(bare, "", " ")
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
	d, err := json.MarshalIndent(customization, "", " ")
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

func GetModels() ([]ResModel, error) {
	providers := GetCustomization().Providers
	models := []ResModel{}
	for _, provider := range providers {
		for _, model := range provider.Models {
			models = append(models, ResModel{
				ApiKey:  provider.ApiKey,
				BaseUrl: provider.BaseUrl,
				Model:   model,
			})
		}
	}
	return models, nil
}

func SetApiKey(m ResModel, apikey string) error {
	customization := GetCustomization()
	for i := range customization.Providers {
		if customization.Providers[i].BaseUrl == m.BaseUrl {
			customization.Providers[i].ApiKey = apikey
		}
	}
	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error Setting api keyl")
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

func SetCurrentModel(model ResModel) error {
	customization := GetCustomization()
	customization.CurrentModel = model
	d, err := json.MarshalIndent(customization, "", " ")
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

func GetCurrentModel() (ResModel, error) {
	if GetCustomization().CurrentModel.Model == "" {
		models, err := GetModels()
		if err != nil {
			return ResModel{}, err
		}
		return models[0], nil
	}
	return GetCustomization().CurrentModel, nil
}
