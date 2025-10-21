# Windsurf Rules — Todd Parson (Harness SE)

> Place this file at `.windsurf/rules/harness-se-rules.md` (Windsurf Wave 8+), or as a fallback at project root as `.windsurfrules`.
>
> Purpose: encode how I (Todd Parson) prefer to build PoVs and reusable assets across Harness IDP, IACM, CI/CD, and Artifact Registry so Cascade consistently works the way I do.

---

## 1) Communication & Output Style
1. Default to **concise, executive-summary** responses first; put details behind expandable bullets or follow-ups.
2. Use **American English**; keep tone professional, direct, and solution-oriented.
3. When creating artifacts (README, scripts, Terraform, YAML), **include brief preface**: purpose, where it’s used, and assumptions.
4. Prefer **tables or bullet lists** over walls of text when listing requirements, test cases, or “what good looks like”.
5. For slide/brief copy, bias to **headlines with a single takeaway** and 2–4 supporting bullets.

## 2) Project Defaults (assume unless I say otherwise)
1. **Platform:** Harness (CI, CD, IDP, IACM, CCM, STO) is the primary toolchain; integrate rather than replace.
2. **Clouds:** AWS (most common), plus Azure/GCP examples when helpful.
3. **VCS/CI:** GitHub + GitHub Actions for examples; Harness CI for PoV pipelines; use OIDC for auth.
4. **Registry:** Artifactory (trusted); allow proxying/caching of upstream public registries via approved endpoints only.
5. **IaC:** Terraform / OpenTofu modules, versioned with **SemVer**; remote state typically **S3 + DynamoDB** for lock (or use_lockfile when applicable).
6. **Policy:** OPA/Conftest for policy-as-code; treat governance as first-class.

## 3) Repo & Branching Conventions
1. Prefer **monorepo** examples named `<customer>-monorepo` and an admin repo `<customer>-admin`.
2. Use **GitFlow-lite**:
   - feature/* branches → PR → **develop** (integration)
   - **release/** branches (optional) → staging/QA
   - **main** = production
3. **Conventional Commits**; auto-derive next version and generate `CHANGELOG.md` on merge to `main`.
4. Require PR checks: lint, unit tests, `terraform validate/plan`, security scans where relevant.

## 4) Terraform / OpenTofu Standards
1. **Formatting:** Avoid single-line HCL blocks; prefer multi-line with **one attribute per line**.
2. **Headers:** Each module: `README.md` (inputs/outputs), version badge, usage snippet, example.
3. **Providers & Versions:** pin provider versions; add `required_version`.
4. **State:** Use S3 backend. Include safe bucket provisioning with idempotent **empty+delete** helper (handles versioned buckets correctly).
5. **Security:** Least-privilege IAM policies; separate **assumable roles** for CI/CD; include OIDC trust with `aud` scoping.
6. **Quality Gates:** tflint, tfsec (or equivalent), `terraform fmt -check`, `terraform validate`.
7. **Outputs:** Keep minimal; never leak secrets; mark sensitive when applicable.

## 5) Bash / Scripting Standards
1. Shebang and safety: `#!/usr/bin/env bash` + `set -euo pipefail` + `IFS=$'\n\t'` where helpful.
2. Perform **preflight auth checks** for AWS (`aws sts get-caller-identity`) and fail fast with actionable messages.
3. Prefer `getopts` for flags; support `--dry-run`.
4. Use structured logging (emoji or prefixes ✅/❌/ℹ️) and exit codes.
5. Guard destructive ops (S3 delete, IAM detach) with explicit prompts or `--force` flag.

## 6) Docker & Supply Chain
1. Prefer **multi-stage builds**; do not run as root; pin base images; use digest pins for production.
2. Publish to **trusted registry**; optionally enable **provenance/SBOM** generation (keep it lightweight for PoVs).
3. Enforce **base-image and dependency** rules via policy (OPA or registry policy), not ad hoc shell.

## 7) Harness CD / Orchestration Patterns
1. Pipelines must model **promotion across environments** (Dev → QA → Prod) with **gates** (approvals, checks, policy evaluation).
2. Support **staggered/long-lived releases**: introduce a “Release” entity/label that tracks a change across environments with clear visualization.
3. Triggers:
   - From CI: new tag or artifact published → env promotion pipeline.
   - Manual promotion allowed but policy-gated.
4. Surface **evidence** as pipeline output: links to runs, plans, SBOM/provenance, policy results.

## 8) Governance (CI/CD & IDP)
1. Treat governance as **enablement, not blockade**: clear errors/messages and remediation paths.
2. Enforce at least:
   - Only approved registries/base images
   - Required provenance/SBOM (right-sized for PoV)
   - Policy checks at promotion time (OPA/Conftest or Harness Policy)
3. All rules are codified and versioned; no hidden UI-only toggles.

## 9) Artifact Registry Policies
1. **Promotion:** move artifacts to higher trust repos or add immutable labels (e.g., `dev`, `qa`, `prod`) only after passing checks.
2. **Cleanup:** retention by stage and time; never delete artifacts referenced by active deployments; include preview filters.
3. **Proxy/Caching:** enable upstream proxy with allowlist; block direct pulls from public registries in CI/CD unless explicitly allowed.

## 10) Harness IDP Practices
1. Catalog ingestion: prefer **automated discovery** (Harness CD discovery or migration endpoints). For PoVs, demo approach/design without deep integration if sandbox limits exist.
2. Keep **entities** (System/Domain/Component/API/Resource) tidy; include annotations for CI/CD links, observability, and ownership.
3. Provide **developer templates** via IDP; guard with RBAC so platform teams author templates, developers consume them.

## 11) Cookiecutter & Scaffolding
1. The canonical cookiecutter lives inside an **admin repo subdirectory** (e.g., `idp-cookiecutter/`).
2. In pipelines, either:
   - Use a source path parameter to target subdirectory, **or**
   - Shallow clone the repo and set `workingDir` to the template subfolder, then run cookiecutter non-interactively with variables.
3. Templates must include: repo bootstrap, branch protection, CI skeleton, IDP registration artifacts, and IaC starter.

## 12) CI Patterns
1. Feature-branch CI: build + unit tests; optional ephemeral preview.
2. On merge to **develop**: produce integration artifacts; run security scans; publish to staging registry.
3. On release/main: bump version via conventional-commits; publish final artifact; generate/update `CHANGELOG.md`.

## 13) Flux vs Harness (when comparing)
1. Explicitly call out where Flux requires extra “glue”: cross-env orchestration, human approvals, promotion policies, and enterprise RBAC’d templates.
2. Use this to frame **Harness differentiation**; do not denigrate Flux—offer migration/interop patterns when asked.

## 14) Documentation & Evidence
1. Every PoV deliverable includes **What Good Looks Like** and minimal **Evidence** (screenshots/links/CLI output).
2. Prefer repo-local `docs/` with short **Executive Summary** and a **POV Execution Plan** page.

## 15) Safety & Guardrails for Cascade
1. Never push secrets or tokens; mask in logs; use OIDC/IRSA wherever possible.
2. Do not modify files under `app/config` or `infra/shared` unless explicitly asked.
3. Before destructive operations (e.g., `terraform destroy`, bucket deletion), require explicit confirmation or `--force`.
4. When uncertain, propose a plan and ask for confirmation with a succinct diff.

