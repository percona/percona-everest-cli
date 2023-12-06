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

test.describe('Everest CLI install', async () => {
  test.beforeEach(async ({ cli }) => {
    await cli.execute('minikube delete');
    await cli.execute('minikube start');
  });

  test('multiple runs against multiple namespaces', async ({ page, cli, request }) => {
    const verifyClusterResources = async () => {
      await test.step('verify installed operators in k8s', async () => {
        let out = await cli.exec('kubectl get pods --namespace=percona-everest');

        await out.outContainsNormalizedMany([
          'everest-operator-controller-manager',
        ]);

        out = await cli.exec('kubectl get pods --namespace=dev');

        await out.outContainsNormalizedMany([
          'percona-server-mongodb-operator',
        ]);

        await out.outNotContains([
          'percona-postgresql-operator',
          'percona-xtradb-cluster-operator',
        ]);
        out = await cli.exec('kubectl get pods --namespace=prod');

        await out.outContainsNormalizedMany([
          'percona-server-mongodb-operator',
        ]);

        await out.outNotContains([
          'percona-postgresql-operator',
          'percona-xtradb-cluster-operator',
        ]);
      });
    };

    await test.step('run everest install command', async () => {
      const out = await cli.everestExecSkipWizard(
        'install --operator.mongodb=true --operator.postgresql=false --operator.xtradb-cluster=false --monitoring.enable=0 --namespace=prod --namespace=dev',
      );

      await out.assertSuccess();
      await out.outErrContainsNormalizedMany([
        'percona-server-mongodb-operator operator has been installed',
        'everest-operator operator has been installed',
      ]);
    });

    await page.waitForTimeout(10_000);

    await verifyClusterResources();
    await test.step('re-run everest install command', async () => {
      let out = await cli.everestExecSkipWizard(
        'install --operator.mongodb=true --operator.postgresql=true --operator.xtradb-cluster=false --monitoring.enable=0 --namespace=prod --namespace=dev',
      );

      await out.assertSuccess();
      await out.outErrContainsNormalizedMany([
        'percona-server-mongodb-operator operator has been installed',
        'everest-operator operator has been installed',
      ]);
      out = await cli.exec('kubectl -n percona-everest get configmap everest-configuration -o yaml');
      await out.outContainsNormalizedMany([
        'namespaces: prod,dev',
        'operators: percona-server-mongodb-operator,percona-postgresql-operator',
      ]);
    });
    await test.step('re-run everest install command in the different namespace', async () => {
      let out = await cli.everestExecSkipWizard(
        'install --operator.mongodb=true --operator.postgresql=true --operator.xtradb-cluster=false --monitoring.enable=0 --namespace=prod --namespace=dev --namespace=staging',
      );

      await out.assertSuccess();
      await out.outErrContainsNormalizedMany([
        'percona-server-mongodb-operator operator has been installed',
        'everest-operator operator has been installed',
      ]);
      out = await cli.exec('kubectl -n percona-everest get configmap everest-configuration -o yaml');
      await out.outContainsNormalizedMany([
        'namespaces: prod,dev,staging',
        'operators: percona-server-mongodb-operator,percona-postgresql-operator',
      ]);
    });
  });
});
