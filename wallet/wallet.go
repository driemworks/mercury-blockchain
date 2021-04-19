package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/driemworks/mercury-blockchain/state"

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
	return acc.Address, nil
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
func Encrypt(receiverPublicKey [32]byte, data []byte, version string) (*EncryptedData, error) {
	switch version {
	case X25519:
		ephemeralPublic, ephemeralPrivate, _ := box.GenerateKey(rand.Reader)
		publicKey := [32]byte{}
		copy(publicKey[:], receiverPublicKey[:32])

		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return nil, err
		}

		fmt.Println(nonce)
		fmt.Println(ephemeralPublic)
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
func Decrypt(password string, keystoredir string, encryptedData EncryptedData) ([]byte, error) {
	switch encryptedData.Version {
	case X25519:
		// assemble decryption parameters
		nonce, _ := base64.StdEncoding.DecodeString(encryptedData.Nonce)
		ciphertext, _ := base64.StdEncoding.DecodeString(encryptedData.Ciphertext)
		ephemPublicKey, _ := base64.StdEncoding.DecodeString(encryptedData.EphemPublicKey)

		publicKey := [32]byte{}
		copy(publicKey[:], ephemPublicKey[:])

		nonce24 := [24]byte{}
		copy(nonce24[:], nonce[:24])

		privKey, err := RecoverPrivateKey(keystoredir, password, common.Address{})
		if err != nil {
			return nil, errors.New("Failed to recover private key ")
		}
		privateKey := [32]byte{}
		copy(privateKey[:], privKey[:32])

		fmt.Println(nonce24)
		fmt.Println(&publicKey)
		fmt.Println(ciphertext)

		decryptedMessage, ok := box.Open(nil, ciphertext, &nonce24, &publicKey, &privateKey)
		if !ok {
			return nil, errors.New("failed to decrypt the message")
		}
		return decryptedMessage, nil
	default:
		return nil, errors.New("Decryption type/version not supported")
	}
}

func SignTxWithKeystoreAccount(tx state.Tx, address common.Address, pwd, keystoreDir string) (state.SignedTx, error) {
	keystoreJSON, err := recoverKeystoreJSON(keystoreDir, address)
	if err != nil {
		return state.SignedTx{}, err
	}
	key, err := keystore.DecryptKey(keystoreJSON, pwd)
	if err != nil {
		return state.SignedTx{}, err
	}

	signedTx, err := SignTx(tx, key.PrivateKey)
	if err != nil {
		return state.SignedTx{}, err
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

func SignTx(tx state.Tx, privKey *ecdsa.PrivateKey) (state.SignedTx, error) {
	rawTx, err := tx.Encode()
	if err != nil {
		return state.SignedTx{}, err
	}

	sig, err := Sign(rawTx, privKey)
	if err != nil {
		return state.SignedTx{}, err
	}

	return state.NewSignedTx(tx, sig), nil
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
