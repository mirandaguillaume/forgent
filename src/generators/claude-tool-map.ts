/**
 * Maps Forgent skill tool names to Claude Code tool names.
 * Claude Code tools: Read, Write, Edit, Glob, Grep, Bash, WebFetch, WebSearch, TodoWrite, Task, etc.
 */
const TOOL_MAP: Record<string, string> = {
  // File reading
  read_file: 'Read',
  read: 'Read',
  // File writing
  write_file: 'Write',
  write: 'Write',
  edit_file: 'Edit',
  edit: 'Edit',
  // Search
  grep: 'Grep',
  search: 'Glob',
  find: 'Glob',
  glob: 'Glob',
  // Shell
  bash: 'Bash',
  shell: 'Bash',
  exec: 'Bash',
  terminal: 'Bash',
  // Web
  web_fetch: 'WebFetch',
  http: 'WebFetch',
  fetch: 'WebFetch',
  web_search: 'WebSearch',
  // Task management
  todo: 'TodoWrite',
  // Subagents
  task: 'Task',
  delegate: 'Task',
};

/**
 * Maps a list of Forgent tool names to deduplicated Claude Code tool names.
 */
export function mapToolsToClaude(axTools: string[]): string[] {
  const claudeTools = new Set<string>();
  for (const tool of axTools) {
    const mapped = TOOL_MAP[tool.toLowerCase()];
    if (mapped) {
      claudeTools.add(mapped);
    }
  }
  return [...claudeTools];
}

/**
 * Infers additional Claude Code tools based on security facet.
 */
export function inferToolsFromSecurity(
  filesystem: string,
  network: string,
): string[] {
  const tools: string[] = [];

  // Filesystem access implies read tools
  if (filesystem === 'read-only' || filesystem === 'read-write' || filesystem === 'full') {
    tools.push('Read', 'Glob', 'Grep');
  }
  if (filesystem === 'read-write' || filesystem === 'full') {
    tools.push('Write', 'Edit');
  }
  if (filesystem === 'full') {
    tools.push('Bash');
  }

  // Network access
  if (network === 'allowlist' || network === 'full') {
    tools.push('WebFetch');
  }
  if (network === 'full') {
    tools.push('WebSearch');
  }

  return tools;
}

/**
 * Merges and deduplicates tool lists, maintaining a stable order.
 */
export function mergeToolLists(...lists: string[][]): string[] {
  const ORDER = ['Glob', 'Grep', 'Read', 'Write', 'Edit', 'Bash', 'WebFetch', 'WebSearch', 'TodoWrite', 'Task'];
  const merged = new Set<string>();
  for (const list of lists) {
    for (const tool of list) {
      merged.add(tool);
    }
  }
  // Return in canonical order, then any unknown tools at the end
  const ordered = ORDER.filter((t) => merged.has(t));
  const extra = [...merged].filter((t) => !ORDER.includes(t));
  return [...ordered, ...extra];
}
