package domain

// Version is the canonical single source of truth for the MayFly release version.
// It can optionally be overridden at build time via:
// -ldflags="-X mayfly/pkg/domain.Version=..."
var Version = "0.0.5"

