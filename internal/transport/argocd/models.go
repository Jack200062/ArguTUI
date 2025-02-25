package argocd

type Application struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Project string `json:"project"`
}
