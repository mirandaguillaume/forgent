import { readdirSync, readFileSync, existsSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml, parseAgentYaml } from '../utils/yaml-loader.js';
import { scoreSkill, scoreAgent, type SkillScore, type AgentScore } from '../analyzers/score.js';
import type { SkillBehavior } from '../model/skill-behavior.js';

export interface ScoreReport {
  skills: SkillScore[];
  agents: AgentScore[];
}

export function runScore(skillsDir: string, agentsDir?: string): ScoreReport {
  const skills: SkillBehavior[] = [];
  const skillScores: SkillScore[] = [];

  if (existsSync(skillsDir)) {
    const files = readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'));
    for (const file of files) {
      const content = readFileSync(join(skillsDir, file), 'utf-8');
      const parsed = parseSkillYaml(content);
      if (parsed.success) {
        skills.push(parsed.data);
        skillScores.push(scoreSkill(parsed.data));
      }
    }
  }

  const agentScores: AgentScore[] = [];
  if (agentsDir && existsSync(agentsDir)) {
    const skillMap = new Map(skills.map((s) => [s.skill, s]));
    const files = readdirSync(agentsDir).filter((f) => f.endsWith('.agent.yaml'));
    for (const file of files) {
      const content = readFileSync(join(agentsDir, file), 'utf-8');
      const parsed = parseAgentYaml(content);
      if (parsed.success) {
        const resolved = parsed.data.skills
          .map((name) => skillMap.get(name))
          .filter((s): s is SkillBehavior => s !== undefined);
        agentScores.push(scoreAgent(parsed.data, resolved));
      }
    }
  }

  return { skills: skillScores, agents: agentScores };
}

function scoreColor(score: number): (text: string) => string {
  if (score >= 80) return chalk.green;
  if (score >= 60) return chalk.yellow;
  return chalk.red;
}

function bar(score: number, max: number): string {
  const width = 20;
  const filled = Math.round((score / max) * width);
  const empty = width - filled;
  return chalk.green('█'.repeat(filled)) + chalk.gray('░'.repeat(empty));
}

export function printScoreReport(report: ScoreReport): void {
  if (report.skills.length > 0) {
    console.log(chalk.bold('\n  Skills\n'));
    for (const s of report.skills) {
      const color = scoreColor(s.total);
      console.log(`  ${color(`${s.total}`)}  ${bar(s.total, 100)}  ${s.skill}`);
      console.log(
        chalk.gray(`       context:${s.breakdown.context} strategy:${s.breakdown.strategy} guardrails:${s.breakdown.guardrails} observability:${s.breakdown.observability} security:${s.breakdown.security}`),
      );
    }
  }

  if (report.agents.length > 0) {
    console.log(chalk.bold('\n  Agents\n'));
    for (const a of report.agents) {
      const color = scoreColor(a.total);
      console.log(`  ${color(`${a.total}`)}  ${bar(a.total, 100)}  ${a.agent}`);
      console.log(
        chalk.gray(`       description:${a.breakdown.description} composition:${a.breakdown.composition} dataFlow:${a.breakdown.dataFlow} orchestration:${a.breakdown.orchestration}`),
      );
    }
  }

  if (report.skills.length === 0 && report.agents.length === 0) {
    console.log(chalk.yellow('No skills or agents found to score.'));
  }

  console.log('');
}
