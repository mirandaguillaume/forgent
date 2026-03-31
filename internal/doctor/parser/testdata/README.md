# Agent File Test Corpus

Real-world agent definition files scraped from public GitHub repositories for testing Forgent's doctor analysis pipeline.

## Claude Code Agent Files (12 files)

| File | Source Repository | Path |
|---|---|---|
| zen-qa.md | github.com/sheshbabu/zen | .claude/agents/qa.md |
| ai-agent-team-tech-leader.md | github.com/peterfei/ai-agent-team | .claude/agents/tech-leader.md |
| ooda-decide.md | github.com/al3rez/ooda-subagents | .claude/agents/decide.md |
| shadcn-tester.md | github.com/binjuhor/shadcn-lar | .claude/agents/tester.md |
| contributor-info-meta-agent.md | github.com/bdougie/contributor.info | .claude/agents/meta-agent.md |
| vertz-ava.md | github.com/vertz-dev/vertz | .claude/agents/ava.md |
| sdd-planner.md | github.com/yi-john-huang/sdd-mcp | .claude/agents/planner.md |
| dotfiles-debugger.md | github.com/Subjective/dotfiles | .claude/agents/debugger.md |
| ignixa-fhir-ef.md | github.com/brendankowitz/ignixa-fhir | .claude/agents/ef.md |
| debug-toolbar-prd.md | github.com/JacobCoffee/debug-toolbar | .claude/agents/prd.md |
| simple-rpc-security.md | github.com/AWolf81/simple-rpc-ai-backend | .claude/agents/security.md |
| agentctl-researcher.md | github.com/ryantking/agentctl | .claude/agents/researcher.md |

## GitHub Copilot Agent Files (10 files)

| File | Source Repository | Path |
|---|---|---|
| maui-sandbox-agent.md | github.com/dotnet/maui | .github/agents/sandbox-agent.md |
| azure-sdk-dash.md | github.com/Azure/azure-sdk-for-js | .github/agents/dash.agent.md |
| camunda-ci.md | github.com/camunda/camunda | .github/agents/ci.agent.md |
| calva-issue.md | github.com/BetterThanTomorrow/calva | .github/agents/issue.agent.md |
| openxml-readme.md | github.com/dotnet/Open-XML-SDK | .github/agents/readme.agent.md |
| versatile-thermostat-python.md | github.com/jmcollin78/versatile_thermostat | .github/agents/python.agent.md |
| polyglot-docs.md | github.com/untra/polyglot | .github/agents/docs.agent.md |
| nunit-agent.md | github.com/nunit/nunit | .github/agents/NUnit.agent.md |
| copilot-rules-spec.md | github.com/NikiforovAll/github-copilot-rules | .github/agents/spec.agent.md |
| zigbee-specification.md | github.com/Nerivec/zigbee2mqtt-windfront | .github/agents/specification.agent.md |

## Collection Details

- Collected: 2026-03-30
- Method: GitHub Code Search API via `gh search code`
- Criteria: Public repositories with agent definition files
- License: All from public repositories (verified accessible)
- Purpose: Engineering spike — validate local LLM analysis quality
