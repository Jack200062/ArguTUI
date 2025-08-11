package config

type Instance struct {
	Name               string `mapstructure:"name"`
	Url                string `mapstructure:"url"`
	Token              string `mapstructure:"token"`
	InsecureSkipVerify bool   `mapstructure:"insecureskipverify"`
	Anonymous          bool   `mapstructure:"anonymous"`
}

type Config struct {
	Instances []*Instance `mapstructure:"instances"`
}
