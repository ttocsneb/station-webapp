package util

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Base       string `toml:"base"`
	Db         string `toml:"db"`
	Listen     string `toml:"listen"`
	MqttServer string `toml:"mqtt_server"`
	MqttId     string `toml:"mqtt_id"`
	StationId  string `toml:"station_id"`
}

var Conf Config

func LoadConfig(path string) (*Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(contents, &Conf)
	if err != nil {
		return nil, err
	}
	return &Conf, nil
}
