package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

//

func GetKeystoreDirPath(datadir string) string {
	return filepath.Join(datadir, "keystore")
}

func getDatabaseDirPath(datadir string) string {
	return filepath.Join(datadir, "manifest")
}

func getGenesisJsonFilePath(datadir string) string {
	return filepath.Join(getDatabaseDirPath(datadir), "genesis.json")
}

func getBlocksDbFilePath(datadir string, isTemp bool) string {
	if isTemp {
		return filepath.Join(getDatabaseDirPath(datadir), "block.db.tmp")
	}
	return filepath.Join(getDatabaseDirPath(datadir), "block.db")
}

func GetEncryptionKeysFilePath(datadir string) string {
	return filepath.Join(GetKeystoreDirPath(datadir), "keys.json")
}

func initDataDirIfNotExists(dataDir string) error {
	// if the genesis.json file dne, create it
	if !fileExists(getGenesisJsonFilePath(dataDir)) {
		// create root directory for our db (create parents dirs if needed)
		// ensure root dir exists
		if !fileExists(getDatabaseDirPath(dataDir)) {
			if err := os.MkdirAll(getDatabaseDirPath(dataDir), os.ModePerm); err != nil {
				return err
			}
		}
		// write genesis.json
		if err := writeGenesisToDisk(getGenesisJsonFilePath(dataDir)); err != nil {
			return err
		}
	}

	if !fileExists(getBlocksDbFilePath(dataDir, false)) {
		if !fileExists(getDatabaseDirPath(dataDir)) {
			if err := os.MkdirAll(getDatabaseDirPath(dataDir), os.ModePerm); err != nil {
				return err
			}
		}
		// write empty block.db
		if err := writeEmptyFileToDisk(getBlocksDbFilePath(dataDir, false)); err != nil {
			return err
		}
	}
	return nil
}

func WriteEncryptionKeys(datadir string, key keystore.CryptoJSON) error {
	if !fileExists(GetEncryptionKeysFilePath(datadir)) {
		_json, err := json.Marshal(key)
		if err != nil {
			return err
		}
		ioutil.WriteFile(GetEncryptionKeysFilePath(datadir), _json, os.ModePerm)
	}
	return nil
}

func writeEmptyFileToDisk(path string) error {
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

func ExpandPath(p string) string {
	if i := strings.Index(p, ":"); i > 0 {
		return p
	}
	if i := strings.Index(p, "@"); i > 0 {
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func Unicode(s string) string {
	r, _ := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)

	return fmt.Sprint(r)
}

func Rename(originalPath string, destinationPath string) error {
	return os.Rename(originalPath, destinationPath)
}

func RemoveDir(path string) error {
	return os.RemoveAll(path)
}

func writeGenesisToDisk(path string) error {
	return ioutil.WriteFile(path, []byte(genesisJson), 0644)
}
