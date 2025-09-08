/*
Copyright 2025 Christian (ReK2) F.N.
*/
package experiments

import (
	"context"
	"fmt"

	"github.com/operantai/woodpecker/internal/categories"
	"github.com/operantai/woodpecker/internal/k8s"
	"github.com/operantai/woodpecker/internal/verifier"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// ConfigMapInjectionExperiment tests if malicious code can be injected via ConfigMaps
type ConfigMapInjectionExperiment struct {
	Metadata   ExperimentMetadata       `yaml:"metadata"`
	Parameters ConfigMapInjectionParams `yaml:"parameters"`
}

type ConfigMapInjectionParams struct {
	ScriptContent string `yaml:"scriptContent"`
	MountPath     string `yaml:"mountPath"`
	TargetCommand string `yaml:"targetCommand"`
}

func (e *ConfigMapInjectionExperiment) Type() string {
	return "configmap-code-injection"
}

func (e *ConfigMapInjectionExperiment) Description() string {
	return "Test if malicious code can be injected and executed via ConfigMaps"
}

func (e *ConfigMapInjectionExperiment) Technique() string {
	return "T1055" // Process Injection
}

func (e *ConfigMapInjectionExperiment) Tactic() string {
	return "Execution"
}

func (e *ConfigMapInjectionExperiment) Framework() string {
	return string(categories.Mitre)
}

func (e *ConfigMapInjectionExperiment) Run(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var config ConfigMapInjectionExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	// Validate required parameters
	if config.Parameters.ScriptContent == "" {
		return fmt.Errorf("scriptContent parameter is required")
	}

	if config.Parameters.MountPath == "" {
		return fmt.Errorf("mountPath parameter is required")
	}

	if config.Parameters.TargetCommand == "" {
		return fmt.Errorf("targetCommand parameter is required")
	}

	clientset := client.Clientset

	// Step 1: Create ConfigMap with malicious script
	configMapName := fmt.Sprintf("%s-script", config.Metadata.Name)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
		},
		Data: map[string]string{
			"payload.sh": config.Parameters.ScriptContent,
		},
	}

	_, err = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create configmap: %w", err)
	}

	// Step 2: Create Deployment that mounts and executes the ConfigMap
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Metadata.Name,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": config.Metadata.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"experiment": config.Metadata.Name,
						"app":        config.Metadata.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "target-container",
							Image: "busybox:latest",
							Command: []string{
								"sh",
								"-c",
								fmt.Sprintf("sh %s && tail -f /dev/null",
									config.Parameters.TargetCommand),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "script-volume",
									MountPath: config.Parameters.MountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "script-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
									DefaultMode: pointer.Int32(0755),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = clientset.AppsV1().Deployments(config.Metadata.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		// Clean up ConfigMap if deployment fails
		_ = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

func (e *ConfigMapInjectionExperiment) Verify(ctx context.Context, experimentConfig *ExperimentConfig) (*verifier.LegacyOutcome, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	var config ConfigMapInjectionExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return nil, err
	}

	v := verifier.NewLegacy(
		config.Metadata.Name,
		e.Description(),
		e.Framework(),
		e.Tactic(),
		e.Technique(),
	)

	clientset := client.Clientset

	// Check if the deployment was created successfully
	deployment, err := clientset.AppsV1().Deployments(config.Metadata.Namespace).Get(ctx, config.Metadata.Name, metav1.GetOptions{})
	if err != nil {
		v.Fail("deployment-created")
		return v.GetOutcome(), nil
	}

	if deployment.Status.ReadyReplicas > 0 {
		v.Success("deployment-created")
	} else {
		v.Fail("deployment-created")
	}

	// Check if the malicious script was executed
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", config.Metadata.Name),
	}
	pods, err := clientset.CoreV1().Pods(config.Metadata.Namespace).List(ctx, listOptions)
	if err != nil || len(pods.Items) == 0 {
		v.Fail("script-executed")
		return v.GetOutcome(), nil
	}

	// Try to exec into the pod and check for injection success
	pod := pods.Items[0]
	if pod.Status.Phase == corev1.PodRunning {
		// Check if script executed by looking for the evidence file
		command := []string{"cat", "/tmp/configmap-injection-evidence"}
		stdout, stderr, err := client.ExecuteRemoteCommand(ctx, config.Metadata.Namespace, pod.Name, "target-container", command)

		if err == nil && stdout != "" {
			v.Success("script-executed")
			v.StoreResultOutputs("injection-output", stdout)
		} else {
			v.Fail("script-executed")
			if stderr != "" {
				v.StoreResultOutputs("error", stderr)
			}
		}
	} else {
		v.Fail("script-executed")
	}

	// Check if ConfigMap was successfully mounted
	mountCommand := []string{"ls", "-la", config.Parameters.MountPath}
	mountStdout, _, mountErr := client.ExecuteRemoteCommand(ctx, config.Metadata.Namespace, pod.Name, "target-container", mountCommand)
	if mountErr == nil && mountStdout != "" {
		v.Success("configmap-mounted")
		v.StoreResultOutputs("mount-details", mountStdout)
	} else {
		v.Fail("configmap-mounted")
	}

	return v.GetOutcome(), nil
}

func (e *ConfigMapInjectionExperiment) Cleanup(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var config ConfigMapInjectionExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	clientset := client.Clientset

	// Delete deployment (this will also delete the pods it created)
	err = clientset.AppsV1().Deployments(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		// Only return error if it's not a "not found" error (resource might already be deleted)
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete ConfigMap
	configMapName := fmt.Sprintf("%s-script", config.Metadata.Name)
	err = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		// Only return error if it's not a "not found" error
		return fmt.Errorf("failed to delete configmap: %w", err)
	}

	// Also delete any pods that might be stuck (belt and suspenders approach)
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("experiment=%s", config.Metadata.Name),
	}
	err = clientset.CoreV1().Pods(config.Metadata.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil && !errors.IsNotFound(err) {
		// Non-critical error, just log it
		fmt.Printf("Warning: failed to delete pods: %v\n", err)
	}

	return nil
}
