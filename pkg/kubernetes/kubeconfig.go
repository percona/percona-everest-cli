package kubernetes

import corev1 "k8s.io/api/core/v1"

// GenerateKubeConfigWithToken returns a kubeconfig with the token as provided in the secret.
func (k *Kubernetes) GenerateKubeConfigWithToken(user string, secret *corev1.Secret) (string, error) {
	kubeConfig, err := k.client.GenerateKubeConfigWithToken(user, secret)
	if err != nil {
		k.l.Errorf("failed generating kubeconfig: %v", err)
		return "", err
	}

	return string(kubeConfig), nil
}
