package argocd

import "strings"

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

func (a Application) SearchString() string {
	return strings.ToLower(a.Name + " " + a.Status + " " + a.Project)
}

func (r Resource) SearchString() string {
	return strings.ToLower(r.Kind + " " + r.Name + " " + r.Namespace)
}
