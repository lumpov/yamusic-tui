package cache

import (
	"os"
	"path/filepath"

	"github.com/dece2183/yamusic-tui/config"
)

const (
	_FILE_PERM os.FileMode = 0600
	_DIR_PERM  os.FileMode = 0700
)

func getCacheDir() (string, error) {
	var (
		cacheDir string
		err      error
	)

	if len(config.Current.CacheDir) == 0 {
		userDir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(userDir, config.DirName)
	} else {
		cacheDir, err = filepath.Abs(config.Current.CacheDir)
		if err != nil {
			return "", err
		}
	}

	err = os.MkdirAll(cacheDir, _DIR_PERM)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(cacheDir, _DIR_PERM); err != nil {
		return "", err
	}

	return cacheDir, nil
}

func Read(trackId string) (*os.File, int64, error) {
	dir, err := getCacheDir()
	if err != nil {
		return nil, 0, err
	}

	file, err := os.Open(filepath.Join(dir, trackId+".mp3"))
	if err != nil {
		return nil, 0, err
	}

	stat, _ := file.Stat()
	return file, stat.Size(), nil
}

func Write(trackId string) (*os.File, error) {
	dir, err := getCacheDir()
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filepath.Join(dir, trackId+".mp3"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, _FILE_PERM)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(_FILE_PERM); err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}

func Remove(trackId string) error {
	dir, err := getCacheDir()
	if err != nil {
		return err
	}

	return os.Remove(filepath.Join(dir, trackId+".mp3"))
}
