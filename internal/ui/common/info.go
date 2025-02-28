package common

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
	return "Url: " + info.URL +
		"\nName: " + info.Name
}
