const TOOL_MAP: Record<string, string> = {
  read_file: 'read',
  read: 'read',
  write_file: 'edit',
  write: 'edit',
  edit_file: 'edit',
  edit: 'edit',
  grep: 'search',
  search: 'search',
  find: 'search',
  glob: 'search',
  bash: 'execute',
  shell: 'execute',
  exec: 'execute',
  terminal: 'execute',
  web_fetch: 'web',
  http: 'web',
  fetch: 'web',
  web_search: 'web',
  todo: 'todo',
  task: 'agent',
  delegate: 'agent',
};

export function mapToolsToCopilot(tools: string[]): string[] {
  const mapped = new Set<string>();
  for (const tool of tools) {
    const alias = TOOL_MAP[tool.toLowerCase()];
    if (alias) mapped.add(alias);
  }
  return [...mapped];
}

export function inferCopilotToolsFromSecurity(filesystem: string, network: string): string[] {
  const tools: string[] = [];
  if (filesystem === 'read-only' || filesystem === 'read-write' || filesystem === 'full') {
    tools.push('read', 'search');
  }
  if (filesystem === 'read-write' || filesystem === 'full') {
    tools.push('edit');
  }
  if (filesystem === 'full') {
    tools.push('execute');
  }
  if (network === 'allowlist' || network === 'full') {
    tools.push('web');
  }
  return tools;
}

export function mergeCopilotToolLists(...lists: string[][]): string[] {
  const ORDER = ['read', 'edit', 'search', 'execute', 'web', 'agent', 'todo'];
  const merged = new Set<string>();
  for (const list of lists) {
    for (const tool of list) merged.add(tool);
  }
  const ordered = ORDER.filter((t) => merged.has(t));
  const extra = [...merged].filter((t) => !ORDER.includes(t));
  return [...ordered, ...extra];
}
