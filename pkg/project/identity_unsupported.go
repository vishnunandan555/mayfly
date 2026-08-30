//go:build !linux && !darwin && !windows

package project

// InspectDirectory is the fallback identity resolver for other OS platforms.
func InspectDirectory(path string) (Identity, error) {
	canonical, err := ResolveDirectory(path)
	if err != nil {
		return Identity{}, err
	}

	return Identity{
		ID:            GenerateID(0, 0, canonical),
		CanonicalPath: canonical,
	}, nil
}
