---
title: Forgent
layout: hextra-home
---

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  Forge agents from composable skill specs
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  A CLI for designing, building, and composing AI agents&nbsp;<br class="sm:hx-block hx-hidden" />across frameworks — Claude Code, GitHub Copilot, and more.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-6">
{{< hextra/hero-badge link="https://github.com/mirandaguillaume/forgent/releases" >}}
  <span>Latest Release</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}
</div>

<div class="hx-mt-6"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Skill Behavior Model"
    subtitle="Define agents as compositions of reusable skills with 6 facets: Context, Strategy, Guardrails, Dependencies, Observability, Security."
    style="background: radial-gradient(ellipse at 50% 80%,rgba(59,130,246,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Multi-Target Build"
    subtitle="Write skills once in YAML, compile to Claude Code, GitHub Copilot, or any framework. One spec, many runtimes."
    style="background: radial-gradient(ellipse at 50% 80%,rgba(142,53,74,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Design Quality Scoring"
    subtitle="Lint, score, and diagnose your agent designs. Catch missing guardrails, broken data flows, and circular dependencies before deployment."
    style="background: radial-gradient(ellipse at 50% 80%,rgba(221,210,59,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Single Binary"
    subtitle="Written in Go. Zero runtime dependencies. Install with go install or download a pre-built binary."
  >}}
  {{< hextra/feature-card
    title="Watch & Rebuild"
    subtitle="forgent build --watch monitors your skills and agents, rebuilding on every change."
  >}}
  {{< hextra/feature-card
    title="Extensible Registry"
    subtitle="Add new build targets by implementing the TargetGenerator interface. Public spec, private implementations — the OCI pattern for agents."
  >}}
{{< /hextra/feature-grid >}}
