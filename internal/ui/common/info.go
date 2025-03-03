package common

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type InstanceInfo struct {
	URL  string
	Name string
}

func NewInstanceInfo(url, name string) *InstanceInfo {
	return &InstanceInfo{
		Name: name,
		URL:  url,
	}
}

func (info *InstanceInfo) String() string {
	return fmt.Sprintf("URL: %s\nName: %s", info.URL, info.Name)
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

	return fmt.Sprintf("[%s]URL[white]: %s\n[%s]Name[white]: %s",
		keyColor, url,
		keyColor, name)
}
