import { test } from '@fixtures';
import Output from '@support/output';
import shell from 'shelljs';

export class CliHelper {
  constructor(private pathToBinary = '../bin/everest') {}

  /**
   * Shell(sh) exec() wrapper to use outside {@link test}
   * returns handy {@link Output} object.
   *
   * @param       command   sh command to execute
   * @return      {@link Output} instance
   */
  async execute(command: string): Promise<Output> {
    const { stdout, stderr, code } = shell.exec(command.replace(/(\r\n|\n|\r)/gm, ''), {
      silent: false,
    });

    // if (stdout.length > 0) console.log(`Out: "${stdout}"`);

    // if (stderr.length > 0) console.log(`Error: "${stderr}"`);

    return new Output(command, code, stdout, stderr);
  }

  /**
   * Shell(sh) exec() wrapper to return handy {@link Output} object.
   *
   * @param       command   sh command to execute
   * @return      {@link Output} instance
   */
  async exec(command: string) {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return test.step(`Run "${command}" command`, async () => {
      return this.execute(command);
    });
  }

  /**
   * Same as {@link exec()} but with "--skip-wizard" suffix.
   *
   * @param       command   sh command to execute
   * @return      {@link Output} instance
   */
  async everestExecSkipWizard(command: string) {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return test.step(`Run "${command}" command with --skip-wizard`, async () => {
      return this.execute(`${this.pathToBinary} ${command} --skip-wizard`);
    });
  }

  /**
   * Silent Shell(sh) exec() wrapper to return handy {@link Output} object.
   * Provides no logs to skip huge outputs.
   *
   * @param       command   sh command to execute
   * @return      {@link Output} instance
   */
  async execSilent(command: string): Promise<Output> {
    const { stdout, stderr, code } = await test.step(`Run "${command}" command`, async () => {
      return shell.exec(command.replace(/(\r\n|\n|\r)/gm, ''), {
        silent: true,
      });
    });

    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    return new Output(command, code, stdout, stderr);
  }

  /**
   * Silent Shell(sh) exec() wrapper to return handy {@link Output} object.
   * Provides no logs to skip huge outputs.
   *
   * @param       command   sh command to execute
   * @return      {@link Output} instance
   */
  async everestExecSilent(command: string): Promise<Output> {
    const { stdout, stderr, code } = await test.step(`Run everest "${command}" command`, async () => {
      return shell.exec(`${this.pathToBinary} ${command}`.replace(/(\r\n|\n|\r)/gm, ''), {
        silent: true,
      });
    });

    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    return new Output(command, code, stdout, stderr);
  }
}
