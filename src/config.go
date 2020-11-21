package doremy

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type discordConfig struct {
	Token  string `json:"apitoken"`
	Prefix string `json:"prefix"`
}

type pollingConfig struct {
	Emojis              []string `json:"emojis"`
	DaemonTime          float64  `json:"daemon-polling-time"`
	SleepPeriodDuration float64  `json:"sleep-period-min-time"`
}

type configStruct struct {
	Discord  discordConfig `json:"discord"`
	Datafile string        `json:"datafile"`
	Polling  pollingConfig `json:"polling"`
}

func LoadConfig(configPath string) (configStruct, error) {
	config := configStruct{}
	file, err := os.Open(configPath)
	if err != nil {
		return configStruct{}, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return configStruct{}, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return configStruct{}, err
	}
	return config, nil
}
