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
