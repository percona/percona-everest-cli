import { test, expect } from '@fixtures';

let kubernetesId = ''

test.beforeAll(async ({ cli, request }) => {
  const kubernetesList = await request.get('/v1/kubernetes')
  kubernetesId = (await kubernetesList.json())[0].id
  expect(kubernetesId).toBeTruthy()

  // Wait until all dbengines are ready
  await expect.poll(async () => {
    const out = await cli.execSilent('kubectl -n percona-everest get dbengine -o json')
    await out.assertSuccess()
    
    const res = JSON.parse(out.stdout)
    const installed = res.items.filter(i => i.status.status === 'installed')
    for(const engine of ['pxc', 'psmdb', 'postgresql']) {
      if (res.items.findIndex(i => i.spec.type === engine) == -1) {
        return `dbengine ${engine} not yet available`
      } 
    }
    
    if (installed.length !== res.items.length) {
      return false
    }

    return true
  }, {
    message: 'dbengine not yet installed',
    intervals: [1000],
    timeout: 240 * 1000
  }).toBe(true)
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
