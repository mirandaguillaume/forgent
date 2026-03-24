---
name: infrastructure-config-audit
description: Production configuration security assessment through policy validation-based skill consuming config_files, infrastructure_code to produce config_audit_report
---

# Infrastructure Config Audit

## Guardrails
- timeout: 300s
- require_production_config_approval: true
- validate_before_deployment: true

## Context
Consumes: config_files, infrastructure_code
Produces: config_audit_report
Memory: short-term

## Strategy
Approach: Production configuration security assessment through policy validation
Tools: read_file, grep, search, bash

### Steps
1. Review production configuration security
2. Validate container and Kubernetes manifests
3. Assess Infrastructure as Code templates
4. Check CI/CD pipeline configurations
5. Verify secrets management setup
6. Validate monitoring configurations

## Security
- Filesystem: read-only
- Network: allowlist
