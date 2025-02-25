package config

type Config struct {
	Argocd *ArgocdConfig
}

type ArgocdConfig struct {
	Url                string
	Token              string
	InsecureSkipVerify bool
}
