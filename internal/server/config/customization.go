package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ```bash
// OPENAI_API_KEY=sk-...
// OPENAI_BASE_URL=https://...
// SKILL_PATH=path/to/skill/folder
// API_URL=http://localhost:8080
// ```
type Customization struct {
	SkillsPath   string         `json:"skills_path"`
	Port         string         `json:"port"`
	Theme        string         `json:"theme"`
	Providers    []Provider     `json:"providers"`
	CurrentModel ResModel       `json:"current_model"`
	RecentModels []RecentModels `json:"recent_models"`
}

type Provider struct {
	BaseUrl string   `json:"base_url"`
	ApiKey  string   `json:"api_key"`
	Models  []string `json:"models"`
}

type ResModel struct {
	Model   string `json:"model"`
	ApiKey  string `json:"api_key"`
	BaseUrl string `json:"base_url"`
}
type RecentModels struct {
	Model    string `json:"model"`
	ApiKey   string `json:"api_key"`
	BaseUrl  string `json:"base_url"`
	LastUsed int64  `json:"last_used"`
}
type AllModels struct {
	Models       []ResModel
	RecentModels []RecentModels
}

func CustomizationPath() (string, error) {
	path := filepath.Join(Dir(), "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		def_prov := getDefaultProviders()
		bare := Customization{
			Theme:      "light",
			SkillsPath: filepath.Join(Dir(), "skills"),
			Port:       "8080",
			Providers:  getDefaultProviders(),
			CurrentModel: ResModel{
				Model:   def_prov[0].Models[0],
				BaseUrl: def_prov[0].BaseUrl,
				ApiKey:  "",
			},
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

func GetModels() ([]ResModel, []RecentModels, error) {
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
	return models, GetCustomization().RecentModels, nil
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

func GetRecentModels() []RecentModels {
	customization := GetCustomization()
	return customization.RecentModels
}

func SetCurrentModel(model ResModel) error {
	customization := GetCustomization()
	customization.CurrentModel = model
	r := customization.RecentModels
	changed := false
	if len(r) != 0 {
		for i, m := range r {
			if m.BaseUrl == model.BaseUrl && m.Model == model.Model {
				r[i].LastUsed = time.Now().Unix()
				changed = true
			}
		}
	}

	if !changed {
		r = append(r, RecentModels{
			Model:    model.Model,
			ApiKey:   model.ApiKey,
			BaseUrl:  model.BaseUrl,
			LastUsed: time.Now().Unix(),
		})
	}
	customization.RecentModels = r
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
	models, recent_models, err := GetModels()
	sort.Slice(recent_models, func(i, j int) bool {
		return recent_models[i].LastUsed < recent_models[j].LastUsed
	})
	if err != nil {
		return ResModel{}, err
	}
	if len(recent_models) > 0 {
		r := recent_models[0]
		return ResModel{Model: r.Model, ApiKey: r.ApiKey, BaseUrl: r.BaseUrl}, nil
	} else {
		return models[0], nil
	}
}
