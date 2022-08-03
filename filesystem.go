package main

import (
	"io"
	"io/ioutil"
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
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
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

func ListFilesInDir(dir string) ([]*os.File, error) {
	str, sErr := GetStorage()
	if sErr != nil {
		return nil, sErr
	}
	path := filepath.Join(str, dir)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fs []*os.File
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		p := filepath.Join(path, file.Name())

		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}

	return fs, nil
}

func CloseFiles(fs []*os.File) error {
	for _, f := range fs {
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

// check if directory is empty
func IsEmptyDir(dir string) (bool, error) {
	str, sErr := GetStorage()
	if sErr != nil {
		return false, sErr
	}
	path := filepath.Join(str, dir)

	d, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer d.Close()

	_, err = d.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
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
