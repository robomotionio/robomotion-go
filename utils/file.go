package utils

import (
	"os"
	"path"
	"runtime"
)

func DirExists(dir string) bool {
	_, err := os.Stat(dir)
	return err == nil
}

func FileExists(dir, fileName string) bool {
	p := path.Join(dir, fileName)
	info, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func TempDir() string {
	var (
		home = UserHomeDir()
		dir  = ""
	)
	if runtime.GOOS == "windows" {
		dir = home + "\\AppData\\Local\\Temp\\Robomotion"
	} else {
		dir = "/tmp/robomotion"
	}
	return dir
}
