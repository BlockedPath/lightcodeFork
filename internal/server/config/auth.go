package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type AuthVal struct {
	Type      string   `json:"type"`
	Access    string   `json:"access_token"`
	Refresh   string   `json:"refresh_token"`
	Expires   int64    `json:"expires"`
	AccountId string   `json:"account_id"`
	Models    []string `json:"models"`
}

type AuthJson map[string]AuthVal

func GetAuthVal(provider string) (AuthVal, error) {
	path, err := GetAuthPath()
	if err != nil {
		return AuthVal{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AuthVal{}, err
	}
	var authJson AuthJson
	err = json.Unmarshal(data, &authJson)
	if err != nil {
		return AuthVal{}, err
	}

	authVal, ok := authJson[provider]
	if !ok {
		return AuthVal{}, fmt.Errorf("auth provider %q not found in auth.json", provider)
	}

	return authVal, nil
}

func UpdateAuthVal(provider string, newVal AuthVal) error {
	path, err := GetAuthPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var authJson AuthJson
	err = json.Unmarshal(data, &authJson)
	if err != nil {
		return err
	}

	authJson[provider] = newVal

	updatedData, err := json.MarshalIndent(authJson, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, updatedData, 0644)
}

func GetAllAuthModels() ([]ResModel, error) {
	path, err := GetAuthPath()
	if err != nil {
		fmt.Println("Error getting auth path:", err)
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading auth file:", err)
		return nil, err
	}
	var authJson AuthJson
	err = json.Unmarshal(data, &authJson)
	if err != nil {
		fmt.Println("Error unmarshalling auth JSON:", err)
		return nil, err
	}
	keys := make([]string, 0, len(authJson))
	for k := range authJson {
		keys = append(keys, k)
	}
	var models []ResModel

	for _, au := range keys {
		authVal := authJson[au]
		for _, m := range authVal.Models {
			// fmt.Printf("Adding model from auth: %s with provider %s\n", m, au)
			models = append(models, ResModel{Model: m, ApiKey: "", BaseUrl: au})
		}
	}
	return models, nil
}
