package experiments

// ExperimentsRegistry is a list of all experiments
var ExperimentsRegistry = []Experiment{
	&PrivilegedContainerExperimentConfig{},
	&HostPathMountExperimentConfig{},
	&ClusterAdminBindingExperimentConfig{},
	&ContainerSecretsExperimentConfig{},
	&RemoteExecuteAPIExperimentConfig{},
	&ExecuteAPIExperimentConfig{},
	&ListK8sSecretsConfig{},
	&LLMDataLeakageExperiment{},
	&LLMDataPoisoningExperiment{},
	&KubeExec{},
	&ConfigMapInjectionExperiment{},
	&OWASPBrokenAccessControlExperiment{},
}

func ListExperiments() map[string]string {
	experiments := make(map[string]string)
	for _, experiment := range ExperimentsRegistry {
		experiments[experiment.Type()] = experiment.Description()
	}
	return experiments
}
