package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CreateFile(p string, data []byte) (*os.File, error) {
	str, sErr := GetStorage()
	if sErr != nil {
		return nil, sErr
	}
	path := filepath.Join(str, p)
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return nil, err
	}
	dst, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(dst, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return dst, nil
}

func GetFile(p string) (*os.File, error) {
	str, sErr := GetStorage()
	if sErr != nil {
		return nil, sErr
	}
	path := filepath.Join(str, p)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func CreateDir(dir string) (string, error) {
	str, sErr := GetStorage()
	if sErr != nil {
		return "", sErr
	}
	path := filepath.Join(str, dir)
	if err := os.MkdirAll(path, 0777); err != nil {
		return "", err
	}
	return path, nil
}

func DeleteDir(dir string, force bool) error {
	str, sErr := GetStorage()
	if sErr != nil {
		return sErr
	}
	path := filepath.Join(str, dir)
	if force {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	} else {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

//Delete file giving the path as p
func DeleteFile(p string) error {
	str, sErr := GetStorage()
	if sErr != nil {
		return sErr
	}
	path := filepath.Join(str, p)
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

func Exists(p string) (bool, error) {
	str, sErr := GetStorage()
	if sErr != nil {
		return false, sErr
	}
	path := filepath.Join(str, p)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetStorage() (string, error) {
	if err := os.MkdirAll("cloud", os.ModeDir); err != nil {
		return "", err
	}
	return "./cloud", nil
}

func NameWithoutExt(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
