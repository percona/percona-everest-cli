import { CliHelper } from "@tests/helpers/cliHelper"

export async function waitForDBEngines(cli: CliHelper) {
    const out = await cli.execSilent('kubectl -n percona-everest get dbengine -o json')
    await out.assertSuccess()
    
    const res = JSON.parse(out.stdout)
    const installed = res.items.filter(i => i.status.status === 'installed')
    for(const engine of ['pxc', 'psmdb', 'postgresql']) {
      if (!res?.items || res?.items.findIndex(i => i.spec.type === engine) == -1) {
        return `dbengine ${engine} not yet available`
      } 
    }
    
    if (installed.length !== res.items.length) {
      return false
    }

    return true
}