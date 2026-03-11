import type { AgentComposition } from '../model/agent.js';
import type { SkillBehavior } from '../model/skill-behavior.js';
import { mapToolsToClaude, inferToolsFromSecurity, mergeToolLists } from './claude-tool-map.js';
import { toTitle } from '../utils/to-title.js';

/**
 * Resolves Claude Code tools for an agent based on its skills.
 */
export function resolveAgentTools(skills: SkillBehavior[]): string[] {
  const allTools: string[][] = [];
  for (const skill of skills) {
    allTools.push(mapToolsToClaude(skill.strategy.tools));
    allTools.push(inferToolsFromSecurity(skill.security.filesystem, skill.security.network));
  }
  return mergeToolLists(...allTools);
}

/**
 * Generates a Claude Code agent markdown file.
 *
 * The agent body references skill files in .claude/skills/ instead of embedding them.
 * This keeps agents DRY, composable, and auto-updating when skills change.
 * The outputDir parameter determines the skill file paths used in references.
 */
export function generateAgentMd(agent: AgentComposition, resolvedSkills?: SkillBehavior[], outputDir: string = '.claude'): string {
  const lines: string[] = [];

  // Frontmatter — Claude Code format
  lines.push('---');
  lines.push(`name: ${agent.agent}`);
  if (agent.description) {
    lines.push(`description: ${agent.description}`);
  }

  // Tools — resolve from skills if available, always include Read for skill file access
  if (resolvedSkills && resolvedSkills.length > 0) {
    const tools = resolveAgentTools(resolvedSkills);
    if (!tools.includes('Read')) tools.unshift('Read');
    lines.push(`tools: ${tools.join(', ')}`);
  }

  lines.push('---');
  lines.push('');

  // Body — references skills instead of embedding them
  lines.push(`You are ${toTitle(agent.agent)}. ${agent.description || ''}`);
  lines.push('');

  // Orchestration instructions
  lines.push('## Execution');
  const skillCount = agent.skills.length;

  switch (agent.orchestration) {
    case 'sequential':
      lines.push(`Execute ${skillCount} skills in order. Read each skill file, follow its instructions, then pass the output to the next skill.`);
      break;
    case 'parallel':
      lines.push(`Execute ${skillCount} skills concurrently. Read each skill file and follow its instructions independently.`);
      break;
    case 'parallel-then-merge':
      lines.push(`Execute ${skillCount} skills concurrently, then merge their outputs. Read each skill file and follow its instructions.`);
      break;
    case 'adaptive':
      lines.push(`Choose execution order dynamically. Read each skill file and follow its instructions based on intermediate results.`);
      break;
  }
  lines.push('');

  // Skill references — ordered steps pointing to .claude/skills/<name>/SKILL.md
  for (let i = 0; i < agent.skills.length; i++) {
    const skillName = agent.skills[i];
    const skillPath = `${outputDir}/skills/${skillName}/SKILL.md`;
    const resolved = resolvedSkills?.find((s) => s.skill === skillName);

    lines.push(`### Step ${i + 1}: ${toTitle(skillName)}`);
    lines.push(`Read \`${skillPath}\` and follow its instructions.`);

    // Brief context hint so the agent knows what to expect
    if (resolved) {
      const parts: string[] = [];
      if (resolved.context.consumes.length > 0) {
        parts.push(`Consumes: ${resolved.context.consumes.join(', ')}`);
      }
      if (resolved.context.produces.length > 0) {
        parts.push(`Produces: ${resolved.context.produces.join(', ')}`);
      }
      if (parts.length > 0) {
        lines.push(parts.join(' → '));
      }
    }
    lines.push('');
  }

  // Output format
  if (resolvedSkills && resolvedSkills.length > 0) {
    const allProduced = resolvedSkills.flatMap((s) => s.context.produces);
    const unique = [...new Set(allProduced)];
    if (unique.length > 0) {
      lines.push('## Output');
      lines.push(`Produce a structured report containing: ${unique.join(', ')}.`);
      lines.push('');
    }
  }

  return lines.join('\n');
}
