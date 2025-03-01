package argocd

import (
	"context"
	"fmt"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type ArgoCdClient struct {
	cfg    *config.Instance
	client apiclient.Client
	logger *logging.Logger
	ctx    context.Context
}

func NewArgoCdClient(cfg *config.Instance, l *logging.Logger, ctx context.Context) *ArgoCdClient {
	clientOpt := &apiclient.ClientOptions{
		Insecure:   cfg.InsecureSkipVerify,
		ServerAddr: cfg.Url,
		AuthToken:  cfg.Token,
	}
	c, err := apiclient.NewClient(clientOpt)
	if err != nil {
		l.Fatal("Error creating ArgoCD client: %v", err)
	}
	return &ArgoCdClient{
		cfg:    cfg,
		client: c,
		logger: l,
		ctx:    ctx,
	}
}

func (a *ArgoCdClient) GetApps() ([]Application, error) {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating argocd client: %v", err)
	}
	appList, err := appClient.List(a.ctx, &application.ApplicationQuery{})
	if err != nil {
		return nil, a.logger.Errorf("Error getting application list: %v", err)
	}

	var apps []Application
	for _, app := range appList.Items {
		var lastActivity string
		var syncCommit string
		if app.Status.OperationState != nil {
			if app.Status.OperationState.SyncResult != nil {
				syncCommit = app.Status.OperationState.SyncResult.Revision
			} else {
				syncCommit = "n/a"
			}
			if !app.Status.OperationState.FinishedAt.IsZero() {
				lastActivity = app.Status.OperationState.FinishedAt.Format("2006-01-02 15:04:05")
			} else {
				lastActivity = "n/a"
			}
		} else {
			syncCommit = "n/a"
			lastActivity = "n/a"
		}

		apps = append(apps, Application{
			Name:         app.Name,
			HealthStatus: string(app.Status.Health.Status),
			SyncStatus:   string(app.Status.Sync.Status),
			SyncCommit:   syncCommit,
			Project:      app.Spec.Project,
			LastActivity: lastActivity,
		})
	}
	return apps, nil
}

func (a *ArgoCdClient) GetAppResources(appName string) ([]Resource, error) {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating argocd client: %v", err)
	}

	tree, err := a.GetResourceTree(appName)
	if err != nil {
		return nil, a.logger.Errorf("Error getting resource tree for %s: %v", appName, err)
	}

	resList, err := appClient.ManagedResources(a.ctx, &application.ResourcesQuery{
		ApplicationName: &appName,
	})
	if err != nil {
		return nil, a.logger.Errorf("Error getting managed resources for %s: %v", appName, err)
	}

	healthMap := make(map[string]string)
	for _, node := range tree.Nodes {
		key := fmt.Sprintf("%s/%s/%s/%s", node.Group, node.Kind, node.Namespace, node.Name)
		if node.Health != nil {
			healthMap[key] = string(node.Health.Status)
		} else {
			healthMap[key] = "Unknown"
		}
	}

	var resources []Resource
	for _, res := range resList.Items {
		key := fmt.Sprintf("%s/%s/%s/%s", res.Group, res.Kind, res.Namespace, res.Name)

		healthStatus, exists := healthMap[key]
		if !exists {
			healthStatus = "Unknown"
		}

		syncStatus := "Synced"
		if res.Diff != "" {
			syncStatus = "OutOfSync"
		}

		resources = append(resources, Resource{
			Kind:         res.Kind,
			Name:         res.Name,
			Namespace:    res.Namespace,
			HealthStatus: healthStatus,
			SyncStatus:   syncStatus,
		})
	}

	return resources, nil
}

func (a *ArgoCdClient) GetResourceTree(appName string) (*v1alpha1.ApplicationTree, error) {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating ArgoCD application client: %v", err)
	}
	query := &application.ResourcesQuery{
		ApplicationName: &appName,
	}
	tree, err := appClient.ResourceTree(a.ctx, query)
	if err != nil {
		return nil, a.logger.Errorf("Error getting resource tree for %s: %v", appName, err)
	}
	return tree, nil
}

func (a *ArgoCdClient) RefreshApp(appName string, refreshType string) error {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return a.logger.Errorf("Error getting application client: %v", err)
	}

	_, err = appClient.Get(a.ctx, &application.ApplicationQuery{
		Name:    &appName,
		Refresh: &refreshType,
	})
	if err != nil {
		return a.logger.Errorf("Error refreshing app %s: %v", appName, err)
	}

	return nil
}
