package config

// InitTest sets Current to the built-in defaults without touching the filesystem.
func InitTest() {
	Current = defaultConfig
}
