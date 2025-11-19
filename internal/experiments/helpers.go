package experiments

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/operantai/woodpecker/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const tmpFileDir = "/tmp/woodpecker"

func createTempFile(experimentType, experiment string) (*os.File, error) {
	if _, err := os.Stat(tmpFileDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(tmpFileDir, 0700); err != nil {
				return nil, err
			}
		}
	}
	file, err := os.CreateTemp(tmpFileDir, fmt.Sprintf("%s-%s", experimentType, experiment))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func getTempFileContentsForExperiment(experimentType, experiment string) ([][]byte, error) {
	var contents [][]byte
	files, err := getTempFilesForExperiment(experimentType, experiment)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}
	return contents, nil
}

func getTempFilesForExperiment(experimentType, experiment string) ([]string, error) {
	d, err := os.Open(tmpFileDir)
	if err != nil {
		return nil, err
	}
	files, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	var fullPaths []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), fmt.Sprintf("%s-%s", experimentType, experiment)) {
			fullPaths = append(fullPaths, filepath.Join(tmpFileDir, file.Name()))
		}
	}
	return fullPaths, nil
}

func removeTempFilesForExperiment(experimentType, experiment string) error {
	files, err := getTempFilesForExperiment(experimentType, experiment)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return err
		}

	}
	return nil
}

const WoodpeckerAI = "woodpecker-ai-verifier"

func isWoodpeckerAIDockerComponentPresent(ctx context.Context, client *dockerClient.Client) bool {
	containers, err := client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return false
	}
	for _, container := range containers {
		if strings.Contains(container.Image, WoodpeckerAI) {
			return true
		}
	}
	return false
}

func isWoodpeckerAIK8sComponentPresent(ctx context.Context, client *k8s.Client, namespace string) bool {
	_, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, WoodpeckerAI, metav1.GetOptions{})
	return err == nil
}

func getAIComponentAddrs(ctx context.Context, config *ExperimentConfig) (string, string, error) {
	aiAppAddr := "127.0.0.1"
	aiAppPort := 9000
	var verifierAddr string
	var verifierPort int
	switch config.Metadata.Namespace {
	case "local":
		/*client, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
		if err != nil {
			return "", "", err
		}
		defer client.Close()
		if !isWoodpeckerAIDockerComponentPresent(ctx, client) {
			return "", "", errors.New("Error in checking for woodpecker AI component to run AI experiments. Is it deployed? Deploy with woodpecker component install command.")
		}*/
		verifierAddr = "127.0.0.1"
		verifierPort = 8000
	default:
		client, err := k8s.NewClient()
		if err != nil {
			return "", "", err
		}
		if !isWoodpeckerAIK8sComponentPresent(ctx, client, config.Metadata.Namespace) {
			return "", "", errors.New("Error in checking for woodpecker AI component to run AI experiments. Is it deployed? Deploy with woodpecker component install command.")
		}
		verifierPortForwarder := client.NewPortForwarder(ctx)
		if err != nil {
			return "", "", err
		}
		defer verifierPortForwarder.Stop()
		verifierForwardedPort, err := verifierPortForwarder.Forward(config.Metadata.Namespace, fmt.Sprintf("app=%s", WoodpeckerAI), 8000)
		if err != nil {
			return "", "", err
		}

		aiAppPortForwarder := client.NewPortForwarder(ctx)
		if err != nil {
			return "", "", err
		}
		defer aiAppPortForwarder.Stop()
		aiAppPortForwardedPort, err := verifierPortForwarder.Forward(config.Metadata.Namespace, fmt.Sprintf("app=%s", "woodpecker-ai-app"), 8081)
		if err != nil {
			return "", "", err
		}

		aiAppAddr = aiAppPortForwarder.Addr()
		aiAppPort = int(aiAppPortForwardedPort.Local)
		verifierAddr = verifierPortForwarder.Addr()
		verifierPort = int(verifierForwardedPort.Local)
	}
	return fmt.Sprintf("%s:%d", verifierAddr, verifierPort), fmt.Sprintf("%s:%d", aiAppAddr, aiAppPort), nil
}

func CreateSecretsFromEnvVars(experimentName string, envVars []EnvVar) map[string]*corev1.Secret {
	var secrets map[string]*corev1.Secret = make(map[string]*corev1.Secret)
	for _, env := range envVars {
		if env.EnvType == "secret" {
			secretName := strings.ReplaceAll(env.EnvKey, "_", "-")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName,
					Labels: map[string]string{
						"experiment": experimentName,
					},
				},
				Data: map[string][]byte{
					env.EnvKey: []byte(env.EnvValue),
				},
				Type: corev1.SecretTypeOpaque,
			}
			secrets[env.EnvKey] = secret
		}
	}
	return secrets
}
