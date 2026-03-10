import { parse } from 'yaml';
import { validateSkill, validateAgent } from '../model/schema.js';
import type { SkillBehavior } from '../model/skill-behavior.js';
import type { AgentComposition } from '../model/agent.js';

export type ParseResult<T> =
  | { success: true; data: T }
  | { success: false; error: string; validationErrors?: string[] };

export function parseSkillYaml(content: string): ParseResult<SkillBehavior> {
  let parsed: unknown;
  try {
    parsed = parse(content);
  } catch (e) {
    return { success: false, error: `YAML syntax error: ${(e as Error).message}` };
  }

  if (parsed == null || typeof parsed !== 'object') {
    return { success: false, error: 'YAML content is empty or not an object' };
  }

  const validation = validateSkill(parsed);
  if (!validation.valid) {
    return {
      success: false,
      error: 'Schema validation failed',
      validationErrors: validation.errors,
    };
  }

  return { success: true, data: parsed as SkillBehavior };
}

export function parseAgentYaml(content: string): ParseResult<AgentComposition> {
  let parsed: unknown;
  try {
    parsed = parse(content);
  } catch (e) {
    return { success: false, error: `YAML syntax error: ${(e as Error).message}` };
  }

  if (parsed == null || typeof parsed !== 'object') {
    return { success: false, error: 'YAML content is empty or not an object' };
  }

  const validation = validateAgent(parsed);
  if (!validation.valid) {
    return {
      success: false,
      error: 'Schema validation failed',
      validationErrors: validation.errors,
    };
  }

  return { success: true, data: parsed as AgentComposition };
}
