# AX CLI — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the `ax` CLI MVP with skill model, creation, and full diagnostic capabilities (doctor, lint, trace).

**Architecture:** TypeScript CLI using Commander.js. Skills are defined in YAML, validated against a JSON Schema. Analyzers and linters operate on parsed skill/agent models. No runtime execution — ax only describes, validates, and diagnoses.

**Tech Stack:** TypeScript 5.x, Node.js 20+, Commander.js, yaml (npm), ajv (JSON Schema), chalk, vitest

**Location:** `tools/ax/`

---

### Task 1: Project Scaffolding

**Files:**
- Create: `tools/ax/package.json`
- Create: `tools/ax/tsconfig.json`
- Create: `tools/ax/vitest.config.ts`
- Create: `tools/ax/src/index.ts`

**Step 1: Create directory structure**

```bash
mkdir -p tools/ax/src/{commands,model,analyzers,linters,utils} tools/ax/templates tools/ax/tests/{model,commands,analyzers,linters}
```

**Step 2: Create package.json**

```json
{
  "name": "ax-cli",
  "version": "0.1.0",
  "description": "Agent Experience CLI — diagnose, lint, and compose AI agent skills",
  "type": "module",
  "bin": {
    "ax": "./dist/index.js"
  },
  "scripts": {
    "build": "tsc",
    "dev": "tsx src/index.ts",
    "test": "vitest run",
    "test:watch": "vitest",
    "lint": "tsc --noEmit"
  },
  "dependencies": {
    "ajv": "^8.17.0",
    "chalk": "^5.3.0",
    "commander": "^12.1.0",
    "yaml": "^2.6.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "tsx": "^4.19.0",
    "typescript": "^5.6.0",
    "vitest": "^2.1.0"
  },
  "engines": {
    "node": ">=20.0.0"
  }
}
```

**Step 3: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "declaration": true,
    "sourceMap": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "tests"]
}
```

**Step 4: Create vitest.config.ts**

```typescript
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    root: '.',
  },
});
```

**Step 5: Create minimal CLI entry point**

```typescript
// src/index.ts
#!/usr/bin/env node
import { Command } from 'commander';

const program = new Command();

program
  .name('ax')
  .description('AX — Agent Experience CLI')
  .version('0.1.0');

program.parse();
```

**Step 6: Install dependencies and verify**

```bash
cd tools/ax && npm install && npx tsx src/index.ts --help
```
Expected: Shows "AX — Agent Experience CLI" with version 0.1.0

**Step 7: Commit**

```bash
git add tools/ax/
git commit -m "feat(ax): scaffold ax-cli project with TypeScript + Commander"
```

---

### Task 2: Skill Behavior Model — TypeScript Types

**Files:**
- Create: `tools/ax/src/model/skill-behavior.ts`
- Create: `tools/ax/src/model/agent.ts`
- Create: `tools/ax/tests/model/skill-behavior.test.ts`

**Step 1: Write the failing test for skill model types**

```typescript
// tests/model/skill-behavior.test.ts
import { describe, it, expect } from 'vitest';
import type { SkillBehavior, ContextFacet, StrategyFacet, GuardrailsFacet, DependencyFacet, ObservabilityFacet, SecurityFacet, NegotiationStrategy } from '../src/model/skill-behavior.js';

describe('SkillBehavior types', () => {
  it('should allow creating a valid skill behavior', () => {
    const skill: SkillBehavior = {
      skill: 'code-review',
      version: '1.2.0',
      context: {
        consumes: ['git_diff', 'file_tree'],
        produces: ['review_comments', 'risk_score'],
        memory: 'conversation',
      },
      strategy: {
        tools: ['read_file', 'grep'],
        approach: 'diff-first',
        steps: ['analyze_diff', 'check_patterns'],
      },
      guardrails: [
        'no_approve_without_tests',
        { max_comments: 10 },
        { timeout: '5min' },
      ],
      depends_on: [
        { skill: 'test-coverage', provides: 'test_results' },
      ],
      observability: {
        trace_level: 'detailed',
        metrics: ['tokens', 'latency', 'decisions'],
      },
      security: {
        filesystem: 'read-only',
        network: 'none',
        secrets: [],
      },
      negotiation: {
        file_conflicts: 'yield',
        priority: 2,
      },
    };

    expect(skill.skill).toBe('code-review');
    expect(skill.version).toBe('1.2.0');
    expect(skill.context.consumes).toContain('git_diff');
    expect(skill.strategy.tools).toHaveLength(2);
    expect(skill.guardrails).toHaveLength(3);
    expect(skill.depends_on).toHaveLength(1);
    expect(skill.observability.trace_level).toBe('detailed');
    expect(skill.security.filesystem).toBe('read-only');
    expect(skill.negotiation.file_conflicts).toBe('yield');
  });

  it('should allow minimal skill with only required fields', () => {
    const skill: SkillBehavior = {
      skill: 'simple-task',
      version: '0.1.0',
      context: {
        consumes: [],
        produces: [],
        memory: 'short-term',
      },
      strategy: {
        tools: [],
        approach: 'sequential',
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

    expect(skill.skill).toBe('simple-task');
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/model/skill-behavior.test.ts
```
Expected: FAIL — cannot resolve module

**Step 3: Implement the skill behavior types**

```typescript
// src/model/skill-behavior.ts

export type MemoryType = 'short-term' | 'conversation' | 'long-term';

export interface ContextFacet {
  consumes: string[];
  produces: string[];
  memory: MemoryType;
}

export interface StrategyFacet {
  tools: string[];
  approach: string;
  steps?: string[];
}

export type GuardrailRule = string | Record<string, string | number>;

export type GuardrailsFacet = GuardrailRule[];

export interface Dependency {
  skill: string;
  provides: string;
}

export type DependencyFacet = Dependency[];

export type TraceLevel = 'minimal' | 'standard' | 'detailed';

export interface ObservabilityFacet {
  trace_level: TraceLevel;
  metrics: string[];
}

export type AccessLevel = 'none' | 'read-only' | 'read-write' | 'full';
export type NetworkAccess = 'none' | 'allowlist' | 'full';

export interface SecurityFacet {
  filesystem: AccessLevel;
  network: NetworkAccess;
  secrets: string[];
  sandbox?: 'none' | 'container' | 'vm';
}

export type NegotiationStrategy = 'yield' | 'override' | 'merge';

export interface NegotiationFacet {
  file_conflicts: NegotiationStrategy;
  priority: number;
}

export interface SkillBehavior {
  skill: string;
  version: string;
  context: ContextFacet;
  strategy: StrategyFacet;
  guardrails: GuardrailsFacet;
  depends_on: DependencyFacet;
  observability: ObservabilityFacet;
  security: SecurityFacet;
  negotiation: NegotiationFacet;
}
```

**Step 4: Run test to verify it passes**

```bash
cd tools/ax && npx vitest run tests/model/skill-behavior.test.ts
```
Expected: PASS

**Step 5: Create agent composition types**

```typescript
// src/model/agent.ts
import type { SkillBehavior } from './skill-behavior.js';

export type OrchestrationStrategy =
  | 'sequential'
  | 'parallel'
  | 'parallel-then-merge'
  | 'adaptive';

export interface AgentComposition {
  agent: string;
  skills: string[];
  orchestration: OrchestrationStrategy;
  description?: string;
}

export interface ResolvedAgent {
  agent: string;
  skills: SkillBehavior[];
  orchestration: OrchestrationStrategy;
}
```

**Step 6: Commit**

```bash
git add tools/ax/src/model/ tools/ax/tests/model/
git commit -m "feat(ax): define Skill Behavior Model types with 6 facets"
```

---

### Task 3: JSON Schema Validation for Skills

**Files:**
- Create: `tools/ax/src/model/schema.ts`
- Create: `tools/ax/tests/model/schema.test.ts`

**Step 1: Write failing test for schema validation**

```typescript
// tests/model/schema.test.ts
import { describe, it, expect } from 'vitest';
import { validateSkill, validateAgent } from '../src/model/schema.js';

describe('Schema validation', () => {
  it('should validate a correct skill', () => {
    const result = validateSkill({
      skill: 'code-review',
      version: '1.0.0',
      context: { consumes: ['git_diff'], produces: ['comments'], memory: 'conversation' },
      strategy: { tools: ['read_file'], approach: 'diff-first' },
      guardrails: [],
      depends_on: [],
      observability: { trace_level: 'minimal', metrics: [] },
      security: { filesystem: 'read-only', network: 'none', secrets: [] },
      negotiation: { file_conflicts: 'yield', priority: 1 },
    });
    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('should reject a skill missing required fields', () => {
    const result = validateSkill({ skill: 'incomplete' });
    expect(result.valid).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
  });

  it('should reject invalid memory type', () => {
    const result = validateSkill({
      skill: 'bad-memory',
      version: '1.0.0',
      context: { consumes: [], produces: [], memory: 'invalid-type' },
      strategy: { tools: [], approach: 'seq' },
      guardrails: [],
      depends_on: [],
      observability: { trace_level: 'minimal', metrics: [] },
      security: { filesystem: 'read-only', network: 'none', secrets: [] },
      negotiation: { file_conflicts: 'yield', priority: 1 },
    });
    expect(result.valid).toBe(false);
  });

  it('should validate a correct agent composition', () => {
    const result = validateAgent({
      agent: 'reviewer',
      skills: ['code-review', 'security-audit'],
      orchestration: 'parallel-then-merge',
    });
    expect(result.valid).toBe(true);
  });

  it('should reject agent with invalid orchestration', () => {
    const result = validateAgent({
      agent: 'bad',
      skills: [],
      orchestration: 'nonexistent',
    });
    expect(result.valid).toBe(false);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/model/schema.test.ts
```
Expected: FAIL

**Step 3: Implement schema validation with ajv**

```typescript
// src/model/schema.ts
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
```

**Step 4: Run test to verify it passes**

```bash
cd tools/ax && npx vitest run tests/model/schema.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add tools/ax/src/model/schema.ts tools/ax/tests/model/schema.test.ts
git commit -m "feat(ax): add JSON Schema validation for skills and agents"
```

---

### Task 4: YAML Parsing Utilities

**Files:**
- Create: `tools/ax/src/utils/yaml-loader.ts`
- Create: `tools/ax/tests/utils/yaml-loader.test.ts`

**Step 1: Write failing test**

```typescript
// tests/utils/yaml-loader.test.ts
import { describe, it, expect } from 'vitest';
import { parseSkillYaml, parseAgentYaml } from '../src/utils/yaml-loader.js';

const validSkillYaml = `
skill: code-review
version: "1.0.0"
context:
  consumes: [git_diff]
  produces: [comments]
  memory: conversation
strategy:
  tools: [read_file]
  approach: diff-first
guardrails: []
depends_on: []
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 1
`;

describe('YAML loader', () => {
  it('should parse a valid skill YAML string', () => {
    const result = parseSkillYaml(validSkillYaml);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.skill).toBe('code-review');
    }
  });

  it('should return error for invalid YAML syntax', () => {
    const result = parseSkillYaml('skill: [invalid: yaml: :::');
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error).toBeDefined();
    }
  });

  it('should return validation errors for schema violations', () => {
    const result = parseSkillYaml('skill: incomplete\nversion: "1.0.0"');
    expect(result.success).toBe(false);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/utils/yaml-loader.test.ts
```
Expected: FAIL

**Step 3: Implement YAML loader**

```typescript
// src/utils/yaml-loader.ts
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
```

**Step 4: Run test to verify it passes**

```bash
cd tools/ax && npx vitest run tests/utils/yaml-loader.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add tools/ax/src/utils/yaml-loader.ts tools/ax/tests/utils/yaml-loader.test.ts
git commit -m "feat(ax): add YAML parsing with schema validation"
```

---

### Task 5: `ax init` Command

**Files:**
- Create: `tools/ax/src/commands/init.ts`
- Create: `tools/ax/templates/skill.yaml`
- Create: `tools/ax/templates/agent.yaml`
- Modify: `tools/ax/src/index.ts`
- Create: `tools/ax/tests/commands/init.test.ts`

**Step 1: Write failing test**

```typescript
// tests/commands/init.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, existsSync, readFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { initProject } from '../src/commands/init.js';

describe('ax init', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'ax-test-'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should create ax.yaml config file', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'ax.yaml'))).toBe(true);
  });

  it('should create skills/ directory', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'skills'))).toBe(true);
  });

  it('should create agents/ directory', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'agents'))).toBe(true);
  });

  it('should create example skill file', () => {
    initProject(tempDir);
    const examplePath = join(tempDir, 'skills', 'example.skill.yaml');
    expect(existsSync(examplePath)).toBe(true);
    const content = readFileSync(examplePath, 'utf-8');
    expect(content).toContain('skill:');
  });

  it('should not overwrite existing ax.yaml', () => {
    const { writeFileSync } = require('fs');
    writeFileSync(join(tempDir, 'ax.yaml'), 'existing: true');
    const result = initProject(tempDir);
    expect(result.alreadyInitialized).toBe(true);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/commands/init.test.ts
```
Expected: FAIL

**Step 3: Create skill template**

```yaml
# templates/skill.yaml
skill: my-skill
version: "0.1.0"

context:
  consumes: []
  produces: []
  memory: short-term

strategy:
  tools: []
  approach: sequential
  steps: []

guardrails: []

depends_on: []

observability:
  trace_level: minimal
  metrics: []

security:
  filesystem: none
  network: none
  secrets: []

negotiation:
  file_conflicts: yield
  priority: 0
```

**Step 4: Create agent template**

```yaml
# templates/agent.yaml
agent: my-agent
skills: []
orchestration: sequential
description: ""
```

**Step 5: Implement init command**

```typescript
// src/commands/init.ts
import { mkdirSync, writeFileSync, existsSync, readFileSync, copyFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

export interface InitResult {
  alreadyInitialized: boolean;
  path: string;
}

export function initProject(targetDir: string): InitResult {
  const configPath = join(targetDir, 'ax.yaml');

  if (existsSync(configPath)) {
    return { alreadyInitialized: true, path: targetDir };
  }

  // Create directories
  mkdirSync(join(targetDir, 'skills'), { recursive: true });
  mkdirSync(join(targetDir, 'agents'), { recursive: true });

  // Write config
  const config = `# AX Project Configuration
version: "0.1.0"
skills_dir: skills
agents_dir: agents
`;
  writeFileSync(configPath, config);

  // Copy example skill
  const templatePath = join(__dirname, '..', '..', 'templates', 'skill.yaml');
  if (existsSync(templatePath)) {
    copyFileSync(templatePath, join(targetDir, 'skills', 'example.skill.yaml'));
  } else {
    // Inline fallback if running from dist
    const fallback = `skill: example
version: "0.1.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: []
  approach: sequential
guardrails: []
depends_on: []
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`;
    writeFileSync(join(targetDir, 'skills', 'example.skill.yaml'), fallback);
  }

  return { alreadyInitialized: false, path: targetDir };
}
```

**Step 6: Wire into CLI**

Add to `src/index.ts`:

```typescript
#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { initProject } from './commands/init.js';

const program = new Command();

program
  .name('ax')
  .description('AX — Agent Experience CLI')
  .version('0.1.0');

program
  .command('init')
  .description('Initialize an AX project in the current directory')
  .argument('[path]', 'target directory', '.')
  .action((path: string) => {
    const result = initProject(path);
    if (result.alreadyInitialized) {
      console.log(chalk.yellow('AX project already initialized.'));
    } else {
      console.log(chalk.green('AX project initialized at'), result.path);
      console.log('  Created: ax.yaml, skills/, agents/');
    }
  });

program.parse();
```

**Step 7: Run test to verify it passes**

```bash
cd tools/ax && npx vitest run tests/commands/init.test.ts
```
Expected: PASS

**Step 8: Commit**

```bash
git add tools/ax/src/commands/init.ts tools/ax/src/index.ts tools/ax/templates/ tools/ax/tests/commands/init.test.ts
git commit -m "feat(ax): add 'ax init' command with project scaffolding"
```

---

### Task 6: `ax skill create` Command

**Files:**
- Create: `tools/ax/src/commands/skill-create.ts`
- Create: `tools/ax/tests/commands/skill-create.test.ts`
- Modify: `tools/ax/src/index.ts`

**Step 1: Write failing test**

```typescript
// tests/commands/skill-create.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, existsSync, readFileSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { createSkill } from '../src/commands/skill-create.js';

describe('ax skill create', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'ax-test-'));
    mkdirSync(join(tempDir, 'skills'));
    writeFileSync(join(tempDir, 'ax.yaml'), 'version: "0.1.0"\nskills_dir: skills');
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should create a skill YAML file', () => {
    const result = createSkill(tempDir, 'code-review');
    expect(result.success).toBe(true);
    expect(existsSync(join(tempDir, 'skills', 'code-review.skill.yaml'))).toBe(true);
  });

  it('should populate skill name in the file', () => {
    createSkill(tempDir, 'code-review');
    const content = readFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), 'utf-8');
    expect(content).toContain('skill: code-review');
  });

  it('should not overwrite existing skill', () => {
    createSkill(tempDir, 'code-review');
    const result = createSkill(tempDir, 'code-review');
    expect(result.success).toBe(false);
    expect(result.error).toContain('already exists');
  });

  it('should accept optional tools', () => {
    createSkill(tempDir, 'searcher', { tools: ['grep', 'find'] });
    const content = readFileSync(join(tempDir, 'skills', 'searcher.skill.yaml'), 'utf-8');
    expect(content).toContain('grep');
    expect(content).toContain('find');
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/commands/skill-create.test.ts
```
Expected: FAIL

**Step 3: Implement skill creation**

```typescript
// src/commands/skill-create.ts
import { existsSync, writeFileSync } from 'fs';
import { join } from 'path';
import { stringify } from 'yaml';

export interface CreateSkillOptions {
  tools?: string[];
  memory?: string;
  approach?: string;
}

export interface CreateSkillResult {
  success: boolean;
  path?: string;
  error?: string;
}

export function createSkill(
  projectDir: string,
  name: string,
  options: CreateSkillOptions = {},
): CreateSkillResult {
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
```

**Step 4: Wire into CLI** — add `skill create` subcommand in `src/index.ts`

**Step 5: Run tests**

```bash
cd tools/ax && npx vitest run tests/commands/skill-create.test.ts
```
Expected: PASS

**Step 6: Commit**

```bash
git add tools/ax/src/commands/skill-create.ts tools/ax/tests/commands/skill-create.test.ts tools/ax/src/index.ts
git commit -m "feat(ax): add 'ax skill create' command"
```

---

### Task 7: Linter Rules Engine

**Files:**
- Create: `tools/ax/src/linters/rules.ts`
- Create: `tools/ax/src/linters/tool-description.ts`
- Create: `tools/ax/src/linters/guardrail-check.ts`
- Create: `tools/ax/src/linters/context-flow.ts`
- Create: `tools/ax/tests/linters/rules.test.ts`

**Step 1: Write failing tests for linter rules**

```typescript
// tests/linters/rules.test.ts
import { describe, it, expect } from 'vitest';
import { lintSkill, type LintResult, type LintSeverity } from '../src/linters/rules.js';
import type { SkillBehavior } from '../src/model/skill-behavior.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'test-skill',
  version: '1.0.0',
  context: { consumes: [], produces: [], memory: 'short-term' },
  strategy: { tools: [], approach: 'sequential' },
  guardrails: [],
  depends_on: [],
  observability: { trace_level: 'minimal', metrics: [] },
  security: { filesystem: 'none', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 0 },
  ...overrides,
});

describe('Linter rules', () => {
  it('should warn when skill has no tools', () => {
    const results = lintSkill(makeSkill());
    expect(results.some((r) => r.rule === 'no-empty-tools')).toBe(true);
  });

  it('should warn when skill has no guardrails', () => {
    const results = lintSkill(makeSkill());
    expect(results.some((r) => r.rule === 'has-guardrails')).toBe(true);
  });

  it('should warn when skill produces data but has no observability metrics', () => {
    const skill = makeSkill({
      context: { consumes: [], produces: ['output'], memory: 'short-term' },
    });
    const results = lintSkill(skill);
    expect(results.some((r) => r.rule === 'observable-outputs')).toBe(true);
  });

  it('should warn when skill has full filesystem access and no guardrails', () => {
    const skill = makeSkill({
      security: { filesystem: 'full', network: 'none', secrets: [] },
    });
    const results = lintSkill(skill);
    expect(results.some((r) => r.rule === 'security-needs-guardrails')).toBe(true);
  });

  it('should pass clean for a well-configured skill', () => {
    const skill = makeSkill({
      strategy: { tools: ['read_file'], approach: 'diff-first' },
      guardrails: ['max_tokens: 1000'],
      observability: { trace_level: 'detailed', metrics: ['tokens'] },
    });
    const results = lintSkill(skill);
    const errors = results.filter((r) => r.severity === 'error');
    expect(errors).toHaveLength(0);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/linters/rules.test.ts
```
Expected: FAIL

**Step 3: Implement linter rules**

```typescript
// src/linters/rules.ts
import type { SkillBehavior } from '../model/skill-behavior.js';

export type LintSeverity = 'error' | 'warning' | 'info';

export interface LintResult {
  rule: string;
  severity: LintSeverity;
  message: string;
  facet: string;
}

type LintRule = (skill: SkillBehavior) => LintResult | null;

const noEmptyTools: LintRule = (skill) => {
  if (skill.strategy.tools.length === 0) {
    return {
      rule: 'no-empty-tools',
      severity: 'warning',
      message: `Skill "${skill.skill}" has no tools defined. An agent without tools has limited capability.`,
      facet: 'strategy',
    };
  }
  return null;
};

const hasGuardrails: LintRule = (skill) => {
  if (skill.guardrails.length === 0) {
    return {
      rule: 'has-guardrails',
      severity: 'warning',
      message: `Skill "${skill.skill}" has no guardrails. Consider adding limits (timeout, max_tokens, etc.).`,
      facet: 'guardrails',
    };
  }
  return null;
};

const observableOutputs: LintRule = (skill) => {
  if (skill.context.produces.length > 0 && skill.observability.metrics.length === 0) {
    return {
      rule: 'observable-outputs',
      severity: 'warning',
      message: `Skill "${skill.skill}" produces data but has no observability metrics. Add metrics to track output quality.`,
      facet: 'observability',
    };
  }
  return null;
};

const securityNeedsGuardrails: LintRule = (skill) => {
  const hasHighAccess = skill.security.filesystem === 'full' || skill.security.filesystem === 'read-write';
  if (hasHighAccess && skill.guardrails.length === 0) {
    return {
      rule: 'security-needs-guardrails',
      severity: 'error',
      message: `Skill "${skill.skill}" has ${skill.security.filesystem} filesystem access but no guardrails. This is dangerous.`,
      facet: 'security',
    };
  }
  return null;
};

const allRules: LintRule[] = [
  noEmptyTools,
  hasGuardrails,
  observableOutputs,
  securityNeedsGuardrails,
];

export function lintSkill(skill: SkillBehavior): LintResult[] {
  return allRules
    .map((rule) => rule(skill))
    .filter((result): result is LintResult => result !== null);
}
```

**Step 4: Run tests**

```bash
cd tools/ax && npx vitest run tests/linters/rules.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add tools/ax/src/linters/ tools/ax/tests/linters/
git commit -m "feat(ax): add AX linter rules engine with 4 rules"
```

---

### Task 8: `ax lint` Command

**Files:**
- Create: `tools/ax/src/commands/lint.ts`
- Modify: `tools/ax/src/index.ts`

**Step 1: Implement lint command**

```typescript
// src/commands/lint.ts
import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml } from '../utils/yaml-loader.js';
import { lintSkill, type LintResult } from '../linters/rules.js';

export interface LintCommandResult {
  totalFiles: number;
  totalIssues: number;
  errors: number;
  warnings: number;
  results: Map<string, LintResult[]>;
}

export function lintDirectory(skillsDir: string): LintCommandResult {
  const files = readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'));
  const results = new Map<string, LintResult[]>();
  let totalIssues = 0;
  let errors = 0;
  let warnings = 0;

  for (const file of files) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);

    if (!parsed.success) {
      results.set(file, [{
        rule: 'valid-schema',
        severity: 'error',
        message: `Invalid skill file: ${parsed.error}`,
        facet: 'schema',
      }]);
      errors++;
      totalIssues++;
      continue;
    }

    const lintResults = lintSkill(parsed.data);
    if (lintResults.length > 0) {
      results.set(file, lintResults);
      for (const r of lintResults) {
        totalIssues++;
        if (r.severity === 'error') errors++;
        if (r.severity === 'warning') warnings++;
      }
    }
  }

  return { totalFiles: files.length, totalIssues, errors, warnings, results };
}

export function printLintResults(result: LintCommandResult): void {
  if (result.totalFiles === 0) {
    console.log(chalk.yellow('No skill files found.'));
    return;
  }

  for (const [file, issues] of result.results) {
    console.log(chalk.bold(`\n${file}`));
    for (const issue of issues) {
      const icon = issue.severity === 'error' ? chalk.red('x') : chalk.yellow('!');
      console.log(`  ${icon} [${issue.facet}] ${issue.message}`);
    }
  }

  console.log(`\nScanned ${result.totalFiles} skills: ${result.errors} errors, ${result.warnings} warnings`);

  if (result.errors > 0) {
    console.log(chalk.red('\nLint failed.'));
  } else if (result.warnings > 0) {
    console.log(chalk.yellow('\nLint passed with warnings.'));
  } else {
    console.log(chalk.green('\nLint passed.'));
  }
}
```

**Step 2: Wire into CLI in `src/index.ts`**

**Step 3: Manual test**

```bash
cd tools/ax && npx tsx src/index.ts lint skills/
```

**Step 4: Commit**

```bash
git add tools/ax/src/commands/lint.ts tools/ax/src/index.ts
git commit -m "feat(ax): add 'ax lint' command for skill quality checks"
```

---

### Task 9: Dependency Analyzer (DAG Validation)

**Files:**
- Create: `tools/ax/src/analyzers/dependency-checker.ts`
- Create: `tools/ax/tests/analyzers/dependency-checker.test.ts`

**Step 1: Write failing test**

```typescript
// tests/analyzers/dependency-checker.test.ts
import { describe, it, expect } from 'vitest';
import { checkDependencies, type DependencyIssue } from '../src/analyzers/dependency-checker.js';
import type { SkillBehavior } from '../src/model/skill-behavior.js';

const makeSkill = (name: string, deps: Array<{ skill: string; provides: string }> = [], consumes: string[] = [], produces: string[] = []): SkillBehavior => ({
  skill: name,
  version: '1.0.0',
  context: { consumes, produces, memory: 'short-term' },
  strategy: { tools: [], approach: 'sequential' },
  guardrails: [],
  depends_on: deps,
  observability: { trace_level: 'minimal', metrics: [] },
  security: { filesystem: 'none', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 0 },
});

describe('Dependency checker', () => {
  it('should detect circular dependencies', () => {
    const skills = [
      makeSkill('a', [{ skill: 'b', provides: 'x' }]),
      makeSkill('b', [{ skill: 'a', provides: 'y' }]),
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'circular')).toBe(true);
  });

  it('should detect missing dependencies', () => {
    const skills = [
      makeSkill('a', [{ skill: 'nonexistent', provides: 'x' }]),
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'missing')).toBe(true);
  });

  it('should detect unmet context requirements', () => {
    const skills = [
      makeSkill('a', [{ skill: 'b', provides: 'test_results' }], ['test_results']),
      makeSkill('b', [], [], ['other_data']), // produces other_data, not test_results
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'unmet-context')).toBe(true);
  });

  it('should pass for valid dependency graph', () => {
    const skills = [
      makeSkill('test-runner', [], [], ['test_results']),
      makeSkill('reviewer', [{ skill: 'test-runner', provides: 'test_results' }], ['test_results']),
    ];
    const issues = checkDependencies(skills);
    expect(issues).toHaveLength(0);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/analyzers/dependency-checker.test.ts
```
Expected: FAIL

**Step 3: Implement dependency checker**

```typescript
// src/analyzers/dependency-checker.ts
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

  function dfs(name: string, path: string[]): boolean {
    if (inStack.has(name)) {
      const cycle = [...path.slice(path.indexOf(name)), name];
      issues.push({
        type: 'circular',
        skill: name,
        message: `Circular dependency detected: ${cycle.join(' -> ')}`,
        details: cycle,
      });
      return true;
    }
    if (visited.has(name)) return false;

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
    return false;
  }

  for (const skill of skills) {
    if (!visited.has(skill.skill)) {
      dfs(skill.skill, []);
    }
  }

  // Check unmet context (skill consumes X, dependency provides Y, but X != Y)
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
```

**Step 4: Run tests**

```bash
cd tools/ax && npx vitest run tests/analyzers/dependency-checker.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add tools/ax/src/analyzers/dependency-checker.ts tools/ax/tests/analyzers/dependency-checker.test.ts
git commit -m "feat(ax): add dependency analyzer with cycle and context validation"
```

---

### Task 10: Loop Detector Analyzer

**Files:**
- Create: `tools/ax/src/analyzers/loop-detector.ts`
- Create: `tools/ax/tests/analyzers/loop-detector.test.ts`

**Step 1: Write failing test**

```typescript
// tests/analyzers/loop-detector.test.ts
import { describe, it, expect } from 'vitest';
import { detectLoopRisks, type LoopRisk } from '../src/analyzers/loop-detector.js';
import type { SkillBehavior } from '../src/model/skill-behavior.js';

const makeSkill = (overrides: Partial<SkillBehavior>): SkillBehavior => ({
  skill: 'test',
  version: '1.0.0',
  context: { consumes: [], produces: [], memory: 'short-term' },
  strategy: { tools: [], approach: 'sequential' },
  guardrails: [],
  depends_on: [],
  observability: { trace_level: 'minimal', metrics: [] },
  security: { filesystem: 'none', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 0 },
  ...overrides,
});

describe('Loop detector', () => {
  it('should flag skill that consumes its own output', () => {
    const skill = makeSkill({
      skill: 'self-loop',
      context: { consumes: ['data'], produces: ['data'], memory: 'short-term' },
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'self-reference')).toBe(true);
  });

  it('should flag skill with no timeout guardrail and long-term memory', () => {
    const skill = makeSkill({
      skill: 'unbounded',
      context: { consumes: [], produces: ['data'], memory: 'long-term' },
      guardrails: [],
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'no-timeout')).toBe(true);
  });

  it('should not flag skill with timeout guardrail', () => {
    const skill = makeSkill({
      skill: 'bounded',
      context: { consumes: [], produces: [], memory: 'long-term' },
      guardrails: [{ timeout: '5min' }],
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'no-timeout')).toBe(false);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/analyzers/loop-detector.test.ts
```
Expected: FAIL

**Step 3: Implement loop detector**

```typescript
// src/analyzers/loop-detector.ts
import type { SkillBehavior } from '../model/skill-behavior.js';

export type LoopRiskType = 'self-reference' | 'no-timeout' | 'unbounded-retry';

export interface LoopRisk {
  type: LoopRiskType;
  skill: string;
  message: string;
  severity: 'warning' | 'error';
}

function hasTimeoutGuardrail(skill: SkillBehavior): boolean {
  return skill.guardrails.some((g) => {
    if (typeof g === 'string') return g.toLowerCase().includes('timeout');
    return 'timeout' in g;
  });
}

export function detectLoopRisks(skill: SkillBehavior): LoopRisk[] {
  const risks: LoopRisk[] = [];

  // Self-reference: consumes and produces the same data
  const overlap = skill.context.consumes.filter((c) =>
    skill.context.produces.includes(c),
  );
  if (overlap.length > 0) {
    risks.push({
      type: 'self-reference',
      skill: skill.skill,
      message: `Skill consumes and produces the same data: [${overlap.join(', ')}]. This can cause infinite loops.`,
      severity: 'error',
    });
  }

  // No timeout with persistent memory
  if (skill.context.memory !== 'short-term' && !hasTimeoutGuardrail(skill)) {
    risks.push({
      type: 'no-timeout',
      skill: skill.skill,
      message: `Skill uses ${skill.context.memory} memory but has no timeout guardrail. Risk of unbounded execution.`,
      severity: 'warning',
    });
  }

  return risks;
}
```

**Step 4: Run tests**

```bash
cd tools/ax && npx vitest run tests/analyzers/loop-detector.test.ts
```
Expected: PASS

**Step 5: Commit**

```bash
git add tools/ax/src/analyzers/loop-detector.ts tools/ax/tests/analyzers/loop-detector.test.ts
git commit -m "feat(ax): add loop risk detector for self-referencing skills"
```

---

### Task 11: `ax doctor` Command

**Files:**
- Create: `tools/ax/src/commands/doctor.ts`
- Modify: `tools/ax/src/index.ts`

**Step 1: Implement doctor command**

The doctor command aggregates all analyzers and linters into a single diagnostic report.

```typescript
// src/commands/doctor.ts
import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml } from '../utils/yaml-loader.js';
import { lintSkill, type LintResult } from '../linters/rules.js';
import { checkDependencies, type DependencyIssue } from '../analyzers/dependency-checker.js';
import { detectLoopRisks, type LoopRisk } from '../analyzers/loop-detector.js';
import type { SkillBehavior } from '../model/skill-behavior.js';

export interface DoctorReport {
  skills: SkillBehavior[];
  parseErrors: Array<{ file: string; error: string }>;
  lintIssues: Map<string, LintResult[]>;
  dependencyIssues: DependencyIssue[];
  loopRisks: Map<string, LoopRisk[]>;
  score: number; // 0-100 health score
}

export function runDoctor(skillsDir: string): DoctorReport {
  const files = readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'));
  const skills: SkillBehavior[] = [];
  const parseErrors: DoctorReport['parseErrors'] = [];
  const lintIssues = new Map<string, LintResult[]>();
  const loopRisks = new Map<string, LoopRisk[]>();

  // Parse all skills
  for (const file of files) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);
    if (parsed.success) {
      skills.push(parsed.data);
    } else {
      parseErrors.push({ file, error: parsed.error });
    }
  }

  // Lint each skill
  for (const skill of skills) {
    const issues = lintSkill(skill);
    if (issues.length > 0) lintIssues.set(skill.skill, issues);
  }

  // Check dependencies across all skills
  const dependencyIssues = checkDependencies(skills);

  // Check loop risks per skill
  for (const skill of skills) {
    const risks = detectLoopRisks(skill);
    if (risks.length > 0) loopRisks.set(skill.skill, risks);
  }

  // Calculate health score
  const totalIssues =
    parseErrors.length +
    [...lintIssues.values()].flat().filter((i) => i.severity === 'error').length +
    dependencyIssues.length +
    [...loopRisks.values()].flat().filter((r) => r.severity === 'error').length;

  const maxScore = Math.max(files.length * 10, 100);
  const score = Math.max(0, Math.round(100 - (totalIssues / maxScore) * 100));

  return { skills, parseErrors, lintIssues, dependencyIssues, loopRisks, score };
}

export function printDoctorReport(report: DoctorReport): void {
  console.log(chalk.bold('\n=== AX Doctor Report ===\n'));
  console.log(`Skills found: ${report.skills.length}`);

  if (report.parseErrors.length > 0) {
    console.log(chalk.red(`\nParse Errors (${report.parseErrors.length}):`));
    for (const err of report.parseErrors) {
      console.log(`  ${chalk.red('x')} ${err.file}: ${err.error}`);
    }
  }

  if (report.lintIssues.size > 0) {
    console.log(chalk.yellow(`\nLint Issues:`));
    for (const [skill, issues] of report.lintIssues) {
      for (const issue of issues) {
        const icon = issue.severity === 'error' ? chalk.red('x') : chalk.yellow('!');
        console.log(`  ${icon} ${skill}: ${issue.message}`);
      }
    }
  }

  if (report.dependencyIssues.length > 0) {
    console.log(chalk.red(`\nDependency Issues (${report.dependencyIssues.length}):`));
    for (const issue of report.dependencyIssues) {
      console.log(`  ${chalk.red('x')} ${issue.skill}: ${issue.message}`);
    }
  }

  if (report.loopRisks.size > 0) {
    console.log(chalk.yellow(`\nLoop Risks:`));
    for (const [skill, risks] of report.loopRisks) {
      for (const risk of risks) {
        const icon = risk.severity === 'error' ? chalk.red('x') : chalk.yellow('!');
        console.log(`  ${icon} ${skill}: ${risk.message}`);
      }
    }
  }

  const scoreColor = report.score >= 80 ? chalk.green : report.score >= 50 ? chalk.yellow : chalk.red;
  console.log(`\nHealth Score: ${scoreColor(`${report.score}/100`)}\n`);
}
```

**Step 2: Wire into CLI**

**Step 3: Commit**

```bash
git add tools/ax/src/commands/doctor.ts tools/ax/src/index.ts
git commit -m "feat(ax): add 'ax doctor' command with aggregated diagnostics"
```

---

### Task 12: `ax trace` Command (Basic)

**Files:**
- Create: `tools/ax/src/commands/trace.ts`
- Create: `tools/ax/src/analyzers/trace-parser.ts`
- Create: `tools/ax/tests/analyzers/trace-parser.test.ts`
- Modify: `tools/ax/src/index.ts`

**Step 1: Write failing test for trace parser**

```typescript
// tests/analyzers/trace-parser.test.ts
import { describe, it, expect } from 'vitest';
import { parseTrace, summarizeTrace, type TraceEvent } from '../src/analyzers/trace-parser.js';

describe('Trace parser', () => {
  it('should parse a sequence of trace events', () => {
    const events: TraceEvent[] = [
      { timestamp: 1000, type: 'tool_call', skill: 'reviewer', tool: 'read_file', duration_ms: 50, tokens_in: 100, tokens_out: 200 },
      { timestamp: 1050, type: 'tool_call', skill: 'reviewer', tool: 'grep', duration_ms: 30, tokens_in: 50, tokens_out: 80 },
      { timestamp: 1080, type: 'decision', skill: 'reviewer', decision: 'approve', confidence: 0.9 },
    ];

    const summary = summarizeTrace(events);
    expect(summary.totalDuration_ms).toBe(80);
    expect(summary.totalTokens).toBe(430);
    expect(summary.toolCalls).toBe(2);
    expect(summary.decisions).toBe(1);
  });

  it('should detect repeated tool calls (potential loop)', () => {
    const events: TraceEvent[] = Array.from({ length: 10 }, (_, i) => ({
      timestamp: i * 100,
      type: 'tool_call' as const,
      skill: 'looper',
      tool: 'read_file',
      duration_ms: 50,
      tokens_in: 100,
      tokens_out: 100,
    }));

    const summary = summarizeTrace(events);
    expect(summary.warnings).toContain('Tool "read_file" called 10 times by "looper" — possible loop');
  });

  it('should parse JSONL trace format', () => {
    const jsonl = [
      '{"timestamp":1000,"type":"tool_call","skill":"s","tool":"t","duration_ms":10,"tokens_in":5,"tokens_out":5}',
      '{"timestamp":1010,"type":"decision","skill":"s","decision":"done","confidence":1}',
    ].join('\n');

    const events = parseTrace(jsonl);
    expect(events).toHaveLength(2);
    expect(events[0].type).toBe('tool_call');
    expect(events[1].type).toBe('decision');
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd tools/ax && npx vitest run tests/analyzers/trace-parser.test.ts
```
Expected: FAIL

**Step 3: Implement trace parser**

```typescript
// src/analyzers/trace-parser.ts

export interface ToolCallEvent {
  timestamp: number;
  type: 'tool_call';
  skill: string;
  tool: string;
  duration_ms: number;
  tokens_in: number;
  tokens_out: number;
}

export interface DecisionEvent {
  timestamp: number;
  type: 'decision';
  skill: string;
  decision: string;
  confidence: number;
}

export type TraceEvent = ToolCallEvent | DecisionEvent;

export interface TraceSummary {
  totalDuration_ms: number;
  totalTokens: number;
  toolCalls: number;
  decisions: number;
  toolFrequency: Map<string, number>;
  warnings: string[];
}

const LOOP_THRESHOLD = 5;

export function parseTrace(jsonl: string): TraceEvent[] {
  return jsonl
    .split('\n')
    .filter((line) => line.trim().length > 0)
    .map((line) => JSON.parse(line) as TraceEvent);
}

export function summarizeTrace(events: TraceEvent[]): TraceSummary {
  let totalDuration_ms = 0;
  let totalTokens = 0;
  let toolCalls = 0;
  let decisions = 0;
  const toolFrequency = new Map<string, { count: number; skill: string }>();
  const warnings: string[] = [];

  for (const event of events) {
    if (event.type === 'tool_call') {
      totalDuration_ms += event.duration_ms;
      totalTokens += event.tokens_in + event.tokens_out;
      toolCalls++;

      const key = `${event.skill}:${event.tool}`;
      const freq = toolFrequency.get(key) || { count: 0, skill: event.skill };
      freq.count++;
      toolFrequency.set(key, freq);
    } else if (event.type === 'decision') {
      decisions++;
    }
  }

  // Detect potential loops
  for (const [key, freq] of toolFrequency) {
    if (freq.count >= LOOP_THRESHOLD) {
      const tool = key.split(':')[1];
      warnings.push(`Tool "${tool}" called ${freq.count} times by "${freq.skill}" — possible loop`);
    }
  }

  return {
    totalDuration_ms,
    totalTokens,
    toolCalls,
    decisions,
    toolFrequency: new Map([...toolFrequency.entries()].map(([k, v]) => [k, v.count])),
    warnings,
  };
}
```

**Step 4: Implement trace command**

```typescript
// src/commands/trace.ts
import { readFileSync } from 'fs';
import chalk from 'chalk';
import { parseTrace, summarizeTrace } from '../analyzers/trace-parser.js';

export function traceFile(tracePath: string): void {
  const content = readFileSync(tracePath, 'utf-8');
  const events = parseTrace(content);
  const summary = summarizeTrace(events);

  console.log(chalk.bold('\n=== AX Trace Summary ===\n'));
  console.log(`Total duration: ${summary.totalDuration_ms}ms`);
  console.log(`Total tokens: ${summary.totalTokens}`);
  console.log(`Tool calls: ${summary.toolCalls}`);
  console.log(`Decisions: ${summary.decisions}`);

  if (summary.toolFrequency.size > 0) {
    console.log(chalk.bold('\nTool usage:'));
    for (const [tool, count] of summary.toolFrequency) {
      console.log(`  ${tool}: ${count}x`);
    }
  }

  if (summary.warnings.length > 0) {
    console.log(chalk.yellow('\nWarnings:'));
    for (const w of summary.warnings) {
      console.log(`  ${chalk.yellow('!')} ${w}`);
    }
  }

  console.log('');
}
```

**Step 5: Wire into CLI**

**Step 6: Run tests**

```bash
cd tools/ax && npx vitest run tests/analyzers/trace-parser.test.ts
```
Expected: PASS

**Step 7: Commit**

```bash
git add tools/ax/src/analyzers/trace-parser.ts tools/ax/src/commands/trace.ts tools/ax/tests/analyzers/trace-parser.test.ts tools/ax/src/index.ts
git commit -m "feat(ax): add 'ax trace' command with JSONL trace analysis"
```

---

### Task 13: Final CLI Integration & Smoke Test

**Files:**
- Modify: `tools/ax/src/index.ts` (final version with all commands)

**Step 1: Verify full CLI with all commands**

```typescript
// Final src/index.ts — ensure all commands are wired
#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { initProject } from './commands/init.js';
import { createSkill } from './commands/skill-create.js';
import { lintDirectory, printLintResults } from './commands/lint.js';
import { runDoctor, printDoctorReport } from './commands/doctor.js';
import { traceFile } from './commands/trace.js';

const program = new Command();

program
  .name('ax')
  .description('AX — Agent Experience CLI')
  .version('0.1.0');

program
  .command('init')
  .description('Initialize an AX project')
  .argument('[path]', 'target directory', '.')
  .action((path: string) => {
    const result = initProject(path);
    if (result.alreadyInitialized) {
      console.log(chalk.yellow('AX project already initialized.'));
    } else {
      console.log(chalk.green('AX project initialized at'), result.path);
    }
  });

const skill = program.command('skill').description('Manage skills');

skill
  .command('create')
  .description('Create a new skill')
  .argument('<name>', 'skill name')
  .option('-t, --tools <tools...>', 'tools the skill can use')
  .option('-m, --memory <type>', 'memory type', 'short-term')
  .action((name: string, opts: { tools?: string[]; memory?: string }) => {
    const result = createSkill('.', name, opts);
    if (result.success) {
      console.log(chalk.green(`Skill "${name}" created at ${result.path}`));
    } else {
      console.log(chalk.red(result.error));
    }
  });

program
  .command('lint')
  .description('Lint skill files for AX best practices')
  .argument('[path]', 'skills directory', 'skills')
  .action((path: string) => {
    const result = lintDirectory(path);
    printLintResults(result);
    process.exit(result.errors > 0 ? 1 : 0);
  });

program
  .command('doctor')
  .description('Run full diagnostic on all skills')
  .argument('[path]', 'skills directory', 'skills')
  .action((path: string) => {
    const report = runDoctor(path);
    printDoctorReport(report);
    process.exit(report.score < 50 ? 1 : 0);
  });

program
  .command('trace')
  .description('Analyze a JSONL trace file')
  .argument('<file>', 'trace file path (JSONL format)')
  .action((file: string) => {
    traceFile(file);
  });

program.parse();
```

**Step 2: Run all tests**

```bash
cd tools/ax && npx vitest run
```
Expected: ALL PASS

**Step 3: Smoke test CLI**

```bash
cd /tmp && mkdir ax-test && cd ax-test
npx tsx /path/to/tools/ax/src/index.ts init
npx tsx /path/to/tools/ax/src/index.ts skill create code-review --tools read_file grep
npx tsx /path/to/tools/ax/src/index.ts lint skills/
npx tsx /path/to/tools/ax/src/index.ts doctor skills/
```

**Step 4: Commit**

```bash
git add tools/ax/src/index.ts
git commit -m "feat(ax): wire all commands into CLI entry point"
```

---

## Summary

| Task | Component | Tests | Commits |
|------|-----------|-------|---------|
| 1 | Project scaffolding | - | 1 |
| 2 | Skill Behavior Model types | 2 | 1 |
| 3 | JSON Schema validation | 5 | 1 |
| 4 | YAML parsing | 3 | 1 |
| 5 | `ax init` command | 5 | 1 |
| 6 | `ax skill create` command | 4 | 1 |
| 7 | Linter rules engine | 5 | 1 |
| 8 | `ax lint` command | - | 1 |
| 9 | Dependency analyzer | 4 | 1 |
| 10 | Loop detector | 3 | 1 |
| 11 | `ax doctor` command | - | 1 |
| 12 | `ax trace` command | 3 | 1 |
| 13 | Final integration | smoke | 1 |
| **Total** | **13 tasks** | **34 tests** | **13 commits** |
