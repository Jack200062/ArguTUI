package argocd

import "strings"

type Application struct {
	Name         string `json:"name"`
	HealthStatus string `json:"healthStatus"`
	SyncStatus   string `json:"syncStatus"`
	SyncCommit   string `json:"syncCommit"`
	Project      string `json:"project"`
	LastActivity string `json:"lastActivity"`
    // Cached lower-cased concatenation for search; not serialized
    SearchIndex  string `json:"-"`
}

type Resource struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	HealthStatus string `json:"healthStatus"`
	SyncStatus   string `json:"syncStatus"`
	Namespace    string `json:"namespace"`
}

type TreeResource struct {
	Kind       string
	Name       string
	Health     string
	SyncStatus string
	Namespace  string

	Children []*TreeResource
	Expanded bool
	Depth    int
}

func (a *Application) SearchString() string {
    if a.SearchIndex != "" {
        return a.SearchIndex
    }
    a.SearchIndex = strings.ToLower(a.Name +
        " " + a.HealthStatus +
        " " + a.Project +
        " " + a.SyncStatus +
        " " + a.SyncCommit)
    return a.SearchIndex
}

func (r *Resource) SearchString() string {
	return strings.ToLower(r.Kind + " " + r.Name + " " + r.Namespace)
}
