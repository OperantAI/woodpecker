/*
Copyright 2023 Operant AI
*/
package experiments

import (
	"context"
	"strings"

	"github.com/operantai/woodpecker/internal/categories"
	"github.com/operantai/woodpecker/internal/k8s"
	"github.com/operantai/woodpecker/internal/verifier"

	"gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PostmanCollectionExperimentConfig struct {
	Metadata   ExperimentMetadata `yaml:"metadata"`
	Parameters PostmanCollection  `yaml:"parameters"`
}

type PostmanCollection struct {
	CollectionPath  string   `yaml:"collectionPath"`
	Image           string   `yaml:"image"`
	ImagePullSecret string   `yaml:"imagePullSecret,omitempty"`
	Env             []EnvVar `yaml:"env"`
	Schedule        string   `yaml:"schedule,omitempty"`
	Collection      string   `yaml:"collection,omitempty"`
}

type EnvVar struct {
	EnvKey   string `yaml:"envKey"`
	EnvValue string `yaml:"envValue"`
	EnvType  string `yaml:"envType,omitempty"`
}

func (p *PostmanCollectionExperimentConfig) Type() string {
	return "postman-collection"
}

func (p *PostmanCollectionExperimentConfig) Description() string {
	return "Creates a container in the cluster with a Postman collection to run API tests against services in the cluster."
}

func (p *PostmanCollectionExperimentConfig) Technique() string {
	return categories.MITRE.Discovery.NetworkMapping.Technique
}

func (p *PostmanCollectionExperimentConfig) Tactic() string {
	return categories.MITRE.Discovery.NetworkMapping.Tactic
}

func (p *PostmanCollectionExperimentConfig) Framework() string {
	return string(categories.Mitre)
}

func (p *PostmanCollectionExperimentConfig) Run(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}
	var config PostmanCollectionExperimentConfig
	yamlObj, err := yaml.Marshal(experimentConfig)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Metadata.Name,
			Namespace: config.Metadata.Namespace,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
		},
	}

	secretMap := CreateSecretsFromEnvVars(config.Metadata.Name, config.Parameters.Env)

	for _, secret := range secretMap {
		_, err = client.Clientset.CoreV1().Secrets(config.Metadata.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	clientset := client.Clientset
	_, err = clientset.CoreV1().ServiceAccounts(config.Metadata.Namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Metadata.Name + "-config",
			Namespace: config.Metadata.Namespace,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
		},
		Data: map[string]string{
			"postman-collection.yaml": config.Parameters.Collection,
		},
	}
	_, err = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Metadata.Name,
			Labels: map[string]string{
				"experiment": config.Metadata.Name,
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: config.Parameters.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"experiment": config.Metadata.Name,
								"app":        config.Metadata.Name,
							},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: sa.Name,
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							ImagePullSecrets: []corev1.LocalObjectReference{
								{
									Name: config.Parameters.ImagePullSecret,
								},
							},
							Containers: []corev1.Container{
								{
									Name:            "postman-collection-runner",
									Image:           config.Parameters.Image,
									ImagePullPolicy: corev1.PullAlways,
									Args: []string{
										"run", "-c", config.Parameters.CollectionPath,
									},
									Env: func() []corev1.EnvVar {
										var envVars []corev1.EnvVar
										for _, env := range config.Parameters.Env {
											if env.EnvType == "secret" {
												envVars = append(envVars, corev1.EnvVar{
													Name: env.EnvKey,
													ValueFrom: &corev1.EnvVarSource{
														SecretKeyRef: &corev1.SecretKeySelector{
															Key: env.EnvKey,
															LocalObjectReference: corev1.LocalObjectReference{
																Name: secretMap[env.EnvKey].Name,
															},
														},
													},
												})
												continue
											}
											envVars = append(envVars, corev1.EnvVar{
												Name:  strings.ReplaceAll(env.EnvKey, "_", "-"),
												Value: env.EnvValue,
											})
										}
										return envVars
									}(),
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      configMap.Name,
											MountPath: "/postman",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: configMap.Name,
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMap.Name,
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
	_, err = clientset.BatchV1().CronJobs(config.Metadata.Namespace).Create(ctx, cronjob, metav1.CreateOptions{})
	return err
}

func (p *PostmanCollectionExperimentConfig) Verify(ctx context.Context, experimentConfig *ExperimentConfig) (*verifier.LegacyOutcome, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}
	var config PostmanCollectionExperimentConfig
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return nil, err
	}

	v := verifier.NewLegacy(
		config.Metadata.Name,
		config.Description(),
		config.Framework(),
		config.Tactic(),
		config.Technique(),
	)

	listOptions := metav1.ListOptions{
		LabelSelector: "experiment=" + config.Metadata.Name,
	}

	clientset := client.Clientset
	cronjobs, err := clientset.BatchV1().CronJobs(config.Metadata.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	for _, cj := range cronjobs.Items {
		if cj.Name == config.Metadata.Name {
			v.Success(config.Metadata.Type)
		} else {
			v.Fail(config.Metadata.Type)
		}
	}

	return v.GetOutcome(), nil
}

func (p *PostmanCollectionExperimentConfig) Cleanup(ctx context.Context, experimentConfig *ExperimentConfig) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}
	var config PostmanCollectionExperimentConfig
	yamlObj, _ := yaml.Marshal(experimentConfig)
	err = yaml.Unmarshal(yamlObj, &config)
	if err != nil {
		return err
	}

	clientset := client.Clientset
	err = clientset.BatchV1().CronJobs(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	for _, env := range config.Parameters.Env {
		if env.EnvType == "secret" {
			err = clientset.CoreV1().Secrets(config.Metadata.Namespace).Delete(ctx, env.EnvKey, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	err = clientset.CoreV1().ConfigMaps(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name+"-config", metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = clientset.CoreV1().ServiceAccounts(config.Metadata.Namespace).Delete(ctx, config.Metadata.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
