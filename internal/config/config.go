package config

import (
	"github.com/caarlos0/env/v6"
)

type Config struct {
	LabelSelector string `env:"LABELSELECTOR" envDefault:"type=external"` //only check custom endpoints with label type=external
	Touser        string `env:"TOUSER", envDefault:"@all"`
	Corpid        string `env:"CORPID"`
	Corpsecret    string `env:"CORPSECRET"`
	Agentid       int    `env:"AGENTID"`
	LogLevel      string `env:"LOGLEVEL" envDefault:"debug"`

	Retry    int `env:"RETRY" envDefault:"3"`
	Interval int `env:"INTERVAL" envDefault:"2"`
	Timeout  int `env:"TIMEOUT" envDefault:"500"`
}

func (cfg *Config) Init() error {
	// app config
	err := env.Parse(cfg)
	return err
}
