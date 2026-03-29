package frontend

import "embed"

// Dist holds the embedded frontend build.
// When building, place the frontend build output in frontend/dist/.
// For now this is empty — the binary will serve API-only until frontend is added.
//
//go:embed all:dist
var Dist embed.FS
