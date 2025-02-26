package common

type InstanceInfo struct {
	URL string
}

func NewInstanceInfo(url string) *InstanceInfo {
	return &InstanceInfo{
		URL: url,
	}
}

func (info *InstanceInfo) String() string {
	return "Argocd Url: " + info.URL
}
