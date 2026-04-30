// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

// Package schema provides the embedded chaos-machine input JSON Schema
// for validation. The schema is embedded at compile time via //go:embed.
package schema

import _ "embed"

// InputSchema is the raw JSON Schema for chaos-machine experiment input.
// Embedded from the canonical schema file at build time.
//
//go:embed chaos-machine-input.json
var InputSchema []byte
