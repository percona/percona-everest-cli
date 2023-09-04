import { APIRequestContext } from '@playwright/test';
import { expect, test } from '@fixtures';

export async function apiVerifyClusterExists(request: APIRequestContext, clusterName: string) {
  await test.step('verify registered k8s cluster in everest backend', async () => {
    const kubernetesList = await request.get('/v1/kubernetes');
    const clustersList = (await kubernetesList.json());

    expect(clustersList).toHaveLength(1);
    expect(clustersList[0]).toMatchObject({
      name: clusterName,
      namespace: 'percona-everest',
    });

    console.log(`Everest clusters list: 
    ${JSON.stringify(clustersList, null, 2)}`);
  });
}
