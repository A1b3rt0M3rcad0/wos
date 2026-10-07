#!/usr/bin/env node
import { operate } from '../lib/installer.mjs';

const help = `wos-skill <install|update|uninstall|list|doctor> [options]
  --agent codex|claude|hermes|openclaw|generic|all  (default: codex)
  --project PATH   Project/workspace root (default: current directory)
  --global         Install in native user discovery directories
  --dir PATH       Explicit discovery directory; required for generic agents
  --dry-run        Show destinations/actions without writing
  --version        Print package version
Use --dir explicitly for runtime profiles with custom discovery paths.
No credentials, MCP configuration, agent launch or lifecycle scripts are installed.`;

try {
  const args = process.argv.slice(2);
  if (!args.length || args.includes('--help') || args.includes('-h')) { console.log(help); }
  else if (args.length === 1 && args[0] === '--version') {
    const { readFile } = await import('node:fs/promises');
    console.log(JSON.parse(await readFile(new URL('../package.json', import.meta.url), 'utf8')).version);
  } else {
    const command = args.shift(); const options = {};
    while (args.length) {
      const a = args.shift();
      if (a === '--global') options.global = true;
      else if (a === '--dry-run') options.dryRun = true;
      else if (['--agent', '--project', '--dir'].includes(a)) {
        if (!args.length || args[0].startsWith('--')) throw new Error(`Missing value for ${a}`);
        options[a.slice(2)] = args.shift();
      } else throw new Error(`Unknown option: ${a}`);
    }
    console.log(JSON.stringify(await operate(command, options), null, 2));
  }
} catch (e) { console.error(`wos-skill: ${e.message}`); process.exitCode = 1; }
