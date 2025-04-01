package applicationlist

import (
	"sync"
	"time"

	"github.com/Jack200062/ArguTUI/internal/cache"
	"github.com/Jack200062/ArguTUI/internal/models"
	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
)

type PreloadManager struct {
	numWorkers   int
	client       *argocd.ArgoCdClient
	cacheManager *cache.CacheManager
	taskQueue    chan models.Application
	wg           sync.WaitGroup
	instanceName string
}

func NewPreloadManager(numWorkers int, client *argocd.ArgoCdClient, cacheManager *cache.CacheManager, instanceName string) *PreloadManager {
	return &PreloadManager{
		numWorkers:   numWorkers,
		client:       client,
		cacheManager: cacheManager,
		taskQueue:    make(chan models.Application, 1000),
		instanceName: instanceName,
	}
}

func (pm *PreloadManager) Init() {
	for i := 0; i < pm.numWorkers; i++ {
		pm.wg.Add(1)
		go func() {
			defer pm.wg.Done()
			for task := range pm.taskQueue {
				appName := task.Name
				pm.preloadResources(appName)
			}
		}()
	}
}

func (pm *PreloadManager) preloadResources(appName string) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resources, err := pm.client.GetAppResources(appName)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		pm.cacheManager.SetResources(pm.instanceName, appName, resources, pm.cacheManager.DefaultExpiration)
		break
	}
}

func (pm *PreloadManager) StartPreload(apps []models.Application) {
	for _, app := range apps {
		select {
		case pm.taskQueue <- app:
		default:
			// If the task queue is full, wait for a bit before trying again
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (pm *PreloadManager) Stop() {
	close(pm.taskQueue)
	pm.wg.Wait()
}
