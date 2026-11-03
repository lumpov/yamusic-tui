//go:build !darwin && !windows && !linux

package idleprevent

func Supported() bool { return false }

func acquire() bool { return false }

func release() {}
