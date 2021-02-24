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
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/nacl/box"
)

const keystoreDirName = "keystore"

//
const X25519 = "x25519-xsalsa20-poly1305"

// EncryptedData is encrypted blob
type EncryptedData struct {
	Version    string `json:"version"`
	Nonce      string `json:"nonce"`
	PublicKey  string `json:"public_key"`
	Ciphertext string `json:"cipher_text"`
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

// Encrypt plain data
func Encrypt(keystoreDir string, pwd string, fromAddress common.Address,
	toPublicKey []byte, data []byte, version string) (*EncryptedData, error) {
	switch version {
	case X25519:

		fmt.Println("Encrypting data using X25519")
		_toPublicKey := [32]byte{}
		copy(_toPublicKey[:], toPublicKey)

		_fromPublicKey := [32]byte{}
		copy(_fromPublicKey[:], fromAddress.Hash().Bytes()[:])

		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return nil, err
		}
		senderPrivateKey, err := RecoverPrivateKey(keystoreDir, pwd, fromAddress)
		if err != nil {
			return nil, err
		}
		fromPrivateKey := [32]byte{}
		copy(fromPrivateKey[:], senderPrivateKey[:32])
		fmt.Printf("SEALING BOX WITH:\n box: %x\n nonce: %x\n public key: %x\n private key: %x\n",
			data, &nonce, &_toPublicKey, &fromPrivateKey)

		var sharedKey [32]byte
		box.Precompute(&sharedKey, &_toPublicKey, &fromPrivateKey)
		fmt.Println("SHARED KEY")
		fmt.Println(sharedKey)
		out := box.Seal(nil, data, &nonce, &_toPublicKey, &fromPrivateKey)

		return &EncryptedData{
			Version:    version,
			Nonce:      base64.StdEncoding.EncodeToString(nonce[:]),
			PublicKey:  base64.StdEncoding.EncodeToString(_fromPublicKey[:]),
			Ciphertext: base64.StdEncoding.EncodeToString(out),
		}, nil
	default:
		return nil, errors.New("Encryption type/version not supported")
	}
}

// TODO!!!! For some reason, this isn't working cross-client
// i.e. If I am node_1 and I encrypt a message with the /encrypt endpoint for node_2
//		then the message cannot be decrypted, but if node_1 encrypts for itself, then it's fine....
// Decrypt some encrypted data.
func Decrypt(keystoreDir string, address common.Address, pwd string, encryptedData *EncryptedData) ([]byte, error) {
	switch encryptedData.Version {
	case X25519:
		fmt.Println("Decrypting data using X25519")
		// get your own private key
		_toPrivateKey, err := RecoverPrivateKey(keystoreDir, pwd, address)
		if err != nil {
			return []byte{}, err
		}
		toPrivateKey := [32]byte{}
		copy(toPrivateKey[:], _toPrivateKey[:32])
		// assemble decryption parameters
		nonce, _ := base64.StdEncoding.DecodeString(encryptedData.Nonce)
		ciphertext, _ := base64.StdEncoding.DecodeString(encryptedData.Ciphertext)
		// public key of message sender
		_fromPublicKey, _ := base64.StdEncoding.DecodeString(encryptedData.PublicKey)
		fromPublicKey := [32]byte{}
		copy(fromPublicKey[:], _fromPublicKey[:32])

		nonce24 := [24]byte{}
		copy(nonce24[:], nonce[:24])

		var sharedKey [32]byte
		box.Precompute(&sharedKey, &fromPublicKey, &toPrivateKey)
		fmt.Println("SHARED KEY")
		fmt.Println(sharedKey)
		fmt.Printf("OPENING BOX WITH:\n box: %x\n nonce: %x\n public key: %x\n private key: %x\n",
			ciphertext, &nonce24, &fromPublicKey, &toPrivateKey)
		decryptedMessage, ok := box.Open(nil, ciphertext, &nonce24, &fromPublicKey, &toPrivateKey)
		if !ok {
			fmt.Println(ok)
		}
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
