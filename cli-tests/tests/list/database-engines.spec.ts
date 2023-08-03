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
import { test, expect } from '@fixtures';
import { waitForDBEngines } from '@tests/support/kubernetes';

let kubernetesId = ''

test.beforeAll(async ({ cli, request }) => {
  const kubernetesList = await request.get('/v1/kubernetes')
  kubernetesId = (await kubernetesList.json())[0].id
  expect(kubernetesId).toBeTruthy()

  // Wait until all dbengines are ready
  await expect.poll(() => waitForDBEngines(cli), {
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
