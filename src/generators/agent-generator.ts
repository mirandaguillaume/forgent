import type { AgentComposition } from '../model/agent.js';

function toTitle(slug: string): string {
  return slug
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function generateAgentMd(agent: AgentComposition): string {
  const lines: string[] = [];

  // Frontmatter
  lines.push('---');
  lines.push(`name: ${agent.agent}`);
  if (agent.description) {
    lines.push(`description: ${agent.description}`);
  }
  lines.push('model: inherit');
  lines.push('---');
  lines.push('');

  // Body
  lines.push(`You are ${toTitle(agent.agent)}, orchestrating skills in ${agent.orchestration} mode.`);
  lines.push('');

  // Skills
  lines.push('## Skills');
  for (const skill of agent.skills) {
    lines.push(`- ${skill}`);
  }
  lines.push('');

  // Orchestration
  lines.push('## Orchestration');
  switch (agent.orchestration) {
    case 'sequential':
      lines.push('Execute skills one after another, passing outputs as inputs to the next.');
      break;
    case 'parallel':
      lines.push('Execute all skills concurrently. Each skill works independently.');
      break;
    case 'parallel-then-merge':
      lines.push('Execute all skills concurrently, then merge their outputs into a unified result.');
      break;
    case 'adaptive':
      lines.push('Choose execution order dynamically based on intermediate results.');
      break;
  }
  lines.push('');

  return lines.join('\n');
}
