package cache

import (
	"sync"
	"time"

	"github.com/Jack200062/ArguTUI/internal/models"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	gocache "github.com/patrickmn/go-cache"
)

const (
	AppListPrefix      = "app_list:"
	AppPrefix          = "app:"
	ResourcesPrefix    = "resources:"
	ResourceTreePrefix = "resource_tree:"
)

type CacheManager struct {
	cache             *gocache.Cache
	logger            *logging.Logger
	mutex             sync.RWMutex
	DefaultExpiration time.Duration
}

func NewCacheManager(defaultExpiration, cleanupInterval time.Duration, logger *logging.Logger) *CacheManager {
	return &CacheManager{
		cache:             gocache.New(defaultExpiration, cleanupInterval),
		logger:            logger,
		DefaultExpiration: defaultExpiration,
	}
}

func (s *CacheManager) GetAppList(instanceName string) ([]models.Application, bool) {
	key := AppListPrefix + instanceName
	if data, found := s.cache.Get(key); found {
		if apps, ok := data.([]models.Application); ok {
			return apps, true
		}
	}
	return nil, false
}

func (s *CacheManager) SetAppList(instanceName string, apps []models.Application, expiration time.Duration) {
	key := AppListPrefix + instanceName

	appsCopy := make([]models.Application, len(apps))
	copy(appsCopy, apps)

	s.cache.Set(key, appsCopy, expiration)
	s.logger.Debugf("Cached app list for instance %s (%d apps)", instanceName, len(apps))
}

func (s *CacheManager) InvalidateAppList(instanceName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	key := AppListPrefix + instanceName
	s.cache.Delete(key)
}

func (s *CacheManager) GetResources(instanceName, appName string) ([]models.Resource, bool) {
	key := ResourcesPrefix + instanceName + ":" + appName
	if data, found := s.cache.Get(key); found {
		if resources, ok := data.([]models.Resource); ok {
			return resources, true
		}
	}
	return nil, false
}

func (s *CacheManager) SetResources(instanceName, appName string, resources []models.Resource, expiration time.Duration) {
	key := ResourcesPrefix + instanceName + ":" + appName

	resourcesCopy := make([]models.Resource, len(resources))
	copy(resourcesCopy, resources)

	s.cache.Set(key, resourcesCopy, expiration)
	s.logger.Debugf("Cached %d resources for app %s", len(resources), appName)
}

func (s *CacheManager) GetResourceTree(instanceName, appName string) (*v1alpha1.ApplicationTree, bool) {
	key := ResourceTreePrefix + instanceName + ":" + appName
	if data, found := s.cache.Get(key); found {
		if tree, ok := data.(*v1alpha1.ApplicationTree); ok {
			return tree, true
		}
	}
	return nil, false
}

func (s *CacheManager) SetResourceTree(instanceName, appName string, tree *v1alpha1.ApplicationTree, expiration time.Duration) {

	key := ResourceTreePrefix + instanceName + ":" + appName

	s.cache.Set(key, tree, expiration)
}

func (s *CacheManager) Flush() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.cache.Flush()
}
