//go:build !darwin

package tracker

import "testing"

func TestOtoBackendType(t *testing.T) {
	var _ audioBackend = (*otoBackend)(nil)
	var _ audioPlayer = (*otoPlayer)(nil)
	var _ audioPlayer = (*fakePlayer)(nil)
}
