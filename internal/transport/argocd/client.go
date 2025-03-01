package argocd

import (
	"context"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
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
			SyncCommit:   syncCommit[0:7],
			Project:      app.Spec.Project,
			LastActivity: lastActivity,
		})
	}
	return apps, nil
}

func (a *ArgoCdClient) GetAppResources(appName string) ([]Resource, error) {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error getting application client: %v", err)
	}
	resList, err := appClient.ManagedResources(a.ctx, &application.ResourcesQuery{
		ApplicationName: &appName,
	})
	if err != nil {
		return nil, a.logger.Errorf("Error getting resources for app %s: %v", appName, err)
	}

	var resources []Resource
	for _, res := range resList.Items {
		resources = append(resources, Resource{
			Kind:      res.Kind,
			Name:      res.Name,
			Namespace: res.Namespace,
		})
	}
	return resources, nil
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
