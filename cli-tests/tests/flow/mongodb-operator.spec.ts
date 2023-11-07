// percona-everest-cli
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { test } from '@fixtures';
// eslint-disable-next-line import/no-extraneous-dependencies
import { faker } from '@faker-js/faker';
import { apiVerifyClusterExists } from '@support/backend';
import { cliDeleteCluster } from '@support/everest-cli';

test.describe('Everest CLI install operators', async () => {
  test.beforeEach(async ({ cli }) => {
    await cli.execute('docker-compose -f quickstart.yml up -d --force-recreate --renew-anon-volumes');
    await cli.execute('minikube delete');
    await cli.execute('minikube start');
    // await cli.execute('minikube start --apiserver-name=host.docker.internal');
  });

  test('install only mongodb-operator', async ({ page, cli, request }) => {
    const verifyClusterResources = async () => {
      await test.step('verify installed operators in k8s', async () => {
        const out = await cli.exec('kubectl get pods --namespace=percona-everest');

        await out.outContainsNormalizedMany([
          'percona-server-mongodb-operator',
          'everest-operator-controller-manager',
        ]);

        await out.outNotContains([
          'percona-postgresql-operator',
          'percona-xtradb-cluster-operator',
        ]);
      });
    };
    const clusterName = `test-${faker.number.int()}`;

    await test.step('run everest install operators command', async () => {
      const out = await cli.everestExecSkipWizard(
        `install operators --operator.mongodb=true --operator.postgresql=false --operator.xtradb-cluster=false --monitoring.enable=0 --name=${clusterName}`,
      );

      await out.assertSuccess();
      await out.outErrContainsNormalizedMany([
        'percona-server-mongodb-operator operator has been installed',
        'everest-operator operator has been installed',
      ]);
    });

    await page.waitForTimeout(10_000);

    await verifyClusterResources();
    await apiVerifyClusterExists(request, clusterName);
    await cliDeleteCluster(cli, request, clusterName);
    await verifyClusterResources();
  });
});
