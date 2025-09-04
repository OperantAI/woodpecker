/*
Copyright 2023 Operant AI
*/
package categories

type Framework string

const (
	Mitre      Framework = "MITRE"
	MitreAtlas Framework = "MITRE-ATLAS"
	OWASP      Framework = "OWASP"
)

// mitreTactics struct to hold MITRE categories
type mitreTactics struct {
	InitialAccess       InitialAccess
	Execution           Execution
	Persistence         Persistence
	PrivilegeEscalation PrivilegeEscalation
	DefenseEvasion      DefenseEvasion
	Credentials         Credentials
	Discovery           Discovery
	LateralMovement     LateralMovement
}

type mitreAtlasTactics struct {
	ResourceDevelopment AMLResourceDevelopment
	Persistence         AMLPersistence
	Exfiltration        AMLExfiltration
}

type mitreEntry struct {
	CategoryID string
	Tactic     string
	Technique  string
}

type AMLResourceDevelopment struct {
	PoisonTrainingData mitreEntry
}

type AMLPersistence struct {
	PoisonTrainingData mitreEntry
}

type AMLExfiltration struct {
	LLMDataLeakage mitreEntry
}

type InitialAccess struct {
	UsingCloudCredentials       mitreEntry
	CompromisedImagesInRegistry mitreEntry
	KubeConfigFile              mitreEntry
	ApplicationVulnerability    mitreEntry
	ExposedSensitiveInterfaces  mitreEntry
}

type Execution struct {
	ExecIntoContainer           mitreEntry
	BashCmdInsideContainer      mitreEntry
	NewContainer                mitreEntry
	ApplicationExploit          mitreEntry
	RCE                         mitreEntry
	SSHServerRunningInContainer mitreEntry
	SidecarInjection            mitreEntry
}

type Persistence struct {
	BackdoorContainer            mitreEntry
	WriteableHostPathMount       mitreEntry
	KubernetesCronJob            mitreEntry
	MaliciousAdmissionController mitreEntry
}

type PrivilegeEscalation struct {
	PrivilegedContainer  mitreEntry
	ClusterAdminBinding  mitreEntry
	HostPathMount        mitreEntry
	AccessCloudResources mitreEntry
}

type DefenseEvasion struct {
	ClearContainerLogs         mitreEntry
	DeleteK8sEvents            mitreEntry
	PodContainerNameSimilarity mitreEntry
	ConnectFromProxyServer     mitreEntry
}

type Credentials struct {
	ListK8sSecrets                             mitreEntry
	MountServicePrincipal                      mitreEntry
	AccessContainerServiceAccount              mitreEntry
	ApplicationCredentialsInConfigurationFiles mitreEntry
	AccessManagedIdentityCredentials           mitreEntry
	MaliciousAdmissionController               mitreEntry
}

type Discovery struct {
	AccessTheK8sApiServer     mitreEntry
	AccessKubeletAPI          mitreEntry
	NetworkMapping            mitreEntry
	AccessKubernetesDashboard mitreEntry
	InstanceMetadataAPI       mitreEntry
}

type LateralMovement struct {
	AccessCloudResources                        mitreEntry
	ContainerServiceAccount                     mitreEntry
	ClusterInternalNetworking                   mitreEntry
	ApplicationsCredentialsInConfigurationFiles mitreEntry
	WritableVolumeMountsOnTheHost               mitreEntry
	CoreDNSPoisoning                            mitreEntry
	ARPPoisoningOrIPSpoofing                    mitreEntry
}

// Exported instances of the categories
var (
	MITRE      mitreTactics
	MITREATLAS mitreAtlasTactics
)

func init() {
	MITRE = mitreTactics{
		InitialAccess{
			UsingCloudCredentials:       mitreEntry{"TA0001", "Initial Access", "Using Cloud Credentials"},
			CompromisedImagesInRegistry: mitreEntry{"TA0001", "Initial Access", "Compromised Images in Registry"},
			KubeConfigFile:              mitreEntry{"TA0001", "Initial Access", "Kube Config File"},
			ApplicationVulnerability:    mitreEntry{"TA0001", "Initial Access", "Application Vulnerability"},
			ExposedSensitiveInterfaces:  mitreEntry{"TA0001", "Initial Access", "Exposed Sensitive Interfaces"},
		},
		Execution{
			ExecIntoContainer:           mitreEntry{"TA0002", "Execution", "Exec Into Container"},
			BashCmdInsideContainer:      mitreEntry{"TA0002", "Execution", "Bash Cmd Inside Container"},
			NewContainer:                mitreEntry{"TA0002", "Execution", "New Container"},
			ApplicationExploit:          mitreEntry{"TA0002", "Execution", "Application Exploit"},
			RCE:                         mitreEntry{"TA0002", "Execution", "Application Exploit"},
			SSHServerRunningInContainer: mitreEntry{"TA0002", "Execution", "SSH Server Running In Container"},
			SidecarInjection:            mitreEntry{"TA0002", "Execution", "Sidecar Injection"},
		},
		Persistence{
			BackdoorContainer:            mitreEntry{"TA0003", "Persistence", "Backdoor Container"},
			WriteableHostPathMount:       mitreEntry{"TA0003", "Persistence", "Writeable Host Path Mount"},
			KubernetesCronJob:            mitreEntry{"TA0003", "Persistence", "Kubernetes Cron Job"},
			MaliciousAdmissionController: mitreEntry{"TA0003", "Persistence", "Malicious Admission Controller"},
		},
		PrivilegeEscalation{
			PrivilegedContainer:  mitreEntry{"TA0004", "Privilege Escalation", "Privileged Container"},
			ClusterAdminBinding:  mitreEntry{"TA0004", "Privilege Escalation", "Cluster Admin Binding"},
			HostPathMount:        mitreEntry{"TA0004", "Privilege Escalation", "Host Path Mount"},
			AccessCloudResources: mitreEntry{"TA0004", "Privilege Escalation", "Access Cloud Resources"},
		},
		DefenseEvasion{
			ClearContainerLogs:         mitreEntry{"TA0005", "Defense Evasion", "Clear Container Logs"},
			DeleteK8sEvents:            mitreEntry{"TA0005", "Defense Evasion", "Delete K8s Events"},
			PodContainerNameSimilarity: mitreEntry{"TA0005", "Defense Evasion", "Pod Container Name Similarity"},
			ConnectFromProxyServer:     mitreEntry{"TA0005", "Defense Evasion", "Connect From Proxy Server"},
		},
		Credentials{
			ListK8sSecrets:                             mitreEntry{"TA0006", "Credential Access", "List K8s Secrets"},
			MountServicePrincipal:                      mitreEntry{"TA0006", "Credential Access", "Mount Service Principal"},
			AccessContainerServiceAccount:              mitreEntry{"TA0006", "Credential Access", "Access Container Service Account"},
			ApplicationCredentialsInConfigurationFiles: mitreEntry{"TA0006", "Credential Access", "Application Credentials In Configuration Files"},
			AccessManagedIdentityCredentials:           mitreEntry{"TA0006", "Credential Access", "Access Managed Identity Credentials"},
			MaliciousAdmissionController:               mitreEntry{"TA0006", "Credential Access", "Malicious Admission Controller"},
		},
		Discovery{
			AccessTheK8sApiServer:     mitreEntry{"TA0007", "Discovery", "Access The K8s API Server"},
			AccessKubeletAPI:          mitreEntry{"TA0007", "Discovery", "Access Kubelet API"},
			NetworkMapping:            mitreEntry{"TA0007", "Discovery", "Network Mapping"},
			AccessKubernetesDashboard: mitreEntry{"TA0007", "Discovery", "Access Kubernetes Dashboard"},
			InstanceMetadataAPI:       mitreEntry{"TA0007", "Discovery", "Instance Metadata API"},
		},
		LateralMovement{
			AccessCloudResources:                        mitreEntry{"TA0008", "Lateral Movement", "Access Cloud Resources"},
			ContainerServiceAccount:                     mitreEntry{"TA0008", "Lateral Movement", "Container Service Account"},
			ClusterInternalNetworking:                   mitreEntry{"TA0008", "Lateral Movement", "Cluster Internal Networking"},
			ApplicationsCredentialsInConfigurationFiles: mitreEntry{"TA0008", "Lateral Movement", "Applications Credentials In Configuration Files"},
			WritableVolumeMountsOnTheHost:               mitreEntry{"TA0008", "Lateral Movement", "Writable Volume Mounts On The Host"},
			CoreDNSPoisoning:                            mitreEntry{"TA0008", "Lateral Movement", "CoreDNS Poisoning"},
			ARPPoisoningOrIPSpoofing:                    mitreEntry{"TA0008", "Lateral Movement", "ARP Poisoning Or IP Spoofing"},
		},
	}

	MITREATLAS = mitreAtlasTactics{
		AMLResourceDevelopment{
			PoisonTrainingData: mitreEntry{"AML.T0020", "Resource Development", "Poison Training Data"},
		},
		AMLPersistence{PoisonTrainingData: mitreEntry{"AML.T0020", "Persistence", "Poison Training Data"}},
		AMLExfiltration{LLMDataLeakage: mitreEntry{"AML.T0057", "Exfiltration", "LLM Data Leakage"}},
	}
}
