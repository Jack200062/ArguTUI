package argocd

import (
	"context"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
)

type ArgoCdClient struct {
	cfg    *config.Config
	client apiclient.Client
	logger *logging.Logger
}

func NewArgoCdClient(cfg *config.Config, l *logging.Logger) *ArgoCdClient {
	clientOpt := &apiclient.ClientOptions{
		Insecure:   cfg.Argocd.InsecureSkipVerify,
		ServerAddr: cfg.Argocd.Url,
		AuthToken:  cfg.Argocd.Token,
	}
	c, err := apiclient.NewClient(clientOpt)
	if err != nil {
		l.Fatal("Error creating ArgoCD client: %v", err)
	}
	return &ArgoCdClient{
		cfg:    cfg,
		client: c,
		logger: l,
	}
}

func (a *ArgoCdClient) GetApps() ([]Application, error) {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error getting applications: %v", err)
	}
	ctx := context.Background()
	appList, err := appClient.List(ctx, &application.ApplicationQuery{})
	if err != nil {
		return nil, a.logger.Errorf("Error listing applications: %v", err)
	}

	var apps []Application
	for _, app := range appList.Items {
		apps = append(apps, Application{
			Name:    app.Name,
			Status:  string(app.Status.Health.Status),
			Project: app.Spec.Project,
		})
	}
	return apps, nil
}

func (a *ArgoCdClient) GetAppResources(appName string) ([]Resource, error) {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error getting application client: %v", err)
	}
	ctx := context.Background()
	resList, err := appClient.ManagedResources(ctx, &application.ResourcesQuery{
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
