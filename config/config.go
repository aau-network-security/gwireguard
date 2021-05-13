package config

import (
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	WgConfig struct {
		Eth                 string `yaml:"eth"`
		Dir                 string `yaml:"dir"`
		WGInterfaceTemplate string `yaml:"wg-interface-temp"`
	} `yaml:"wireguard-config"`
	ServiceConfig struct {
		Domain struct {
			Endpoint string `yaml:"endpoint"`
			Port     uint   `yaml:"port"`
		} `yaml:"domain"`
		TLS struct {
			Enabled  bool   `yaml:"enabled"`
			CertFile string `yaml:"certFile"`
			CertKey  string `yaml:"certKey"`
			CAFile   string `yaml:"caFile"`

			Directory string `yaml:"directory"`
		} `yaml:"tls"`
		Auth struct {
			AKey string `yaml:"aKey"`
			SKey string `yaml:"sKey"`
		} `yaml:"auth"`
	} `yaml:"service-config"`
}

func NewConfig(path string) (*Config, error) {
	f, err := ioutil.ReadFile(path)

	if err != nil {
		log.Error().Msgf("Reading config file err: %v", err)
		return nil, err
	}

	var c Config
	err = yaml.Unmarshal(f, &c)
	if err != nil {
		log.Error().Msgf("Unmarshall error %v \n", err)
		return nil, err
	}
	return &c, nil
}
