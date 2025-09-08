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
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OWASPInjectionExperiment tests for OWASP A03:2021 - Injection in EKS/Kubernetes infrastructure
type OWASPInjectionExperiment struct {
	Metadata   ExperimentMetadata       `yaml:"metadata"`
	Parameters OWASPInjectionParams     `yaml:"parameters"`
}

type OWASPInjectionParams struct {
	InjectionType   string            `yaml:"injectionType"`   // "k8s-api", "ingress-annotation", "service-selector", "rbac-name", "configmap-key"
	InjectionVector string            `yaml:"injectionVector"` // The actual payload to inject
	TargetResource  string            `yaml:"targetResource"`  // Resource type to target
	Environment     map[string]string `yaml:"environment"`     // Additional environment variables
}

func (e *OWASPInjectionExperiment) Type() string {
	return "owasp-injection"
}

func (e *OWASPInjectionExperiment) Description() string {
	return "OWASP A03:2021 - Tests for injection vulnerabilities in EKS/Kubernetes API, ingress controllers, RBAC, and resource naming"
}

func (e *OWASPInjectionExperiment) Framework() string {
	return string(categories.OWASP)
}

func (e *OWASPInjectionExperiment) Tactic() string {
	return "Injection"
}

func (e *OWASPInjectionExperiment) Technique() string {
	return "A03:2021"
}

func (e *OWASPInjectionExperiment) Run(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var config OWASPInjectionExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	// Validate required parameters
	if config.Parameters.InjectionType == "" {
		config.Parameters.InjectionType = "k8s-api"
	}
	if config.Parameters.InjectionVector == "" {
		config.Parameters.InjectionVector = "../../../etc/passwd"
	}
	if config.Parameters.TargetResource == "" {
		config.Parameters.TargetResource = "configmap"
	}

	clientset := client.Clientset

	// Create different injection scenarios based on type
	switch config.Parameters.InjectionType {
	case "k8s-api":
		return e.testKubernetesAPIInjection(ctx, clientset, config)
	case "ingress-annotation":
		return e.testIngressAnnotationInjection(ctx, clientset, config)
	case "service-selector":
		return e.testServiceSelectorInjection(ctx, clientset, config)
	case "rbac-name":
		return e.testRBACNameInjection(ctx, clientset, config)
	case "configmap-key":
		return e.testConfigMapKeyInjection(ctx, clientset, config)
	default:
		return e.testKubernetesAPIInjection(ctx, clientset, config)
	}
}

func (e *OWASPInjectionExperiment) testKubernetesAPIInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment) error {
	// Test injection through Kubernetes API resource names
	// Try to create a ConfigMap with an injected name
	injectedName := fmt.Sprintf("%s-%s", config.Metadata.Name, config.Parameters.InjectionVector)
	// Sanitize for K8s DNS-1123 compliance but preserve injection patterns
	sanitizedName := strings.ToLower(injectedName)
	sanitizedName = strings.ReplaceAll(sanitizedName, "/", "-")
	sanitizedName = strings.ReplaceAll(sanitizedName, ".", "-")
	
	// Truncate if too long
	if len(sanitizedName) > 63 {
		sanitizedName = sanitizedName[:63]
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: sanitizedName,
			Labels: map[string]string{
				"experiment":      config.Metadata.Name,
				"injection-test":  "k8s-api",
				"original-vector": config.Parameters.InjectionVector,
			},
		},
		Data: map[string]string{
			"test-key":        "test-value",
			"injection-data":  config.Parameters.InjectionVector,
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	return err
}

func (e *OWASPInjectionExperiment) testIngressAnnotationInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment) error {
	// Test injection through Ingress annotations (common attack vector for nginx/ALB controllers)
	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Metadata.Name,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/configuration-snippet": config.Parameters.InjectionVector,
				"alb.ingress.kubernetes.io/actions.redirect":        config.Parameters.InjectionVector,
				"traefik.ingress.kubernetes.io/router.middlewares":  config.Parameters.InjectionVector,
				"experiment/injection-test":                         "ingress-annotation",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := clientset.NetworkingV1().Ingresses(config.Metadata.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
	return err
}

func (e *OWASPInjectionExperiment) testServiceSelectorInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment) error {
	// Test injection through Service selectors (could affect load balancing)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Metadata.Name,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
			Annotations: map[string]string{
				"experiment/injection-test": "service-selector",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":                    config.Metadata.Name,
				"injection-test":         config.Parameters.InjectionVector,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	_, err := clientset.CoreV1().Services(config.Metadata.Namespace).Create(ctx, service, metav1.CreateOptions{})
	return err
}

func (e *OWASPInjectionExperiment) testRBACNameInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment) error {
	// Test injection through RBAC resource names
	injectedRoleName := fmt.Sprintf("%s-%s", config.Metadata.Name, strings.ReplaceAll(config.Parameters.InjectionVector, "/", "-"))
	
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: injectedRoleName,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
			Annotations: map[string]string{
				"experiment/injection-test":  "rbac-name",
				"experiment/original-vector": config.Parameters.InjectionVector,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	_, err := clientset.RbacV1().Roles(config.Metadata.Namespace).Create(ctx, role, metav1.CreateOptions{})
	return err
}

func (e *OWASPInjectionExperiment) testConfigMapKeyInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment) error {
	// Test injection through ConfigMap keys and values
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Metadata.Name,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
			Annotations: map[string]string{
				"experiment/injection-test": "configmap-key",
			},
		},
		Data: map[string]string{
			"normal-key":                    "normal-value",
			config.Parameters.InjectionVector: "injected-key-test",
			"injected-value":               config.Parameters.InjectionVector,
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	return err
}

func (e *OWASPInjectionExperiment) Verify(ctx context.Context, experimentConfig *ExperimentConfig) (*verifier.LegacyOutcome, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	var config OWASPInjectionExperiment
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

	// Verify based on injection type
	switch config.Parameters.InjectionType {
	case "k8s-api":
		return e.verifyKubernetesAPIInjection(ctx, clientset, config, v)
	case "ingress-annotation":
		return e.verifyIngressAnnotationInjection(ctx, clientset, config, v)
	case "service-selector":
		return e.verifyServiceSelectorInjection(ctx, clientset, config, v)
	case "rbac-name":
		return e.verifyRBACNameInjection(ctx, clientset, config, v)
	case "configmap-key":
		return e.verifyConfigMapKeyInjection(ctx, clientset, config, v)
	default:
		return e.verifyKubernetesAPIInjection(ctx, clientset, config, v)
	}
}

func (e *OWASPInjectionExperiment) verifyKubernetesAPIInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment, v *verifier.LegacyVerifier) (*verifier.LegacyOutcome, error) {
	// Check if the ConfigMap was created with injected content
	configMaps, err := clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "experiment=" + config.Metadata.Name,
	})
	if err != nil {
		v.Fail("k8s-api-resource-created")
		return v.GetOutcome(), nil
	}

	if len(configMaps.Items) > 0 {
		v.Success("k8s-api-resource-created")
		
		cm := configMaps.Items[0]
		// Check if injection vector is present in the resource
		if strings.Contains(cm.Name, strings.ReplaceAll(config.Parameters.InjectionVector, "/", "-")) {
			v.Fail("k8s-api-name-injection-prevented")  // Injection in name succeeded = vulnerable
		} else {
			v.Success("k8s-api-name-injection-prevented")
		}
		
		// Check if injection vector is in data
		if cm.Data["injection-data"] == config.Parameters.InjectionVector {
			v.Success("k8s-api-data-stored")  // This is expected behavior
		}
		
		v.StoreResultOutputs("configmap-name", cm.Name)
		v.StoreResultOutputs("configmap-data", fmt.Sprintf("%v", cm.Data))
	} else {
		v.Fail("k8s-api-resource-created")
	}

	return v.GetOutcome(), nil
}

func (e *OWASPInjectionExperiment) verifyIngressAnnotationInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment, v *verifier.LegacyVerifier) (*verifier.LegacyOutcome, error) {
	// Check if the Ingress was created with injected annotations
	ingress, err := clientset.NetworkingV1().Ingresses(config.Metadata.Namespace).Get(ctx, config.Metadata.Name, metav1.GetOptions{})
	if err != nil {
		v.Fail("ingress-created")
		return v.GetOutcome(), nil
	}

	v.Success("ingress-created")
	
	// Check if dangerous injection patterns were accepted in annotations
	for key, value := range ingress.Annotations {
		if strings.Contains(key, "nginx") || strings.Contains(key, "alb") || strings.Contains(key, "traefik") {
			if value == config.Parameters.InjectionVector {
				// Check for dangerous patterns
				if strings.Contains(value, "$(") || strings.Contains(value, "${") || strings.Contains(value, "`") {
					v.Fail("ingress-annotation-injection-prevented")  // Command injection patterns allowed = vulnerable
				} else if strings.Contains(value, "../") || strings.Contains(value, "/etc/") {
					v.Fail("ingress-annotation-path-traversal-prevented")  // Path traversal allowed = vulnerable
				} else {
					v.Success("ingress-annotation-injection-prevented")
				}
			}
		}
	}
	
	v.StoreResultOutputs("ingress-annotations", fmt.Sprintf("%v", ingress.Annotations))
	return v.GetOutcome(), nil
}

func (e *OWASPInjectionExperiment) verifyServiceSelectorInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment, v *verifier.LegacyVerifier) (*verifier.LegacyOutcome, error) {
	// Check if the Service was created with injected selectors
	service, err := clientset.CoreV1().Services(config.Metadata.Namespace).Get(ctx, config.Metadata.Name, metav1.GetOptions{})
	if err != nil {
		v.Fail("service-created")
		return v.GetOutcome(), nil
	}

	v.Success("service-created")
	
	// Check if injection is in selector
	if selector, exists := service.Spec.Selector["injection-test"]; exists {
		if selector == config.Parameters.InjectionVector {
			v.Success("service-selector-injection-stored")  // This is expected - selectors are just labels
		}
	}
	
	v.StoreResultOutputs("service-selector", fmt.Sprintf("%v", service.Spec.Selector))
	return v.GetOutcome(), nil
}

func (e *OWASPInjectionExperiment) verifyRBACNameInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment, v *verifier.LegacyVerifier) (*verifier.LegacyOutcome, error) {
	// Check if the Role was created with injected name
	roles, err := clientset.RbacV1().Roles(config.Metadata.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "experiment=" + config.Metadata.Name,
	})
	if err != nil {
		v.Fail("rbac-role-created")
		return v.GetOutcome(), nil
	}

	if len(roles.Items) > 0 {
		v.Success("rbac-role-created")
		
		role := roles.Items[0]
		// Check if injection vector affected the role name
		if strings.Contains(role.Name, strings.ReplaceAll(config.Parameters.InjectionVector, "/", "-")) {
			v.Fail("rbac-name-injection-prevented")  // Name injection succeeded = vulnerable
		} else {
			v.Success("rbac-name-injection-prevented")
		}
		
		v.StoreResultOutputs("role-name", role.Name)
	} else {
		v.Fail("rbac-role-created")
	}

	return v.GetOutcome(), nil
}

func (e *OWASPInjectionExperiment) verifyConfigMapKeyInjection(ctx context.Context, clientset *kubernetes.Clientset, config OWASPInjectionExperiment, v *verifier.LegacyVerifier) (*verifier.LegacyOutcome, error) {
	// Check if the ConfigMap was created with injected keys
	configMap, err := clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Get(ctx, config.Metadata.Name, metav1.GetOptions{})
	if err != nil {
		v.Fail("configmap-created")
		return v.GetOutcome(), nil
	}

	v.Success("configmap-created")
	
	// Check if injection vector was used as a key
	if _, exists := configMap.Data[config.Parameters.InjectionVector]; exists {
		// Check for dangerous key patterns
		if strings.Contains(config.Parameters.InjectionVector, "../") || strings.Contains(config.Parameters.InjectionVector, "/etc/") {
			v.Fail("configmap-key-injection-prevented")  // Dangerous key patterns allowed = vulnerable
		} else {
			v.Success("configmap-key-injection-prevented")
		}
	}
	
	v.StoreResultOutputs("configmap-keys", fmt.Sprintf("%v", configMap.Data))
	return v.GetOutcome(), nil
}

func (e *OWASPInjectionExperiment) Cleanup(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	var config OWASPInjectionExperiment
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	clientset := client.Clientset

	// Cleanup based on injection type
	switch config.Parameters.InjectionType {
	case "k8s-api":
		// Delete ConfigMaps
		configMaps, err := clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "experiment=" + config.Metadata.Name,
		})
		if err == nil {
			for _, cm := range configMaps.Items {
				_ = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
			}
		}
	case "ingress-annotation":
		_ = clientset.NetworkingV1().Ingresses(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	case "service-selector":
		_ = clientset.CoreV1().Services(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	case "rbac-name":
		roles, err := clientset.RbacV1().Roles(config.Metadata.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "experiment=" + config.Metadata.Name,
		})
		if err == nil {
			for _, role := range roles.Items {
				_ = clientset.RbacV1().Roles(config.Metadata.Namespace).Delete(ctx, role.Name, metav1.DeleteOptions{})
			}
		}
	case "configmap-key":
		_ = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	}

	return nil
}