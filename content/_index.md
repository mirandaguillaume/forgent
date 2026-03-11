---
title: Forgent
layout: hextra-home
---

<div class="forge-hero">
  <div class="forge-hero-layout">
    <div class="forge-hero-text">
      <div class="forge-overline">Open-source CLI</div>
      <h1 class="forge-title">Forge your<br><span>agents</span></h1>
      <p class="forge-subtitle">
        Define AI agent skills in YAML. Lint, score, and compile to Claude Code, GitHub Copilot, or any framework.
      </p>
      <div class="forge-actions">
        <a href="docs/getting-started/" class="forge-btn forge-btn-primary">Get Started &rarr;</a>
        <a href="https://github.com/mirandaguillaume/forgent" class="forge-btn forge-btn-ghost">
          <svg viewBox="0 0 16 16"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
          GitHub
        </a>
      </div>
    </div>
    <div class="forge-terminal-wrap">
      <div class="forge-terminal">
        <div class="forge-terminal-bar">
          <span class="forge-terminal-dot"></span>
          <span class="forge-terminal-dot"></span>
          <span class="forge-terminal-dot"></span>
        </div>
        <div class="forge-terminal-body">
          <div class="forge-t-line" style="--d:0.4s"><span class="forge-t-prompt">$ </span><span class="forge-t-cmd">forgent init</span></div>
          <div class="forge-t-line forge-t-out" style="--d:0.8s"><span class="forge-t-ok">&#10003;</span> Created forgent.yaml, skills/, agents/</div>
          <div class="forge-t-line forge-t-gap" style="--d:1.3s"><span class="forge-t-prompt">$ </span><span class="forge-t-cmd">forgent skill create</span> <span class="forge-t-flag">search-web</span></div>
          <div class="forge-t-line forge-t-out" style="--d:1.7s"><span class="forge-t-ok">&#10003;</span> Created skills/search-web.skill.yaml</div>
          <div class="forge-t-line forge-t-gap" style="--d:2.2s"><span class="forge-t-prompt">$ </span><span class="forge-t-cmd">forgent build</span> <span class="forge-t-flag">--target claude</span></div>
          <div class="forge-t-line forge-t-out" style="--d:2.6s">Building for Claude Code...</div>
          <div class="forge-t-line" style="--d:3.0s"><span class="forge-t-ok">&#10003;</span> <span class="forge-t-cmd">Built 3 skills, 1 agent</span> <span class="forge-t-dim">&rarr; .claude/</span></div>
        </div>
      </div>
    </div>
  </div>
</div>

<div class="forge-install">
  <div class="forge-install-box">
    <span class="forge-install-label">Install:</span>
    <code class="forge-install-cmd">go install github.com/mirandaguillaume/forgent/cmd/forgent@latest</code>
  </div>
</div>

<div class="forge-features">
  <div class="forge-features-head">
    <h2>Built for agent engineering</h2>
    <p>Everything you need to design, validate, and ship agent skills.</p>
  </div>
  <div class="forge-grid">
    <div class="forge-cell">
      <h3 data-icon="&#9889;">Skill Behavior Model</h3>
      <p>6 facets per skill &mdash; Context, Strategy, Guardrails, Dependencies, Observability, Security. Structured, composable, auditable.</p>
    </div>
    <div class="forge-cell">
      <h3 data-icon="&#127919;">Multi-Target Build</h3>
      <p>Write YAML specs once. Compile to Claude Code, GitHub Copilot, or add your own target via the TargetGenerator interface.</p>
    </div>
    <div class="forge-cell">
      <h3 data-icon="&#128269;">Design Quality</h3>
      <p>Lint for missing guardrails, score across 5 facets, detect circular dependencies and broken data flows before deploying.</p>
    </div>
    <div class="forge-cell">
      <h3 data-icon="&#128230;">Single Binary</h3>
      <p>Written in Go. Zero dependencies. One binary on Linux, macOS, and Windows.</p>
    </div>
    <div class="forge-cell">
      <h3 data-icon="&#128260;">Watch Mode</h3>
      <p>Run <code>forgent build --watch</code> to auto-rebuild on every file change during development.</p>
    </div>
    <div class="forge-cell">
      <h3 data-icon="&#128295;">Extensible</h3>
      <p>Implement TargetGenerator, register with init(), done. New build targets in under 100 lines of Go.</p>
    </div>
  </div>
</div>

<div class="forge-pipeline">
  <div class="forge-pipeline-inner">
    <h2>One spec, every framework</h2>
    <div class="forge-flow">
      <div class="forge-flow-box">
        <strong>YAML Specs</strong>
        <small>skills/ &middot; agents/</small>
      </div>
      <span class="forge-flow-arrow">&rarr;</span>
      <div class="forge-flow-box forge-flow-accent">
        <strong>forgent build</strong>
        <small>lint &rarr; validate &rarr; gen</small>
      </div>
      <span class="forge-flow-arrow">&rarr;</span>
      <div class="forge-flow-branch">
        <div class="forge-flow-box">
          <strong>.claude/</strong>
          <small>Claude Code</small>
        </div>
        <div class="forge-flow-box">
          <strong>.github/</strong>
          <small>Copilot</small>
        </div>
      </div>
    </div>
  </div>
</div>
