import { CliHelper } from '@helpers/cliHelper';
import { APIRequestContext } from '@playwright/test';
import { expect, test } from '@fixtures';

export async function cliDeleteCluster(cli: CliHelper, request: APIRequestContext, clusterName: string) {
  await test.step('run delete cluster command', async () => {
    const out = await cli.everestExecSilent(`delete cluster --name=${clusterName} --assume-yes`);

    await out.assertSuccess();
    await out.outErrContainsNormalizedMany([
      `Deleting all Kubernetes monitoring resources in Kubernetes cluster "${clusterName}"`,
      `Deleting Kubernetes cluster "${clusterName}" from Everest`,
      `Kubernetes cluster "${clusterName}" has been deleted successfully`,
    ]);
  });

  await test.step('verify k8s cluster was removed from everest backend', async () => {
    const kubernetesList = await request.get('/v1/kubernetes');
    const clustersList = (await kubernetesList.json());

    expect(clustersList).toHaveLength(0);
  });
}
