package config

import (
	"github.com/pelletier/go-toml"
	"io/ioutil"
)

func SaveConfig(market *MarketConfig) error {
	cfgBytes, err := toml.Marshal(market)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(market.HomeDir, cfgBytes, 0x777)
}

func LoadConfig(cfgPath string, market *MarketConfig) error {
	cfgBytes, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	return toml.Unmarshal(cfgBytes, &market)
}
