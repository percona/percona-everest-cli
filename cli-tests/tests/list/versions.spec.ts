import { test, expect } from '@fixtures';

let kubernetesId = ''

test.beforeAll(async ({ request }) => {
  const kubernetesList = await request.get('/v1/kubernetes')
  kubernetesId = (await kubernetesList.json())[0].id
  expect(kubernetesId).toBeTruthy()
})

test.describe('Versions', async () => {
  test('list', async ({ cli }) => {
    const out = await cli.everestExecSilent(`list versions --kubernetes-id ${kubernetesId}`)

    await out.assertSuccess()
    await out.outContainsNormalizedMany([
      'postgresql',
      'psmdb',
      'pxc',
    ])
  })

  test('list json', async ({ cli }) => {
    const out = await cli.everestExecSilent(`--json list versions --kubernetes-id ${kubernetesId}`)

    await out.assertSuccess()
    const res = JSON.parse(out.stdout)
    expect(Array.isArray(res?.postgresql)).toBeTruthy()
    expect(Array.isArray(res?.psmdb)).toBeTruthy()
    expect(Array.isArray(res?.pxc)).toBeTruthy()
    expect(res?.postgresql?.length).toBeTruthy()
    expect(res?.psmdb?.length).toBeTruthy()
    expect(res?.pxc?.length).toBeTruthy()
  })

  test('list supports --type', async ({ cli }) => {
    const out = await cli.everestExecSilent(`list versions --kubernetes-id ${kubernetesId} --type pxc`)

    await out.assertSuccess()
    await out.outContainsNormalizedMany(['pxc'])
    await out.outNotContains(['postgresql', 'psmdb'])
  })
})
