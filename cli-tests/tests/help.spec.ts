// percona-everest-backend
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

test.describe('Everest CLI "--help" validation', async () => {
  test('top level --help', async ({ cli }) => {
    const out = await cli.everestExecSilent('--help');

    await out.assertSuccess();
    await out.outContainsNormalizedMany([
      'Usage:',
      'everest [command]',
      'Available Commands:',
      'completion Generate the autocompletion script for the specified shell',
      'help Help about any command',
      'install',
      'Flags:',
      '-h, --help help for everest',
      'Use "everest [command] --help" for more information about a command.',
    ]);
  });

  test('top level help', async ({ cli }) => {
    const out = await cli.everestExecSilent('help');

    await out.assertSuccess();
    await out.outContainsNormalizedMany([
      'Usage:',
      'everest [command]',
      'Available Commands:',
      'completion Generate the autocompletion script for the specified shell',
      'help Help about any command',
      'install',
      'Flags:',
      '-h, --help help for everest',
      'Use "everest [command] --help" for more information about a command.',
    ]);
  });

  test('completion --help', async ({ cli }) => {
    const out = await cli.everestExecSilent('completion --help');

    await out.assertSuccess();
    await out.outContainsNormalizedMany([
      'Generate the autocompletion script for everest for the specified shell.',
      "See each sub-command's help for details on how to use the generated script.",
      'Usage:',
      'everest completion [command]',
      'Available Commands:',
      'bash Generate the autocompletion script for bash',
      'fish Generate the autocompletion script for fish',
      'powershell Generate the autocompletion script for powershell',
      'zsh Generate the autocompletion script for zsh',
      'Flags:',
      '-h, --help help for completion',
      'Use "everest completion [command] --help" for more information about a command.',
    ]);
  });

  test('completion bash --help', async ({ cli }) => {
    const out = await cli.everestExecSilent('completion bash --help');

    await out.assertSuccess();
    await out.outContainsNormalizedMany([
      'Generate the autocompletion script for the bash shell.',
      "This script depends on the 'bash-completion' package.",
      "If it is not installed already, you can install it via your OS's package manager.",
      'To load completions in your current shell session:',
      'source <(everest completion bash)',
      'To load completions for every new session, execute once:',
      '#### Linux:',
      'everest completion bash > /etc/bash_completion.d/everest',
      '#### macOS:',
      'everest completion bash > $(brew --prefix)/etc/bash_completion.d/everest',
      'You will need to start a new shell for this setup to take effect.',
      'Usage:',
      'everest completion bash',
      'Flags:',
      '-h, --help help for bash',
      '--no-descriptions disable completion descriptions',
    ]);
  });

  test('install --help', async ({ cli }) => {
    const out = await cli.everestExecSilent('install --help');

    await out.assertSuccess();
    await out.outContainsNormalizedMany([
      'Usage:',
      'everest install [command]',
      'Available Commands:',
      'operators',
      'Flags:',
      '-h, --help help for install',
      'Use "everest install [command] --help" for more information about a command.',
    ]);
  });

  test('install operators --help', async ({ cli }) => {
    const out = await cli.everestExecSilent('install operators --help');

    await out.assertSuccess();
    await out.outContainsNormalizedMany([
      'Usage:',
      'everest install operators [flags]',
      'Flags:',
      '--backup.access-key string Backup access key',
      '--backup.bucket string Backup bucket',
      '--backup.enable Enable backups',
      '--backup.endpoint string Backup endpoint URL',
      '--backup.region string Backup region',
      '--backup.secret-key string Backup secret key',
      '--channel.everest string Channel for Everest operator (default "stable-v0")',
      '--channel.mongodb string Channel for MongoDB operator (default "stable-v1")',
      '--channel.postgresql string Channel for PostgreSQL operator (default "fast-v2")',
      '--channel.victoria-metrics string Channel for VictoriaMetrics operator (default "stable-v0")',
      '--channel.xtradb-cluster string Channel for XtraDB Cluster operator (default "stable-v1")',
      '--everest.endpoint string Everest endpoint URL (default "http://127.0.0.1:8081")',
      '-h, --help help for operators',
      '-k, --kubeconfig string Path to a kubeconfig (default "~/.kube/config")',
      '-m, --monitoring.enable Enable monitoring (default true)',
      '--monitoring.pmm.endpoint string PMM endpoint URL (default "http://127.0.0.1")',
      '--monitoring.pmm.password string PMM password (default "password")',
      '--monitoring.pmm.username string PMM username (default "admin")',
      '--monitoring.type string Monitoring type (default "pmm")',
      '-n, --name string Kubernetes cluster name',
      '--namespace string Namespace into which Percona Everest components are deployed to (default "percona-everest")',
      '--operator.mongodb Install MongoDB operator (default true)',
      '--operator.postgresql Install PostgreSQL operator (default true)',
      '--operator.xtradb-cluster Install XtraDB Cluster operator (default true)',
      '--skip-wizard Skip installation wizard',
    ]);
  });
});
