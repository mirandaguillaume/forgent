import { existsSync, writeFileSync } from 'fs';
import { join } from 'path';
import { stringify } from 'yaml';

export interface CreateSkillOptions {
  tools?: string[];
  memory?: string;
  approach?: string;
}

export type CreateSkillResult =
  | { success: true; path: string }
  | { success: false; error: string };

const VALID_NAME = /^[a-z0-9][a-z0-9_-]*$/;

export function createSkill(
  projectDir: string,
  name: string,
  options: CreateSkillOptions = {},
): CreateSkillResult {
  if (!VALID_NAME.test(name)) {
    return { success: false, error: `Invalid skill name "${name}". Use lowercase letters, numbers, hyphens, and underscores only.` };
  }

  const skillPath = join(projectDir, 'skills', `${name}.skill.yaml`);

  if (existsSync(skillPath)) {
    return { success: false, error: `Skill "${name}" already exists at ${skillPath}` };
  }

  const skill = {
    skill: name,
    version: '0.1.0',
    context: {
      consumes: [],
      produces: [],
      memory: options.memory || 'short-term',
    },
    strategy: {
      tools: options.tools || [],
      approach: options.approach || 'sequential',
      steps: [],
    },
    guardrails: [],
    depends_on: [],
    observability: {
      trace_level: 'minimal',
      metrics: [],
    },
    security: {
      filesystem: 'none',
      network: 'none',
      secrets: [],
    },
    negotiation: {
      file_conflicts: 'yield',
      priority: 0,
    },
  };

  writeFileSync(skillPath, stringify(skill));
  return { success: true, path: skillPath };
}
