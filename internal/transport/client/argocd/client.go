package argocd

import (
	"context"
	"log"

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
		log.Fatalf("Error creating ArgoCD client: %v", err)
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
