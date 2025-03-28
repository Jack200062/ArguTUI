package argocd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/internal/cache"
	"github.com/Jack200062/ArguTUI/internal/models"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type ArgoCdClient struct {
	cfg          *config.Instance
	cacheManager *cache.CacheManager
	client       apiclient.Client
	logger       *logging.Logger
	ctx          context.Context
}

func NewArgoCdClient(cfg *config.Instance, l *logging.Logger, ctx context.Context, cache *cache.CacheManager) *ArgoCdClient {
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
		cfg:          cfg,
		client:       c,
		logger:       l,
		ctx:          ctx,
		cacheManager: cache,
	}
}

func (a *ArgoCdClient) GetApps() ([]models.Application, error) {
	file, err := os.OpenFile("performance.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return nil, nil
	}
	startTime := time.Now()
	defer func() {
		fmt.Fprintf(file, "GetApps took %s\n", time.Since(startTime))
	}()

	if apps, found := a.cacheManager.GetAppList(a.cfg.Name); found {
		fmt.Fprintf(file, "---------- Cache hit for app list: %d apps -----------\n", len(apps))
		return apps, nil
	}

	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating argocd client: %v", err)
	}
	appList, err := appClient.List(a.ctx, &application.ApplicationQuery{})
	if err != nil {
		return nil, a.logger.Errorf("Error getting application list: %v", err)
	}

	var apps []models.Application
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

		apps = append(apps, models.Application{
			Name:         app.Name,
			HealthStatus: string(app.Status.Health.Status),
			SyncStatus:   string(app.Status.Sync.Status),
			SyncCommit:   syncCommit,
			Project:      app.Spec.Project,
			LastActivity: lastSyncTime,
		})
	}

	a.cacheManager.SetAppList(a.cfg.Name, apps, a.cacheManager.DefaultExpiration)

	return apps, nil
}

func (a *ArgoCdClient) GetAppResources(appName string) ([]models.Resource, error) {
	file, err := os.OpenFile("performance.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return nil, nil
	}
	startTime := time.Now()
	defer func() {
		fmt.Fprintf(file, "GetAppResources took %s\n", time.Since(startTime))
	}()
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating argocd client: %v", err)
	}

	tree, err := a.GetResourceTree(appName)
	if err != nil {
		return nil, a.logger.Errorf("Error getting resource tree for %s: %v", appName, err)
	}

	if resources, found := a.cacheManager.GetResources(a.cfg.Name, appName); found {
		fmt.Fprintf(file, "------------ Cache hit for resources: %d resources -------------\n", len(resources))
		return resources, nil
	}

	fmt.Fprintf(file, "ASDAADDSADSDSDASDASSDSASDADS\n")
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

	var resources []models.Resource
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

		resources = append(resources, models.Resource{
			Kind:         res.Kind,
			Name:         res.Name,
			Namespace:    res.Namespace,
			HealthStatus: healthStatus,
			SyncStatus:   syncStatus,
		})
	}

	a.cacheManager.SetResources(a.cfg.Name, appName, resources, a.cacheManager.DefaultExpiration)

	return resources, nil
}

func (a *ArgoCdClient) GetResourceTree(appName string) (*v1alpha1.ApplicationTree, error) {
	file, err := os.OpenFile("performance.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return nil, nil
	}
	startTime := time.Now()
	defer func() {
		fmt.Fprintf(file, "GetResourceTree took %s\n", time.Since(startTime))
	}()
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return nil, a.logger.Errorf("Error creating ArgoCD application client: %v", err)
	}
	query := &application.ResourcesQuery{
		ApplicationName: &appName,
	}

	if tree, found := a.cacheManager.GetResourceTree(a.cfg.Name, appName); found {
		fmt.Fprintf(file, "------------ Cache hit for resource tree -------------\n")
		return tree, nil
	}

	tree, err := appClient.ResourceTree(a.ctx, query)
	if err != nil {
		return nil, a.logger.Errorf("Error getting resource tree for %s: %v", appName, err)
	}

	a.cacheManager.SetResourceTree(a.cfg.Name, appName, tree, a.cacheManager.DefaultExpiration)

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

func (a *ArgoCdClient) SyncApp(appName string) error {
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return a.logger.Errorf("Error getting application client: %v", err)
	}
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
	_, appClient, err := a.client.NewApplicationClient()
	if err != nil {
		return a.logger.Errorf("Error getting application client: %v", err)
	}
	deleteRequest := &application.ApplicationDeleteRequest{
		Name: &appName,
	}
	_, err = appClient.Delete(a.ctx, deleteRequest)
	if err != nil {
		return a.logger.Errorf("Error deleting app %s: %v", appName, err)
	}
	return nil
}
