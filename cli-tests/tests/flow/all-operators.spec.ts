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
    await cli.execute('docker-compose -f quickstart.yml up -d --force-recreate --renew-anon-volumes');
    await cli.execute('minikube delete');
    await cli.execute('minikube start');
    // await cli.execute('minikube start --apiserver-name=host.docker.internal');
  });

  test('install all operators', async ({ page, cli, request }) => {
    const verifyClusterResources = async () => {
      await test.step('verify installed operators in k8s', async () => {
        const out = await cli.exec('kubectl get pods --namespace=percona-everest-all');

        await out.outContainsNormalizedMany([
          'percona-xtradb-cluster-operator',
          'percona-server-mongodb-operator',
          'percona-postgresql-operator',
          'everest-operator-controller-manager',
        ]);
      });
    };
    const clusterName = `test-${faker.number.int()}`;

    await test.step('run everest install command', async () => {
      const out = await cli.everestExecSkipWizard(
        `install --monitoring.enable=0 --name=${clusterName} --namespace=percona-everest-all`,
      );

      await out.assertSuccess();
      await out.outErrContainsNormalizedMany([
        'percona-xtradb-cluster-operator operator has been installed',
        'percona-server-mongodb-operator operator has been installed',
        'percona-postgresql-operator operator has been installed',
        'everest-operator operator has been installed',
        'Your new password is',
      ]);
    });

    await page.waitForTimeout(10_000);

    await verifyClusterResources();

    await test.step('disable telemetry', async () => {
      // check that the telemetry IS NOT disabled by default
      let out = await cli.exec('kubectl get deployments/percona-xtradb-cluster-operator --namespace=percona-everest-all -o yaml');

      await out.outContains(
        'name: DISABLE_TELEMETRY\n          value: "false"',
      );

      out = await cli.everestExecSkipWizardWithEnv('upgrade --namespace=percona-everest-all', 'DISABLE_TELEMETRY=true');
      await out.assertSuccess();
      await out.outErrContainsNormalizedMany([
        'Subscriptions have been patched\t{"component": "upgrade"}',
      ]);

      await page.waitForTimeout(10_000);
      // check that the telemetry IS disabled
      out = await cli.exec('kubectl get deployments/percona-xtradb-cluster-operator --namespace=percona-everest-all -o yaml');
      await out.outContains(
        'name: DISABLE_TELEMETRY\n          value: "true"',
      );
    });
    await test.step('run everest install command using a different namespace', async () => {
      const install = await cli.everestExecSkipWizard(
        `install --monitoring.enable=0  --namespace=different-everest`,
      );

      await install.assertSuccess();

      const out = await cli.exec('kubectl get clusterrolebinding everest-admin-cluster-role-binding -o yaml');
      await out.assertSuccess();

      await out.outContainsNormalizedMany([
        'namespace: percona-everest-all',
        'namespace: different-everest',
      ]);
      await cli.everestExec('uninstall --namespace=different-everest --assume-yes');
    });

    await test.step('uninstall Everest', async () => {
      let out = await cli.everestExec(
        `uninstall --namespace=percona-everest-all --assume-yes`,
      );

      await out.assertSuccess();
      // check that the deployment does not exist
      out = await cli.exec('kubectl get deploy percona-everest -n percona-everest-all');

      await out.outErrContainsNormalizedMany([
        'Error from server (NotFound): deployments.apps "percona-everest" not found',
      ]);

    });
  });
});
