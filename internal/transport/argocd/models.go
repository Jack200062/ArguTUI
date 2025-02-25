package argocd

type Application struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Project string `json:"project"`
}

type Resource struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
