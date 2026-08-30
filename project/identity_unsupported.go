//go:build !linux

package project

func filesystemIdentity(string) string { return "" }
