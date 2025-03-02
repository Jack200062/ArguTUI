package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/internal/ui/common"
	"github.com/Jack200062/ArguTUI/internal/ui/screens/applicationlist"
	screens "github.com/Jack200062/ArguTUI/internal/ui/screens/instanceSelection"
	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/rivo/tview"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("ArguTUI version %s (built at %s)\n", Version, BuildDate)
		return
	}
	logger := logging.NewLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yml"
	}

	cfg, err := config.Init(configPath, logger)
	if err != nil {
		logger.Fatal("Failed to init config: %v", err)
	}

	tviewApp := tview.NewApplication()
	router := ui.NewRouter(tviewApp)

	switchToInstance := func(inst *config.Instance) {
		instanceInfo := common.NewInstanceInfo(inst.Url, inst.Name)

		argocdClient := argocd.NewArgoCdClient(inst, logger, ctx)

		apps, err := argocdClient.GetApps()
		if err != nil {
			logger.Errorf("Error getting all applications: %v", err)
			return
		}

		appList := applicationlist.New(tviewApp, argocdClient, router, instanceInfo, apps)
		router.AddScreen(appList)
		router.SwitchTo(appList.Name())
	}

	if len(cfg.Instances) > 1 {
		instanceSelection := screens.NewInstanceSelectionScreen(tviewApp, cfg, router, switchToInstance)
		router.AddScreen(instanceSelection)
		router.SwitchTo(instanceSelection.Name())
	} else if len(cfg.Instances) == 1 {
		switchToInstance(cfg.Instances[0])
	} else {
		logger.Fatal("No instances found in config")
	}

	if err := tviewApp.Run(); err != nil {
		logger.Fatal("Error running ArguTUI: %v", err)
	}

	logger.Info("Closing application")
	tviewApp.Stop()
	cancel()
}
