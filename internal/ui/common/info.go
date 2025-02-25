package common

type InstanceInfo struct {
	URL   string
	Token string
}

func NewInstanceInfo(url, token string) *InstanceInfo {
	return &InstanceInfo{
		URL:   url,
		Token: "********",
	}
}

func (info *InstanceInfo) String() string {
	return "ArgoCD Instance:\n" +
		"- URL: " + info.URL + "\n" +
		"- Token: (hidden)"
}
