import { describe, it, expect } from 'vitest';
import { mapToolsToCopilot, inferCopilotToolsFromSecurity, mergeCopilotToolLists } from '../../src/generators/copilot-tool-map.js';

describe('mapToolsToCopilot', () => {
  it('should map known Forgent tools to Copilot aliases', () => {
    expect(mapToolsToCopilot(['read_file', 'grep', 'bash'])).toEqual(
      expect.arrayContaining(['read', 'search', 'execute']),
    );
  });

  it('should deduplicate tools', () => {
    const result = mapToolsToCopilot(['read_file', 'read']);
    expect(result.filter((t) => t === 'read')).toHaveLength(1);
  });

  it('should ignore unknown tools', () => {
    expect(mapToolsToCopilot(['unknown_tool'])).toEqual([]);
  });

  it('should map web tools to single alias', () => {
    const result = mapToolsToCopilot(['web_fetch', 'web_search']);
    expect(result).toContain('web');
    expect(result.filter((t) => t === 'web')).toHaveLength(1);
  });

  it('should map delegation to agent', () => {
    expect(mapToolsToCopilot(['task', 'delegate'])).toEqual(
      expect.arrayContaining(['agent']),
    );
  });
});

describe('inferCopilotToolsFromSecurity', () => {
  it('should infer read tools from read-only filesystem', () => {
    const tools = inferCopilotToolsFromSecurity('read-only', 'none');
    expect(tools).toContain('read');
    expect(tools).toContain('search');
    expect(tools).not.toContain('edit');
  });

  it('should infer edit from read-write', () => {
    const tools = inferCopilotToolsFromSecurity('read-write', 'none');
    expect(tools).toContain('edit');
  });

  it('should infer execute from full filesystem', () => {
    const tools = inferCopilotToolsFromSecurity('full', 'none');
    expect(tools).toContain('execute');
  });

  it('should infer web from network access', () => {
    const tools = inferCopilotToolsFromSecurity('none', 'full');
    expect(tools).toContain('web');
  });

  it('should return empty for no access', () => {
    expect(inferCopilotToolsFromSecurity('none', 'none')).toEqual([]);
  });
});

describe('mergeCopilotToolLists', () => {
  it('should merge and deduplicate', () => {
    const result = mergeCopilotToolLists(['read', 'edit'], ['read', 'execute']);
    expect(new Set(result).size).toBe(result.length);
    expect(result).toContain('read');
    expect(result).toContain('edit');
    expect(result).toContain('execute');
  });

  it('should maintain canonical order', () => {
    const result = mergeCopilotToolLists(['execute', 'read']);
    expect(result.indexOf('read')).toBeLessThan(result.indexOf('execute'));
  });
});
