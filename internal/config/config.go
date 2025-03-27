package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const filename = ".gatorconfig.json"

func Read() Config {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	configPath := homeDirectory + "/" + filename

	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		defaultConfig := Config{
			DbURL:           "postgres://usuario:senha@localhost:5432/rss_db?sslmode=disable",
			CurrentUserName: "",
		}

		jsonData, err := json.Marshal(defaultConfig)
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(configPath, jsonData, 0644)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Arquivo de configuração criado em: %s\n", configPath)
		return defaultConfig
	} else if err != nil {
		log.Fatal(err)
	}

	dat, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	cfg := Config{}
	if err := json.NewDecoder(bytes.NewReader(dat)).Decode(&cfg); err != nil {
		log.Fatal(err)
	}
	return cfg
}

func (cfg *Config) SetUser(username string) {
	cfg.CurrentUserName = username

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(homeDirectory+"/"+filename, jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}

}

func (cfg *Config) GetUser() *Config {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	dat, err := os.ReadFile(homeDirectory + "/" + filename)
	if err != nil {
		log.Fatal(err)
	}

	if err := json.NewDecoder(bytes.NewReader(dat)).Decode(&cfg); err != nil {
		log.Fatal(err)
	}
	return cfg
}
