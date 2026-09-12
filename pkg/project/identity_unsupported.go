//go:build !linux && !darwin && !windows

package project

// InspectDirectory is the fallback identity resolver for other OS platforms.
func InspectDirectory(path string) (Identity, error) {
	canonical, err := ResolveDirectory(path)
	if err != nil {
		return Identity{}, err
	}
	id, err := readProjectID(canonical)
	if err != nil {
		return Identity{}, err
	}

	return Identity{
		ID:            id,
		CanonicalPath: canonical,
	}, nil
}
