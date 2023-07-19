import { test, expect } from '@fixtures';

let kubernetesId = ''

test.beforeAll(async ({ request }) => {
  const kubernetesList = await request.get('/v1/kubernetes')
  kubernetesId = (await kubernetesList.json())[0].id
  expect(kubernetesId).toBeTruthy()
})

test.describe('Database engines', async () => {
  test('list', async ({ cli }) => {
    const out = await cli.everestExecSilent(`list databaseengines --kubernetes-id ${kubernetesId}`)

    await out.assertSuccess()
    await out.outContainsNormalizedMany([
      'postgresql',
      'psmdb',
      'pxc',
    ])
  })

  test('list json', async ({ cli }) => {
    const out = await cli.everestExecSilent(`--json list databaseengines --kubernetes-id ${kubernetesId}`)

    await out.assertSuccess()
    const res = JSON.parse(out.stdout)
    expect(res?.postgresql?.version).toBeTruthy()
    expect(res?.psmdb?.version).toBeTruthy()
    expect(res?.pxc?.version).toBeTruthy()
  })
})
