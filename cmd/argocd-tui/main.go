package main

import (
	"context"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yml"
	}

	cfg, err := config.Init(configPath, logger)
	if err != nil {
		logger.Fatal("Не удалось инициализировать конфигурацию: %v", err)
	}

	argocdClient := argocd.NewArgoCdClient(cfg, logger, ctx)
	apps, err := argocdClient.GetApps()
	if err != nil {
		logger.Errorf("Ошибка получения приложений: %v", err)
		return
	}

	instanceInfo := common.NewInstanceInfo(cfg.Argocd.Url)
	tviewApp := tview.NewApplication()
	router := ui.NewRouter(tviewApp)

	appList := applicationlist.New(tviewApp, argocdClient, router, instanceInfo, apps)
	router.AddScreen(appList)

	router.SwitchTo(appList.Name())

	if err := tviewApp.Run(); err != nil {
		logger.Fatal("Ошибка запуска TUI: %v", err)
	}

	logger.Info("Завершение работы приложения")

	tviewApp.Stop()
	cancel()
}
