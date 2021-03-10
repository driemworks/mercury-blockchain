package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"ftp2p/manifest"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

const keystoreDirName = "keystore"
const X25519 = "x25519-xsalsa20-poly1305"
const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

// EncryptedData is encrypted blob
type EncryptedData struct {
	Version        string `json:"version"`
	Nonce          string `json:"nonce"`
	EphemPublicKey string `json:"public_key"`
	Ciphertext     string `json:"cipher_text"`
}

type Wallet struct {
	keystore             keystore.KeyStore
	encryptionPublicKey  [32]byte
	encryptionPrivateKey [32]byte
}

func GetKeystoreDirPath(dataDir string) string {
	return filepath.Join(dataDir, keystoreDirName)
}

func NewKeystoreAccount(dataDir, password string) (common.Address, error) {
	ks := keystore.NewKeyStore(GetKeystoreDirPath(dataDir), keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(password)
	if err != nil {
		return common.Address{}, err
	}

	if err := generateEncryptionKeys(dataDir, []byte(password)); err != nil {
		return common.Address{}, err
	}

	return acc.Address, nil
}

func generateEncryptionKeys(datadir string, privateKey []byte) error {
	PublicKey, PrivateKey, _ := box.GenerateKey(rand.Reader)
	_joinedKeys := [64]byte{}
	copy(_joinedKeys[:32], PublicKey[:])
	copy(_joinedKeys[32:], PrivateKey[:])
	// creates a CryptoJSON object
	encryptedEncryptionKeys, err := keystore.EncryptDataV3(
		_joinedKeys[:], privateKey, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return err
	}
	manifest.WriteEncryptionKeys(datadir, encryptedEncryptionKeys)
	return nil
}

func LoadEncryptionKeys(datadir string, password string) ([]byte, error) {
	// load keys.json file
	encryptionKeysJsonFile, err := os.OpenFile(manifest.GetEncryptionKeysFilePath(datadir), os.O_RDONLY, 0600)
	if err != nil {
		return []byte{}, err
	}
	bytes, _ := ioutil.ReadAll(encryptionKeysJsonFile)
	var unmarshalled keystore.CryptoJSON
	json.Unmarshal(bytes, &unmarshalled)
	// decrypt the ciphertext
	decrypted, err := keystore.DecryptDataV3(unmarshalled, password)
	if err != nil {
		return []byte{}, err
	}

	defer encryptionKeysJsonFile.Close()
	return decrypted, nil
}

// GetEncryptionPublicKey returns user's public Encryption key derived from privateKey Ethereum key
func GetEncryptionPublicKey(receiverAddress string) string {
	privateKey0, _ := hexutil.Decode("0x" + receiverAddress)
	privateKey := [32]byte{}
	copy(privateKey[:], privateKey0[:32])

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return base64.StdEncoding.EncodeToString(publicKey[:])
}

// Encrypt plain data
func Encrypt(senderPublicKey, senderPrivateKey, receiverPublicKey [32]byte, data []byte, version string) (*EncryptedData, error) {
	// fmt.Printf("senderPublicKey: %x\n", senderPublicKey)
	// fmt.Printf("senderPrivateKey: %x\n", senderPrivateKey)
	// fmt.Printf("receiverPublicKey: %x\n", receiverPublicKey)
	switch version {
	case X25519:
		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return nil, err
		}

		out := box.Seal(nil, data, &nonce, &receiverPublicKey, &senderPrivateKey)

		return &EncryptedData{
			Version:        version,
			Nonce:          base64.StdEncoding.EncodeToString(nonce[:]),
			EphemPublicKey: base64.StdEncoding.EncodeToString(senderPublicKey[:]),
			Ciphertext:     base64.StdEncoding.EncodeToString(out),
		}, nil
	default:
		return nil, errors.New("Encryption type/version not supported")
	}
}

// Decrypt some encrypted data.
func Decrypt(receiverPrivateKey [32]byte, encryptedData *EncryptedData) ([]byte, error) {
	switch encryptedData.Version {
	case X25519:
		nonce, _ := base64.StdEncoding.DecodeString(encryptedData.Nonce)
		ciphertext, _ := base64.StdEncoding.DecodeString(encryptedData.Ciphertext)
		ephemPublicKey, _ := base64.StdEncoding.DecodeString(encryptedData.EphemPublicKey)

		publicKey := [32]byte{}
		copy(publicKey[:], ephemPublicKey[:32])

		nonce24 := [24]byte{}
		copy(nonce24[:], nonce[:24])
		// fmt.Printf("receiverPrivateKey: %x\n", receiverPrivateKey)
		// fmt.Printf("publicKey: %x\n", publicKey)
		decryptedMessage, ok := box.Open(nil, ciphertext, &nonce24, &publicKey, &receiverPrivateKey)
		if !ok {
			return []byte{}, errors.New("Failed to decrypt the message")
		}
		return decryptedMessage, nil
	default:
		return nil, errors.New("Decryption type/version not supported")
	}
}

func SignTxWithKeystoreAccount(tx manifest.Tx, address common.Address, pwd, keystoreDir string) (manifest.SignedTx, error) {
	keystoreJSON, err := recoverKeystoreJSON(keystoreDir, address)
	if err != nil {
		return manifest.SignedTx{}, err
	}
	key, err := keystore.DecryptKey(keystoreJSON, pwd)
	if err != nil {
		return manifest.SignedTx{}, err
	}

	signedTx, err := SignTx(tx, key.PrivateKey)
	if err != nil {
		return manifest.SignedTx{}, err
	}

	return signedTx, nil
}

func recoverKeystoreJSON(keystoreDir string, address common.Address) ([]byte, error) {
	ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	ksAccount, err := ks.Find(accounts.Account{Address: address})
	if err != nil {
		fmt.Println(err)
		return []byte{}, err
	}
	ksAccountJSON, err := ioutil.ReadFile(ksAccount.URL.Path)
	if err != nil {
		return []byte{}, err
	}
	return ksAccountJSON, nil
}

func SignTx(tx manifest.Tx, privKey *ecdsa.PrivateKey) (manifest.SignedTx, error) {
	rawTx, err := tx.Encode()
	if err != nil {
		return manifest.SignedTx{}, err
	}

	sig, err := Sign(rawTx, privKey)
	if err != nil {
		return manifest.SignedTx{}, err
	}

	return manifest.NewSignedTx(tx, sig), nil
}

func Sign(msg []byte, privKey *ecdsa.PrivateKey) (sig []byte, err error) {
	msgHash := sha256.Sum256(msg)
	return crypto.Sign(msgHash[:], privKey)
}

func Verify(msg, sig []byte) (*ecdsa.PublicKey, error) {
	msgHash := sha256.Sum256(msg)

	recoveredPubKey, err := crypto.SigToPub(msgHash[:], sig)
	if err != nil {
		return nil, fmt.Errorf("unable to verify message signature. %s", err.Error())
	}

	return recoveredPubKey, nil
}

func RecoverPrivateKey(keystoreDir string, pwd string, address common.Address) ([]byte, error) {
	keystoreJSON, err := recoverKeystoreJSON(keystoreDir, address)
	if err != nil {
		return []byte{}, err
	}
	key, err := keystore.DecryptKey(keystoreJSON, pwd)
	if err != nil {
		return []byte{}, err
	}
	receiverPrivateKey := crypto.FromECDSA(key.PrivateKey)
	if err != nil {
		return []byte{}, err
	}
	return receiverPrivateKey, err
}
