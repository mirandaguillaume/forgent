import type { SkillBehavior } from '../model/skill-behavior.js';

export type IssueType = 'circular' | 'missing' | 'unmet-context';

export interface DependencyIssue {
  type: IssueType;
  skill: string;
  message: string;
  details?: string[];
}

export function checkDependencies(skills: SkillBehavior[]): DependencyIssue[] {
  const issues: DependencyIssue[] = [];
  const skillMap = new Map(skills.map((s) => [s.skill, s]));

  // Check missing dependencies
  for (const skill of skills) {
    for (const dep of skill.depends_on) {
      if (!skillMap.has(dep.skill)) {
        issues.push({
          type: 'missing',
          skill: skill.skill,
          message: `Depends on "${dep.skill}" which does not exist`,
        });
      }
    }
  }

  // Check circular dependencies (DFS cycle detection)
  const visited = new Set<string>();
  const inStack = new Set<string>();

  function dfs(name: string, path: string[]): void {
    if (inStack.has(name)) {
      const cycle = [...path.slice(path.indexOf(name)), name];
      issues.push({
        type: 'circular',
        skill: name,
        message: `Circular dependency detected: ${cycle.join(' -> ')}`,
        details: cycle,
      });
      return;
    }
    if (visited.has(name)) return;

    visited.add(name);
    inStack.add(name);

    const skill = skillMap.get(name);
    if (skill) {
      for (const dep of skill.depends_on) {
        if (skillMap.has(dep.skill)) {
          dfs(dep.skill, [...path, name]);
        }
      }
    }

    inStack.delete(name);
  }

  for (const skill of skills) {
    if (!visited.has(skill.skill)) {
      dfs(skill.skill, []);
    }
  }

  // Check unmet context (skill depends on X providing Y, but X doesn't produce Y)
  for (const skill of skills) {
    for (const dep of skill.depends_on) {
      const depSkill = skillMap.get(dep.skill);
      if (depSkill && !depSkill.context.produces.includes(dep.provides)) {
        issues.push({
          type: 'unmet-context',
          skill: skill.skill,
          message: `Expects "${dep.provides}" from "${dep.skill}", but that skill produces: [${depSkill.context.produces.join(', ')}]`,
        });
      }
    }
  }

  return issues;
}
