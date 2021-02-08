package manifest

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func getDatabaseDirPath(datadir string) string {
	return filepath.Join(datadir, "manifest")
}

func getGenesisJsonFilePath(datadir string) string {
	return filepath.Join(getDatabaseDirPath(datadir), "genesis.json")
}

func getBlocksDbFilePath(datadir string) string {
	return filepath.Join(getDatabaseDirPath(datadir), "block.db")
}

func initDataDirIfNotExists(dataDir string) error {
	if fileExists(getGenesisJsonFilePath(dataDir)) {
		return nil
	}

	if err := os.MkdirAll(getDatabaseDirPath(dataDir), os.ModePerm); err != nil {
		return err
	}

	if err := writeGenesisToDisk(getGenesisJsonFilePath(dataDir)); err != nil {
		return err
	}

	if err := writeEmptyBlocksDbToDisk(getBlocksDbFilePath(dataDir)); err != nil {
		return err
	}

	return nil
}

func writeEmptyBlocksDbToDisk(path string) error {
	return ioutil.WriteFile(path, []byte(""), os.ModePerm)
}

func fileExists(filepath string) bool {
	if _, err := os.Stat(filepath); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
