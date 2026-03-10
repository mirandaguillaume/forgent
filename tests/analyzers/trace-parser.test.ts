import { describe, it, expect } from 'vitest';
import { parseTrace, summarizeTrace, type TraceEvent } from '../../src/analyzers/trace-parser.js';

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

  it('should not warn for tool calls below threshold', () => {
    const events: TraceEvent[] = Array.from({ length: 3 }, (_, i) => ({
      timestamp: i * 100,
      type: 'tool_call' as const,
      skill: 'normal',
      tool: 'read_file',
      duration_ms: 50,
      tokens_in: 10,
      tokens_out: 10,
    }));

    const summary = summarizeTrace(events);
    expect(summary.warnings).toHaveLength(0);
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

  it('should skip empty lines in JSONL', () => {
    const jsonl = [
      '{"timestamp":1000,"type":"tool_call","skill":"s","tool":"t","duration_ms":10,"tokens_in":5,"tokens_out":5}',
      '',
      '{"timestamp":1010,"type":"decision","skill":"s","decision":"done","confidence":1}',
      '',
    ].join('\n');

    const events = parseTrace(jsonl);
    expect(events).toHaveLength(2);
  });

  it('should handle empty trace', () => {
    const summary = summarizeTrace([]);
    expect(summary.totalDuration_ms).toBe(0);
    expect(summary.totalTokens).toBe(0);
    expect(summary.toolCalls).toBe(0);
    expect(summary.decisions).toBe(0);
    expect(summary.warnings).toHaveLength(0);
  });

  it('should track tool frequency per skill', () => {
    const events: TraceEvent[] = [
      { timestamp: 100, type: 'tool_call', skill: 'a', tool: 'read_file', duration_ms: 10, tokens_in: 5, tokens_out: 5 },
      { timestamp: 200, type: 'tool_call', skill: 'b', tool: 'read_file', duration_ms: 10, tokens_in: 5, tokens_out: 5 },
    ];

    const summary = summarizeTrace(events);
    expect(summary.toolFrequency.get('a:read_file')).toBe(1);
    expect(summary.toolFrequency.get('b:read_file')).toBe(1);
  });
});
