import type { SkillBehavior } from '../model/skill-behavior.js';
import { toTitle } from '../utils/to-title.js';

function formatGuardrail(g: string | Record<string, string | number>): string {
  if (typeof g === 'string') return `- ${g}`;
  return Object.entries(g)
    .map(([k, v]) => `- ${k}: ${v}`)
    .join('\n');
}

function buildDescription(skill: SkillBehavior): string {
  const parts: string[] = [];
  parts.push(`${skill.strategy.approach}-based skill`);
  if (skill.context.consumes.length > 0) {
    parts.push(`consuming ${skill.context.consumes.join(', ')}`);
  }
  if (skill.context.produces.length > 0) {
    parts.push(`to produce ${skill.context.produces.join(', ')}`);
  }
  const desc = parts.join(' ');
  // Copilot frontmatter description truncated to 1024 chars max
  if (desc.length > 1024) {
    return desc.slice(0, 1021) + '...';
  }
  return desc;
}

export function generateCopilotSkillMd(skill: SkillBehavior): string {
  const lines: string[] = [];

  // Frontmatter
  const desc = buildDescription(skill);
  lines.push('---');
  lines.push(`name: ${skill.skill}`);
  lines.push(`description: ${desc}`);
  lines.push('---');
  lines.push('');

  // Title
  lines.push(`# ${toTitle(skill.skill)}`);
  lines.push('');

  // Guardrails FIRST (primacy bias)
  if (skill.guardrails.length > 0) {
    lines.push('## Guardrails');
    for (const g of skill.guardrails) {
      lines.push(formatGuardrail(g));
    }
    lines.push('');
  }

  // Context
  lines.push('## Context');
  if (skill.context.consumes.length > 0) {
    lines.push(`Consumes: ${skill.context.consumes.join(', ')}`);
  }
  if (skill.context.produces.length > 0) {
    lines.push(`Produces: ${skill.context.produces.join(', ')}`);
  }
  lines.push(`Memory: ${skill.context.memory}`);
  lines.push('');

  // Dependencies
  if (skill.depends_on.length > 0) {
    lines.push('## Dependencies');
    for (const dep of skill.depends_on) {
      lines.push(`- **${dep.skill}** provides \`${dep.provides}\``);
    }
    lines.push('');
  }

  // Strategy
  lines.push('## Strategy');
  lines.push(`Approach: ${skill.strategy.approach}`);
  if (skill.strategy.tools.length > 0) {
    lines.push(`Tools: ${skill.strategy.tools.join(', ')}`);
  }
  if (skill.strategy.steps && skill.strategy.steps.length > 0) {
    lines.push('');
    lines.push('### Steps');
    skill.strategy.steps.forEach((step, i) => {
      lines.push(`${i + 1}. ${step}`);
    });
  }
  lines.push('');

  // Security LAST (recency bias)
  lines.push('## Security');
  lines.push(`- Filesystem: ${skill.security.filesystem}`);
  lines.push(`- Network: ${skill.security.network}`);
  if (skill.security.secrets.length > 0) {
    lines.push(`- Secrets: ${skill.security.secrets.join(', ')}`);
  }
  if (skill.security.sandbox) {
    lines.push(`- Sandbox: ${skill.security.sandbox}`);
  }
  lines.push('');

  return lines.join('\n');
}
