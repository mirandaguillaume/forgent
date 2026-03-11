---
title: Forgent
layout: hextra-home
---

<div class="forge-hero">
  <div class="forge-hero-inner">
    <div class="forge-hero-text">
      <div class="forge-badge forge-animate">CLI Tool</div>
      <h1 class="forge-title forge-animate forge-delay-1">
        <span class="forge-gradient">Forgent</span>
      </h1>
      <p class="forge-tagline forge-animate forge-delay-2">
        Design, build, and compose AI agents across frameworks — Claude Code, GitHub Copilot, and more. Write once, deploy everywhere.
      </p>
      <div class="forge-ctas forge-animate forge-delay-3">
        <a href="docs/getting-started/" class="forge-btn-primary">Get Started&nbsp;&rarr;</a>
        <a href="https://github.com/mirandaguillaume/forgent" class="forge-btn-secondary">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
          GitHub
        </a>
      </div>
    </div>
    <div class="forge-terminal-wrap forge-animate forge-delay-2">
      <div class="forge-terminal">
        <div class="forge-terminal-header">
          <span class="forge-dot forge-dot-red"></span>
          <span class="forge-dot forge-dot-yellow"></span>
          <span class="forge-dot forge-dot-green"></span>
          <span class="forge-terminal-title">terminal</span>
        </div>
        <div class="forge-terminal-body">
          <div class="forge-line" style="--delay: 0.6s">
            <span class="forge-prompt">$</span>
            <span class="forge-cmd">forgent init</span>
          </div>
          <div class="forge-line forge-output" style="--delay: 1.0s">
            <span class="forge-check">&#10003;</span> Created forgent.yaml, skills/, agents/
          </div>
          <div class="forge-line forge-line-gap" style="--delay: 1.5s">
            <span class="forge-prompt">$</span>
            <span class="forge-cmd">forgent skill create</span>
            <span class="forge-flag"> search-web</span>
          </div>
          <div class="forge-line forge-output" style="--delay: 1.9s">
            <span class="forge-check">&#10003;</span> Created skills/search-web.skill.yaml
          </div>
          <div class="forge-line forge-line-gap" style="--delay: 2.4s">
            <span class="forge-prompt">$</span>
            <span class="forge-cmd">forgent build</span>
            <span class="forge-flag"> --target claude</span>
          </div>
          <div class="forge-line forge-output" style="--delay: 2.8s">
            Building for Claude Code...
          </div>
          <div class="forge-line forge-output-bright" style="--delay: 3.2s">
            <span class="forge-check">&#10003;</span> Built 3 skills, 1 agent &rarr; .claude/
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<div class="forge-section">
  <h2 class="forge-section-title">Everything you need to forge agents</h2>
  <p class="forge-section-sub">
    Define agent behavior once in YAML. Validate, score, and compile to any framework.
  </p>
  <div class="forge-grid">
    <div class="forge-card forge-animate forge-delay-1">
      <span class="forge-card-icon">&#9889;</span>
      <h3>Skill Behavior Model</h3>
      <p>Define agents as compositions of reusable skills with 6 facets: Context, Strategy, Guardrails, Dependencies, Observability, Security.</p>
    </div>
    <div class="forge-card forge-animate forge-delay-2">
      <span class="forge-card-icon">&#127919;</span>
      <h3>Multi-Target Build</h3>
      <p>Write skills once in YAML, compile to Claude Code, GitHub Copilot, or any framework. One spec, many runtimes.</p>
    </div>
    <div class="forge-card forge-animate forge-delay-3">
      <span class="forge-card-icon">&#128269;</span>
      <h3>Design Quality</h3>
      <p>Lint, score, and diagnose your agent designs. Catch missing guardrails, broken data flows, and circular dependencies.</p>
    </div>
    <div class="forge-card forge-animate forge-delay-4">
      <span class="forge-card-icon">&#128230;</span>
      <h3>Single Binary</h3>
      <p>Written in Go. Zero runtime dependencies. Install with <code>go install</code> or download a pre-built binary.</p>
    </div>
    <div class="forge-card forge-animate forge-delay-5">
      <span class="forge-card-icon">&#128260;</span>
      <h3>Watch &amp; Rebuild</h3>
      <p><code>forgent build --watch</code> monitors your skills and agents, rebuilding on every change.</p>
    </div>
    <div class="forge-card forge-animate forge-delay-6">
      <span class="forge-card-icon">&#128295;</span>
      <h3>Extensible Registry</h3>
      <p>Add new build targets by implementing the TargetGenerator interface. Public API, private implementations.</p>
    </div>
  </div>
</div>

<div class="forge-pipeline forge-section">
  <h2 class="forge-section-title">One spec, every framework</h2>
  <p class="forge-section-sub">
    Write YAML specs once. Forgent compiles them to framework-native formats.
  </p>
  <div class="forge-pipeline-flow">
    <div class="forge-pipeline-step">
      <div class="forge-pipeline-step-label">Skills &amp; Agents</div>
      <div class="forge-pipeline-step-detail">skills/*.yaml<br>agents/*.yaml</div>
    </div>
    <span class="forge-pipeline-arrow">&rarr;</span>
    <div class="forge-pipeline-step forge-step-accent">
      <div class="forge-pipeline-step-label">forgent build</div>
      <div class="forge-pipeline-step-detail">lint &rarr; validate &rarr; generate</div>
    </div>
    <span class="forge-pipeline-arrow">&rarr;</span>
    <div class="forge-pipeline-branch">
      <div class="forge-pipeline-step">
        <div class="forge-pipeline-step-label">.claude/</div>
        <div class="forge-pipeline-step-detail">Claude Code</div>
      </div>
      <div class="forge-pipeline-step">
        <div class="forge-pipeline-step-label">.github/</div>
        <div class="forge-pipeline-step-detail">Copilot</div>
      </div>
    </div>
  </div>
</div>
