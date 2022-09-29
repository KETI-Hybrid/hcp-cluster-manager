package util

type Config struct {
	Properties struct {
		TimeInWeek []struct {
			Day       string `json:"day"`
			HourSlots []int  `json:"hourSlots"`
		} `json:"timeInWeek"`
		NotAllowedTime []struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"notAllowedTime"`
	} `json:"properties"`
}

type CreateConfig struct {
	Location string `json:"location"`
}

type Output struct {
	Stderr []byte
	Stdout []byte
}

type EKSAPIParameter struct {
	SubscriptionId    string
	ResourceGroupName string
	ResourceName      string
	ApiVersion        string
	Location          string
	ConfigName        string
	ConfigFile        Config
}

type AKSAPIParameter struct {
	ResourceGroupName string
	ClusterName       string
	ConfigName        string
	ConfigFile        Config
	Acr               string
	AKSCluster        string
	BranchName        string
	DoNotWait         string
	Port              string
	BindingSelector   string
	Repository        string
	DisableBrowser    bool
	ListenAddress     string
	ListenPort        string
	Subscription      string
	Location          string
	NodepoolName      string
	Features          []string
	StorageAccount    string
	Addon             AKSAddon            `json:"addon"`
	PodIdentity       AKSPodIdentity      `json:"podIdentity"`
	Install           AKSInstallCLI       `json:"install"`
	K8sConfiguration  AKSk8sConfiguration `json:"k8sConfiguration"`
}

type AKSAddon struct {
	Addon                      string
	AppgwID                    string
	AppgwName                  string
	AppgwSubnetCidr            string
	AppgwSubnetID              string
	AppgwSubnetPrefix          string
	AppgwWatchNamespace        string
	EnableMsiAuthForMonitoring bool
	EnableSecretRotation       bool
	EnableSgxquotehelper       bool
	SubnetName                 string
	WorkspaceResourceID        string
}

type AKSPodIdentity struct {
	Namespace          string
	IdentityResourceID string
	Name               string
	BindingSelector    string
	PodLabels          string
}

type AKSInstallCLI struct {
	BaseSrcURL               string
	ClientVersion            string
	InstallLocation          string
	KubeloginBaseSrcURL      string
	KubeloginInstallLocation string
	KubeloginVersion         string
	Subscription             string
}

type AKSk8sConfiguration struct {
	ClusterType   string
	Name          string
	RepositoryURL string
	Scope         string
}
