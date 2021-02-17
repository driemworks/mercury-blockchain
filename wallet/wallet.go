package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"ftp2p/manifest"
	"io"
	"io/ioutil"
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

//
const X25519 = "x25519-xsalsa20-poly1305"

// EncryptedData is encrypted blob
type EncryptedData struct {
	Version        string `json:"version"`
	Nonce          string `json:"nonce"`
	EphemPublicKey string `json:"ephemeral_public_key"`
	Ciphertext     string `json:"cipher_text"`
}

type Wallet struct {
	keystore keystore.KeyStore
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

	return acc.Address, nil
}

// Credit for the following three functions goes to: https://github.com/bakaoh/eip1024/blob/master/eip1024.go

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
func Encrypt(receiverPublicKey string, data []byte, version string) (*EncryptedData, error) {
	switch version {
	case X25519:
		fmt.Println("Using X25519")
		ephemeralPublic, ephemeralPrivate, _ := box.GenerateKey(rand.Reader)
		publicKey0, _ := base64.StdEncoding.DecodeString(receiverPublicKey)
		publicKey := [32]byte{}
		copy(publicKey[:], publicKey0[:32])

		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return nil, err
		}

		out := box.Seal(nil, data, &nonce, &publicKey, ephemeralPrivate)

		return &EncryptedData{
			Version:        version,
			Nonce:          base64.StdEncoding.EncodeToString(nonce[:]),
			EphemPublicKey: base64.StdEncoding.EncodeToString(ephemeralPublic[:]),
			Ciphertext:     base64.StdEncoding.EncodeToString(out),
		}, nil
	default:
		return nil, errors.New("Encryption type/version not supported")
	}
}

// Decrypt some encrypted data.
func Decrypt(keystoreDir string, address common.Address, pwd string, encryptedData *EncryptedData) ([]byte, error) {
	keystoreJSON, err := recoverKeystoreJSON(keystoreDir, address)
	if err != nil {
		return []byte{}, err
	}
	key, err := keystore.DecryptKey(keystoreJSON, pwd)
	if err != nil {
		return []byte{}, err
	}
	receiverPrivateKey := string(crypto.FromECDSA(key.PrivateKey))
	if err != nil {
		return []byte{}, err
	}
	switch encryptedData.Version {
	case X25519:
		fmt.Println("Using X25519")
		privateKey0, _ := hexutil.Decode("0x" + receiverPrivateKey)
		privateKey := [32]byte{}
		copy(privateKey[:], privateKey0[:32])
		// assemble decryption parameters
		nonce, _ := base64.StdEncoding.DecodeString(encryptedData.Nonce)
		ciphertext, _ := base64.StdEncoding.DecodeString(encryptedData.Ciphertext)
		ephemPublicKey, _ := base64.StdEncoding.DecodeString(encryptedData.EphemPublicKey)

		publicKey := [32]byte{}
		copy(publicKey[:], ephemPublicKey[:32])

		nonce24 := [24]byte{}
		copy(nonce24[:], nonce[:24])

		var out []byte
		decryptedMessage, _ := box.Open(out, ciphertext, &nonce24, &publicKey, &privateKey)
		return decryptedMessage, nil
	default:
		return nil, errors.New("Decryption type/version not supported")
	}
}

func SignTxWithKeystoreAccount(tx manifest.Tx, address common.Address, pwd, keystoreDir string) (manifest.SignedTx, error) {
	// ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)

	// ksAccount, err := ks.Find(accounts.Account{Address: acc})
	// if err != nil {
	// 	return manifest.SignedTx{}, err
	// }
	// ksAccountJson, err := ioutil.ReadFile(ksAccount.URL.Path)
	// if err != nil {
	// 	return manifest.SignedTx{}, err
	// }

	// key, err := keystore.DecryptKey(ksAccountJson, pwd)
	// if err != nil {
	// 	return manifest.SignedTx{}, err
	// }

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
