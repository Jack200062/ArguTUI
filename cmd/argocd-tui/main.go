package main

import (
	"os"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/screens/applicationlist"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/rivo/tview"
)

func main() {
	logger := logging.NewLogger()
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yml"
	}

	cfg, err := config.Init(configPath, logger)
	if err != nil {
		logger.Fatal("failed to initialize config: %v", err)
	} else {
		logger.Infof("successfully initialized config")
	}

	argocdClient := argocd.NewArgoCdClient(cfg, logger)
	/*  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) */
	/* defer cancel() */

	apps, err := argocdClient.GetApps()
	if err != nil {
		logger.Errorf("failed to get applications: %v", err)
		return
	}

	instanceInfo := common.NewInstanceInfo(cfg.Argocd.Url, cfg.Argocd.Token)
	tviewApp := tview.NewApplication()
	router := ui.NewRouter(tviewApp)

	appList := applicationlist.New(tviewApp, argocdClient, router, instanceInfo, apps)
	router.AddScreen(appList)

	router.SwitchTo(appList.Name())
	if err := tviewApp.Run(); err != nil {
		logger.Fatal("failed to run tview app: %v", err)
	}
}
