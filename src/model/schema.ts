import Ajv from 'ajv';

const ajv = new Ajv({ allErrors: true });

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

const skillSchema = {
  type: 'object',
  required: ['skill', 'version', 'context', 'strategy', 'guardrails', 'depends_on', 'observability', 'security', 'negotiation'],
  properties: {
    skill: { type: 'string', minLength: 1 },
    version: { type: 'string', pattern: '^\\d+\\.\\d+\\.\\d+' },
    context: {
      type: 'object',
      required: ['consumes', 'produces', 'memory'],
      properties: {
        consumes: { type: 'array', items: { type: 'string' } },
        produces: { type: 'array', items: { type: 'string' } },
        memory: { type: 'string', enum: ['short-term', 'conversation', 'long-term'] },
      },
    },
    strategy: {
      type: 'object',
      required: ['tools', 'approach'],
      properties: {
        tools: { type: 'array', items: { type: 'string' } },
        approach: { type: 'string' },
        steps: { type: 'array', items: { type: 'string' } },
      },
    },
    guardrails: {
      type: 'array',
      items: {
        oneOf: [
          { type: 'string' },
          { type: 'object' },
        ],
      },
    },
    depends_on: {
      type: 'array',
      items: {
        type: 'object',
        required: ['skill', 'provides'],
        properties: {
          skill: { type: 'string' },
          provides: { type: 'string' },
        },
      },
    },
    observability: {
      type: 'object',
      required: ['trace_level', 'metrics'],
      properties: {
        trace_level: { type: 'string', enum: ['minimal', 'standard', 'detailed'] },
        metrics: { type: 'array', items: { type: 'string' } },
      },
    },
    security: {
      type: 'object',
      required: ['filesystem', 'network', 'secrets'],
      properties: {
        filesystem: { type: 'string', enum: ['none', 'read-only', 'read-write', 'full'] },
        network: { type: 'string', enum: ['none', 'allowlist', 'full'] },
        secrets: { type: 'array', items: { type: 'string' } },
        sandbox: { type: 'string', enum: ['none', 'container', 'vm'] },
      },
    },
    negotiation: {
      type: 'object',
      required: ['file_conflicts', 'priority'],
      properties: {
        file_conflicts: { type: 'string', enum: ['yield', 'override', 'merge'] },
        priority: { type: 'number', minimum: 0 },
      },
    },
  },
  additionalProperties: false,
};

const agentSchema = {
  type: 'object',
  required: ['agent', 'skills', 'orchestration'],
  properties: {
    agent: { type: 'string', minLength: 1 },
    skills: { type: 'array', items: { type: 'string' }, minItems: 1 },
    orchestration: {
      type: 'string',
      enum: ['sequential', 'parallel', 'parallel-then-merge', 'adaptive'],
    },
    description: { type: 'string' },
  },
  additionalProperties: false,
};

const compiledSkillValidator = ajv.compile(skillSchema);
const compiledAgentValidator = ajv.compile(agentSchema);

function formatErrors(errors: typeof compiledSkillValidator.errors): string[] {
  if (!errors) return [];
  return errors.map((e) => `${e.instancePath || '/'}: ${e.message}`);
}

export function validateSkill(data: unknown): ValidationResult {
  const valid = compiledSkillValidator(data);
  return {
    valid: !!valid,
    errors: formatErrors(compiledSkillValidator.errors),
  };
}

export function validateAgent(data: unknown): ValidationResult {
  const valid = compiledAgentValidator(data);
  return {
    valid: !!valid,
    errors: formatErrors(compiledAgentValidator.errors),
  };
}
