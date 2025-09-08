/*
Copyright 2025 Christian (ReK2) F.N.
*/
package experiments

import (
	"context"
	"fmt"
	"strings"

	"github.com/operantai/woodpecker/internal/categories"
	"github.com/operantai/woodpecker/internal/k8s"
	"github.com/operantai/woodpecker/internal/verifier"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OWASPBrokenAccessControlExperiment tests for OWASP A01:2021 - Broken Access Control
type OWASPBrokenAccessControlExperiment struct {
	Metadata   ExperimentMetadata               `yaml:"metadata"`
	Parameters OWASPBrokenAccessControlParams   `yaml:"parameters"`
}

type OWASPBrokenAccessControlParams struct {
	TestType        string   `yaml:"testType"`        // "rbac-bypass", "namespace-access", "service-account"
	TargetNamespace string   `yaml:"targetNamespace"` // Namespace to try accessing
	AccessAttempts  []string `yaml:"accessAttempts"`  // What to try: "list-secrets", "read-configmaps", "exec-pods"
}

func (e *OWASPBrokenAccessControlExperiment) Type() string {
	return "owasp-broken-access-control"
}

func (e *OWASPBrokenAccessControlExperiment) Description() string {
	return "OWASP A01:2021 - Tests for broken access control, RBAC misconfigurations, and unauthorized namespace access in Kubernetes"
}

func (e *OWASPBrokenAccessControlExperiment) Framework() string {
	return string(categories.OWASP)
}

func (e *OWASPBrokenAccessControlExperiment) Tactic() string {
	return "Broken Access Control"
}

func (e *OWASPBrokenAccessControlExperiment) Technique() string {
	return "A01:2021"
}

func (e *OWASPBrokenAccessControlExperiment) Run(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var config OWASPBrokenAccessControlExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	// Validate required parameters
	if config.Parameters.TestType == "" {
		config.Parameters.TestType = "rbac-bypass"
	}
	if config.Parameters.TargetNamespace == "" {
		config.Parameters.TargetNamespace = "kube-system"
	}
	if len(config.Parameters.AccessAttempts) == 0 {
		config.Parameters.AccessAttempts = []string{"list-secrets", "read-configmaps"}
	}

	clientset := client.Clientset

	// Create a service account with minimal permissions
	serviceAccountName := fmt.Sprintf("%s-sa", config.Metadata.Name)
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccountName,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
		},
	}

	_, err = clientset.CoreV1().ServiceAccounts(config.Metadata.Namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Create a pod that will attempt unauthorized access
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Metadata.Name,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
				"app":        config.Metadata.Name,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
			Containers: []corev1.Container{
				{
					Name:  "access-tester",
					Image: "bitnami/kubectl:latest",
					Command: []string{
						"sh",
						"-c",
						generateAccessTestScript(config.Parameters),
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	_, err = clientset.CoreV1().Pods(config.Metadata.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		// Clean up service account if pod creation fails
		_ = clientset.CoreV1().ServiceAccounts(config.Metadata.Namespace).Delete(ctx, serviceAccountName, metav1.DeleteOptions{})
		return fmt.Errorf("failed to create pod: %w", err)
	}

	return nil
}

func generateAccessTestScript(params OWASPBrokenAccessControlParams) string {
	var commands []string
	
	commands = append(commands, "echo '=== OWASP A01:2021 - Broken Access Control Test ===' > /tmp/access-test-results")
	commands = append(commands, "echo 'Reference: https://owasp.org/Top10/A01_2021-Broken_Access_Control/' >> /tmp/access-test-results")
	commands = append(commands, fmt.Sprintf("echo 'Test Started: %s' >> /tmp/access-test-results", "`date`"))
	commands = append(commands, fmt.Sprintf("echo 'Testing unauthorized access to namespace: %s' >> /tmp/access-test-results", params.TargetNamespace))
	commands = append(commands, fmt.Sprintf("echo 'Running as: %s' >> /tmp/access-test-results", "`whoami`"))
	commands = append(commands, fmt.Sprintf("echo 'Service Account: %s' >> /tmp/access-test-results", "`cat /var/run/secrets/kubernetes.io/serviceaccount/namespace 2>/dev/null`/`basename /var/run/secrets/kubernetes.io/serviceaccount/..data`"))
	commands = append(commands, "echo '' >> /tmp/access-test-results")

	for _, attempt := range params.AccessAttempts {
		switch attempt {
		case "list-secrets":
			commands = append(commands, fmt.Sprintf("echo '[*] Attempting to list secrets in %s' >> /tmp/access-test-results", params.TargetNamespace))
			commands = append(commands, fmt.Sprintf("kubectl get secrets -n %s >> /tmp/access-test-results 2>&1 && echo 'VULNERABLE: Can list secrets!' >> /tmp/access-test-results || echo 'SECURE: Cannot list secrets' >> /tmp/access-test-results", params.TargetNamespace))
		case "read-configmaps":
			commands = append(commands, fmt.Sprintf("echo '[*] Attempting to read configmaps in %s' >> /tmp/access-test-results", params.TargetNamespace))
			commands = append(commands, fmt.Sprintf("kubectl get configmaps -n %s >> /tmp/access-test-results 2>&1 && echo 'VULNERABLE: Can read configmaps!' >> /tmp/access-test-results || echo 'SECURE: Cannot read configmaps' >> /tmp/access-test-results", params.TargetNamespace))
		case "exec-pods":
			commands = append(commands, fmt.Sprintf("echo '[*] Attempting to list pods in %s' >> /tmp/access-test-results", params.TargetNamespace))
			commands = append(commands, fmt.Sprintf("kubectl get pods -n %s >> /tmp/access-test-results 2>&1 && echo 'VULNERABLE: Can list pods!' >> /tmp/access-test-results || echo 'SECURE: Cannot list pods' >> /tmp/access-test-results", params.TargetNamespace))
		}
		commands = append(commands, "echo '' >> /tmp/access-test-results")
	}

	// Test default service account token
	commands = append(commands, "echo '[*] Checking service account token access' >> /tmp/access-test-results")
	commands = append(commands, "if [ -f /var/run/secrets/kubernetes.io/serviceaccount/token ]; then echo 'Service account token found' >> /tmp/access-test-results; else echo 'No service account token' >> /tmp/access-test-results; fi")
	
	// Log summary
	commands = append(commands, "echo '' >> /tmp/access-test-results")
	commands = append(commands, "echo '=== Summary ===' >> /tmp/access-test-results")
	commands = append(commands, fmt.Sprintf("echo 'Total access attempts: %d' >> /tmp/access-test-results", len(params.AccessAttempts)))
	commands = append(commands, "echo 'Check above for VULNERABLE vs SECURE results' >> /tmp/access-test-results")
	commands = append(commands, fmt.Sprintf("echo 'Test Completed: %s' >> /tmp/access-test-results", "`date`"))
	commands = append(commands, "echo '=== End of OWASP A01:2021 Test ===' >> /tmp/access-test-results")
	
	// Keep pod alive for verification
	commands = append(commands, "sleep 300")

	return strings.Join(commands, " && ")
}

func (e *OWASPBrokenAccessControlExperiment) Verify(ctx context.Context, experimentConfig *ExperimentConfig) (*verifier.LegacyOutcome, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	var config OWASPBrokenAccessControlExperiment
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

	// Check if pod was created
	pod, err := clientset.CoreV1().Pods(config.Metadata.Namespace).Get(ctx, config.Metadata.Name, metav1.GetOptions{})
	if err != nil {
		v.Fail("pod-created")
		return v.GetOutcome(), nil
	}

	v.Success("pod-created")

	// Wait for pod to be running or completed
	if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded {
		// Check test results
		command := []string{"cat", "/tmp/access-test-results"}
		stdout, stderr, err := client.ExecuteRemoteCommand(ctx, config.Metadata.Namespace, pod.Name, "access-tester", command)
		
		if err == nil && stdout != "" {
			v.StoreResultOutputs("access-test-results", stdout)
			
			// Parse results to determine vulnerabilities
			if strings.Contains(stdout, "VULNERABLE: Can list secrets") {
				v.Fail("rbac-secrets-protected")
			} else if strings.Contains(stdout, "SECURE: Cannot list secrets") {
				v.Success("rbac-secrets-protected")
			}

			if strings.Contains(stdout, "VULNERABLE: Can read configmaps") {
				v.Fail("rbac-configmaps-protected")
			} else if strings.Contains(stdout, "SECURE: Cannot read configmaps") {
				v.Success("rbac-configmaps-protected")
			}

			if strings.Contains(stdout, "VULNERABLE: Can list pods") {
				v.Fail("rbac-pods-protected")
			} else if strings.Contains(stdout, "SECURE: Cannot list pods") {
				v.Success("rbac-pods-protected")
			}

			v.Success("access-test-completed")
		} else {
			v.Fail("access-test-completed")
			if stderr != "" {
				v.StoreResultOutputs("error", stderr)
			}
		}
	} else {
		v.Fail("access-test-completed")
		v.StoreResultOutputs("pod-status", string(pod.Status.Phase))
	}

	return v.GetOutcome(), nil
}

func (e *OWASPBrokenAccessControlExperiment) Cleanup(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var config OWASPBrokenAccessControlExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	clientset := client.Clientset

	// Delete pod
	err = clientset.CoreV1().Pods(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("Warning: failed to delete pod: %v\n", err)
	}

	// Delete service account
	serviceAccountName := fmt.Sprintf("%s-sa", config.Metadata.Name)
	err = clientset.CoreV1().ServiceAccounts(config.Metadata.Namespace).Delete(ctx, serviceAccountName, metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("Warning: failed to delete service account: %v\n", err)
	}

	return nil
}