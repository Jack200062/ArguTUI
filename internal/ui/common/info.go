package common

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type InstanceInfo struct {
	URL  string
	Name string

	AppName      string
	HealthStatus string
	SyncStatus   string
}

func NewInstanceInfo(url, name string) *InstanceInfo {
	return &InstanceInfo{
		Name: name,
		URL:  url,
	}
}

func (info *InstanceInfo) WithAppInfo(appName, healthStatus, syncStatus string) *InstanceInfo {
	info.AppName = appName
	info.HealthStatus = healthStatus
	info.SyncStatus = syncStatus
	return info
}

func (info *InstanceInfo) ClearAppInfo() *InstanceInfo {
	info.AppName = ""
	info.HealthStatus = ""
	info.SyncStatus = ""
	return info
}

func (info *InstanceInfo) FormattedString(keyColor tcell.Color) string {
	if info == nil {
		return "No information available"
	}

	url := info.URL
	if url == "" {
		url = "unknown"
	}

	name := info.Name
	if name == "" {
		name = "unknown"
	}

	result := fmt.Sprintf("[%s]URL[white]: %s\n[%s]Name[white]: %s",
		keyColor, url,
		keyColor, name)

	if info.AppName != "" {
		healthColor := getColorForHealth(info.HealthStatus)
		syncColor := getColorForSync(info.SyncStatus)

		result += fmt.Sprintf("\n[%s]App[white]: %s  [%s]Health[white]: %s  [%s]Sync[white]: %s",
			keyColor, info.AppName,
			keyColor, getColoredText(info.HealthStatus, healthColor),
			keyColor, getColoredText(info.SyncStatus, syncColor))
	}

	return result
}

func getColorForHealth(status string) tcell.Color {
	switch status {
	case "Healthy":
		return tcell.ColorGreen
	case "Progressing":
		return tcell.ColorYellow
	case "Degraded":
		return tcell.ColorRed
	case "Suspended":
		return tcell.ColorBlue
	default:
		return tcell.ColorWhite
	}
}

func getColorForSync(status string) tcell.Color {
	switch status {
	case "Synced":
		return tcell.ColorGreen
	case "OutOfSync":
		return tcell.ColorYellow
	default:
		return tcell.ColorWhite
	}
}

func getColoredText(text string, color tcell.Color) string {
	return fmt.Sprintf("[%s]%s[-:-:-]", color.String(), text)
}
