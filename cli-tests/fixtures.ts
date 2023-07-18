import { test as base } from '@playwright/test';
import { CliHelper } from '@helpers/cliHelper';

export const test = base.extend<{
  cli: CliHelper
}>({
  cli: async ({}, use) => {
    const app = new CliHelper();

    await use(app);
  },
});

export { expect } from '@playwright/test';
