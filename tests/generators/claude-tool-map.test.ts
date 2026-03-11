import { describe, it, expect } from 'vitest';
import { mapToolsToClaude, inferToolsFromSecurity, mergeToolLists } from '../../src/generators/claude-tool-map.js';

describe('mapToolsToClaude', () => {
  it('should map known Forgent tools to Claude Code names', () => {
    expect(mapToolsToClaude(['read_file', 'grep', 'bash'])).toEqual(
      expect.arrayContaining(['Read', 'Grep', 'Bash']),
    );
  });

  it('should be case-insensitive', () => {
    expect(mapToolsToClaude(['Read_File', 'GREP'])).toEqual(
      expect.arrayContaining(['Read', 'Grep']),
    );
  });

  it('should skip unknown tools', () => {
    const result = mapToolsToClaude(['read_file', 'unknown_tool']);
    expect(result).toContain('Read');
    expect(result).toHaveLength(1);
  });

  it('should deduplicate within a single call', () => {
    const result = mapToolsToClaude(['read_file', 'read']);
    expect(result.filter((t) => t === 'Read')).toHaveLength(1);
  });
});

describe('inferToolsFromSecurity', () => {
  it('should infer Read tools for read-only filesystem', () => {
    const tools = inferToolsFromSecurity('read-only', 'none');
    expect(tools).toContain('Read');
    expect(tools).toContain('Glob');
    expect(tools).toContain('Grep');
    expect(tools).not.toContain('Write');
  });

  it('should infer Read + Write tools for read-write filesystem', () => {
    const tools = inferToolsFromSecurity('read-write', 'none');
    expect(tools).toContain('Read');
    expect(tools).toContain('Write');
    expect(tools).toContain('Edit');
    expect(tools).not.toContain('Bash');
  });

  it('should infer all tools for full filesystem', () => {
    const tools = inferToolsFromSecurity('full', 'none');
    expect(tools).toContain('Read');
    expect(tools).toContain('Write');
    expect(tools).toContain('Edit');
    expect(tools).toContain('Bash');
  });

  it('should infer nothing for none filesystem', () => {
    const tools = inferToolsFromSecurity('none', 'none');
    expect(tools).toHaveLength(0);
  });

  it('should infer WebFetch for allowlist network', () => {
    const tools = inferToolsFromSecurity('none', 'allowlist');
    expect(tools).toContain('WebFetch');
    expect(tools).not.toContain('WebSearch');
  });

  it('should infer WebFetch + WebSearch for full network', () => {
    const tools = inferToolsFromSecurity('none', 'full');
    expect(tools).toContain('WebFetch');
    expect(tools).toContain('WebSearch');
  });

  it('should infer nothing for none network', () => {
    const tools = inferToolsFromSecurity('none', 'none');
    expect(tools).toHaveLength(0);
  });
});

describe('mergeToolLists', () => {
  it('should deduplicate across lists', () => {
    const result = mergeToolLists(['Read', 'Grep'], ['Read', 'Bash']);
    expect(result.filter((t) => t === 'Read')).toHaveLength(1);
  });

  it('should maintain canonical order', () => {
    const result = mergeToolLists(['Bash', 'Read', 'Grep']);
    const readIdx = result.indexOf('Read');
    const bashIdx = result.indexOf('Bash');
    expect(readIdx).toBeLessThan(bashIdx);
  });

  it('should append unknown tools at the end', () => {
    const result = mergeToolLists(['Read', 'CustomTool']);
    expect(result[result.length - 1]).toBe('CustomTool');
  });

  it('should handle empty input', () => {
    expect(mergeToolLists([])).toEqual([]);
  });
});
