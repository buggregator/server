//go:build race

package sentry

// The race detector roughly doubles what a test allocates.
const raceEnabled = true
