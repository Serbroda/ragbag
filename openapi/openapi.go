// Package openapi carries the API description into the binary: every instance serves the spec of exactly its
// version at /api/openapi.yaml.
package openapi

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
