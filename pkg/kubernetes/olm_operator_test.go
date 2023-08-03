// Copyright (C) 2020 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package operator contains logic related to kubernetes operators.
package kubernetes

import (
	"context"
	"testing"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/percona/percona-everest-cli/pkg/kubernetes/client"
)

//nolint:paralleltest
func TestInstallOlmOperator(t *testing.T) {
	ctx := context.Background()
	k8sclient := &client.MockKubeClientConnector{}

	l, err := zap.NewDevelopment()
	assert.NoError(t, err)

	olms := NewEmpty(l.Sugar())
	olms.client = k8sclient

	//nolint:paralleltest
	t.Run("Install OLM Operator", func(t *testing.T) {
		k8sclient.On(
			"CreateSubscriptionForCatalog", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		).Return(&v1alpha1.Subscription{}, nil)
		k8sclient.On("GetDeployment", ctx, mock.Anything, "olm").Return(&appsv1.Deployment{}, nil)
		k8sclient.On("ApplyFile", mock.Anything).Return(nil)
		k8sclient.On("DoRolloutWait", ctx, mock.Anything).Return(nil)
		k8sclient.On("GetSubscriptionCSV", ctx, mock.Anything).Return(types.NamespacedName{}, nil)
		k8sclient.On("DoRolloutWait", ctx, mock.Anything).Return(nil)
		err := olms.InstallOLMOperator(ctx)
		assert.NoError(t, err)
	})

	//nolint:paralleltest
	t.Run("Install PSMDB Operator", func(t *testing.T) {
		// Install PSMDB Operator
		subscriptionNamespace := "default"
		operatorGroup := "percona-operators-group"
		catalogSource := "operatorhubio-catalog"
		catalogSourceNamespace := "olm"
		operatorName := "percona-server-mongodb-operator"
		params := InstallOperatorRequest{
			Namespace:              subscriptionNamespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          catalogSource,
			CatalogSourceNamespace: catalogSourceNamespace,
			Channel:                "stable",
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		k8sclient.On("GetOperatorGroup", mock.Anything, subscriptionNamespace, operatorGroup).Return(&v1.OperatorGroup{}, nil)
		mockSubscription := &v1alpha1.Subscription{
			Status: v1alpha1.SubscriptionStatus{
				InstallPlanRef: &corev1.ObjectReference{
					Name: "abcd1234",
				},
			},
		}
		k8sclient.On(
			"CreateSubscriptionForCatalog",
			mock.Anything, subscriptionNamespace, operatorName, "olm",
			catalogSource, operatorName, "stable", "", v1alpha1.ApprovalManual,
		).Return(mockSubscription, nil)
		k8sclient.On("GetSubscription", mock.Anything, subscriptionNamespace, operatorName).Return(mockSubscription, nil)
		mockInstallPlan := &v1alpha1.InstallPlan{}
		k8sclient.On(
			"GetInstallPlan", mock.Anything,
			subscriptionNamespace, mockSubscription.Status.InstallPlanRef.Name,
		).Return(mockInstallPlan, nil)
		k8sclient.On("UpdateInstallPlan", mock.Anything, subscriptionNamespace, mockInstallPlan).Return(mockInstallPlan, nil)
		err := olms.InstallOperator(ctx, params)
		assert.NoError(t, err)
	})
}
