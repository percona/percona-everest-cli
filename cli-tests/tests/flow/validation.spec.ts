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

test.describe('Backups', async () => {
  test('backup.name is required', async ({ page, cli, request }) => {
    const out = await cli.everestExecSkipWizard(`install operators --backup.enable --monitoring.enable=0 --name=cluster-name`);

    await out.exitCodeEquals(1);
    await out.outErrContainsNormalizedMany([
      'Backup name cannot be empty',
    ]);
  });
});