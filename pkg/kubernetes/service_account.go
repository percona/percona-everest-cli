package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceAccount creates a new service account.
func (k *Kubernetes) CreateServiceAccount(name, namespace string) error {
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return k.client.ApplyObject(sa)
}

// CreateServiceAccountToken creates a new secret with service account token.
func (k *Kubernetes) CreateServiceAccountToken(serviceAccountName, secretName, namespace string) error {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: serviceAccountName,
			},
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	return k.client.ApplyObject(secret)
}
