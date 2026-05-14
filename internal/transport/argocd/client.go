package argocd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/session"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/settings"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type ArgoCdClient struct {
	cfg    *config.Instance
	client apiclient.Client
	logger *logging.Logger
	ctx    context.Context
}

func (a *ArgoCdClient) HttpClient() (*http.Client, error) {
	return a.client.HTTPClient()
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

// TODO: cache clients returned bu New...Client() calls to reuse GRPC connections

func (a *ArgoCdClient) GetApps() ([]Application, error) {
	closer, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating argocd client: %v", err)
	}
	defer closer.Close()
	appList, err := appClient.List(a.ctx, &application.ApplicationQuery{})
	if err != nil {
		return nil, a.logger.Errorf("Error getting application list: %v", err)
	}

	var apps []Application
	for _, app := range appList.Items {
		var lastSyncTime string

		if !app.Status.ReconciledAt.IsZero() {
			lastSyncTime = app.Status.ReconciledAt.Format("2006-01-02 15:04:05")
		} else if app.Status.OperationState != nil && !app.Status.OperationState.FinishedAt.IsZero() {
			lastSyncTime = app.Status.OperationState.FinishedAt.Format("2006-01-02 15:04:05")
		} else if !app.CreationTimestamp.IsZero() {
			lastSyncTime = app.CreationTimestamp.Format("2006-01-02 15:04:05")
		} else {
			lastSyncTime = "n/a"
		}

		var syncCommit string
		if app.Status.OperationState != nil &&
			app.Status.OperationState.SyncResult != nil {
			syncCommit = app.Status.OperationState.SyncResult.Revision
		} else {
			syncCommit = "n/a"
		}

		if len(syncCommit) > 7 {
			syncCommit = syncCommit[:7]
		}

		apps = append(apps, Application{
			Name:         app.Name,
			HealthStatus: string(app.Status.Health.Status),
			SyncStatus:   string(app.Status.Sync.Status),
			SyncCommit:   syncCommit,
			Project:      app.Spec.Project,
			LastActivity: lastSyncTime,
		})
		// Fill cached search index to avoid recomputing during filtering
		apps[len(apps)-1].SearchIndex = apps[len(apps)-1].SearchString()
	}
	return apps, nil
}

func (a *ArgoCdClient) GetAppResources(appName string) ([]Resource, error) {
	closer, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating argocd client: %v", err)
	}
	defer closer.Close()

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
	closer, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating ArgoCD application client: %v", err)
	}
	defer closer.Close()
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
	closer, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return a.logger.Errorf("Error getting application client: %+v", err)
	}
	defer closer.Close()
	_, err = appClient.Get(a.ctx, &application.ApplicationQuery{
		Name:    &appName,
		Refresh: &refreshType,
	})
	if err != nil {
		return a.logger.Errorf("Error refreshing app %s: %v", appName, err)
	}

	return nil
}

func (a *ArgoCdClient) SyncApp(appName string) error {
	closer, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return a.logger.Errorf("Error getting application client: %+v", err)
	}
	defer closer.Close()
	syncRequest := &application.ApplicationSyncRequest{
		Name: &appName,
	}
	_, err = appClient.Sync(a.ctx, syncRequest)
	if err != nil {
		return a.logger.Errorf("Error syncing app %s: %v", appName, err)
	}
	return nil
}

func (a *ArgoCdClient) DeleteApp(appName string) error {
	closer, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return a.logger.Errorf("Error getting application client: %+v", err)
	}
	defer closer.Close()
	deleteRequest := &application.ApplicationDeleteRequest{
		Name: &appName,
	}
	_, err = appClient.Delete(a.ctx, deleteRequest)
	if err != nil {
		return a.logger.Errorf("Error deleting app %s: %v", appName, err)
	}
	return nil
}

// Prepare OAuth2 config and openId dsicovert provider
func (a *ArgoCdClient) OpenIDConfig(conf *settings.Settings) (*oauth2.Config, *oidc.Provider, error) {
	return a.client.OIDCConfig(a.ctx, conf)
}

// Get ArgoCD Settings via GET /api/v1/settings
func (a *ArgoCdClient) GetSettings() (*settings.Settings, error) {
	closer, settingsClient, err := a.client.NewSettingsClient()
	if err != nil {
		return nil, a.logger.Errorf("Error getting settings client: %+v", err)
	}
	defer closer.Close()

	return settingsClient.Get(a.ctx, &settings.SettingsQuery{})
}

func (a *ArgoCdClient) CreateSession(username string, password string) (*session.SessionResponse, error) {
	closer, sessionClient, err := a.client.NewSessionClient()
	if err != nil {
		return nil, a.logger.Errorf("Error getting session client: %+v", err)
	}
	defer closer.Close()
	request := session.SessionCreateRequest{
		Username: username,
		Password: password,
	}
	return sessionClient.Create(a.ctx, &request)
}
