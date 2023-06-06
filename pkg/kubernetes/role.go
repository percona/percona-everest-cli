package kubernetes

import (
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRole creates a new role.
func (k *Kubernetes) CreateRole(namespace, name string, rules []rbac.PolicyRule) error {
	m := &rbac.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}

	return k.client.ApplyObject(m)
}

// CreateRoleBinding binds a role to a service account.
func (k *Kubernetes) CreateRoleBinding(namespace, name, roleName, serviceAccountName string) error {
	m := &rbac.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbac.Subject{{
			Kind: "ServiceAccount",
			Name: serviceAccountName,
		}},
	}

	return k.client.ApplyObject(m)
}
