package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type DiscordConfig struct {
	Token  string `json:"apitoken"`
	Prefix string `json:"prefix"`
}

type Config struct {
	Discord DiscordConfig `json:"discord"`
}

func LoadConfig(configPath string) (Config, error) {
	config := Config{}
	file, err := os.Open(configPath)
	if err != nil {
		return Config{}, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}
	return config, nil
}
