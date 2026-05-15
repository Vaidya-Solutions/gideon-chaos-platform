# lambda-chaos-machine

Go Lambda functions for the chaos-machine resilience testing workflow
(ADR-128, ADR-133).

## Functions

- `cmd/steady-state`: Verify system health via CloudWatch metrics and alarms
  before the experiment.
- `cmd/start-experiment`: Start the FIS experiment and store the task token in
  DynamoDB.
- `cmd/continue-execution`: Handle the EventBridge callback and resume Step
  Functions via the task token.
- `cmd/evaluate-hypothesis`: Run post-experiment metric evaluation and produce
  SOC 2 evidence.

## Current Boundary Shape

The runtime still packages and deploys as `lambda-chaos-machine`, but the
command-level business logic is now isolated behind internal package seams:

- `cmd/steady-state` delegates to `internal/steadystate`
- `cmd/start-experiment` delegates to `internal/startexperiment`
- `cmd/continue-execution` delegates to `internal/continueexecution`
- `cmd/evaluate-hypothesis` delegates to `internal/evaluatehypothesis`

This is the split-prep phase for the grouped chaos platform: the command shells
stay stable while the ownership boundaries become explicit enough for later
directory-level component extraction.

## Build

```bash
# Root Makefile orchestration (preferred)
make lambda-build LAMBDA_SERVICE=lambda-chaos-machine
make lambda-package-arm64 LAMBDA_SERVICE=lambda-chaos-machine

# Service-local wrapper
make build
make package-arm64
```

Artifacts are written under the grouped repo-local
`.artifacts/lambda-chaos-machine/<arch>/...` tree, including sibling
`bootstrap` and `lambda.zip` outputs.

## Test

```bash
make test
```

## Architecture

- **Runtime**: `provided.al2023` (custom runtime)
- **Architecture**: ARM64 for production, x86_64 for local execution
- **Build tags**: `-tags lambda.norpc` (single-process, no RPC overhead)
- **Schema validation**: `//go:embed` for JSON Schema

See [ADR-133](../../../docs/adr/ADR-133-compiled-lambda-language-policy.md)
for the compiled-language-only policy.
