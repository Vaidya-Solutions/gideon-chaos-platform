# GitHub Automation

This directory owns the future service-local GitHub Actions surface for the
grouped `gideon-chaos-platform` runtime boundary.

## Status

This grouped boundary is in its initial runtime-host state.

What is complete here:

- the grouped root now has a repo-local `.github` documentation surface
- the first runtime child now lives directly under the grouped root
- workflow ownership and boundary expectations are recorded against that
  grouped layout
- future-hosted grouped workflow YAMLs now exist for quality and ARM64
  packaging validation

What is not complete here:

- no hosted grouped repo boundary exists yet

## Current Workflows

- `chaos-platform-quality.yml`
  - runs grouped lint and grouped tests
- `chaos-platform-package.yml`
  - runs grouped ARM64 packaging validation
  - verifies the expected zip outputs exist under `.artifacts/`
  - uploads artifacts only for `workflow_dispatch` and `workflow_call`

## Expected Workflow Direction

When the grouped runtime boundary is ready, this surface should own:

- grouped runtime quality validation
- grouped runtime packaging validation
- grouped runtime artifact publication for explicit manual or reusable flows

Those future workflows should assume:

- the grouped repo root contains the first moved runtime child directly under
  the grouped root
- runtime-local packaging remains independent and service-local
- package output remains repo-local under `.artifacts/`
- `lambda-chaos-machine/.go-version` remains the canonical Go version file for
  the initial grouped runtime host
- `lambda-chaos-machine/go.sum` remains the dependency cache anchor for the
  initial grouped runtime host

## Ownership Boundary

This `.github` surface is intended to be repo-local and runtime-local.

It should own:

- grouped runtime validation workflows
- grouped runtime packaging workflows
- grouped runtime artifact publication rules

It should not own:

- deployment orchestration
- Terraform apply logic
- global governance or policy enforcement outside this grouped runtime boundary

If a change affects infrastructure topology, stack wiring, or broader chaos
governance, the source of truth remains in the Gideon parent meta repo.
