package argocd

import "strings"

type Application struct {
	Name         string `json:"name"`
	HealthStatus string `json:"healthStatus"`
	SyncStatus   string `json:"syncStatus"`
	SyncCommit   string `json:"syncCommit"`
	Project      string `json:"project"`
	LastActivity string `json:"lastActivity"`
}

type Resource struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	HealthStatus string `json:"healthStatus"`
	SyncStatus   string `json:"syncStatus"`
	Namespace    string `json:"namespace"`
}

func (a *Application) SearchString() string {
	return strings.ToLower(a.Name +
		" " + a.HealthStatus +
		" " + a.Project +
		" " + a.SyncStatus +
		" " + a.SyncCommit +
		" " + a.LastActivity)
}

func (r *Resource) SearchString() string {
	return strings.ToLower(r.Kind + " " + r.Name + " " + r.Namespace)
}
