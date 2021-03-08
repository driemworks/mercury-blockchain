package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"ftp2p/manifest"
	"io"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/box"
)

// The password for testing keystore files:
// 	./node/test_andrej--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57
// 	./node/test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8
const testKeystoreAccountsPwd = "security123"

// Prints a PK:
//
// (*ecdsa.PrivateKey)(0xc000099980)({
// PublicKey: (ecdsa.PublicKey) {
//  Curve: (*secp256k1.BitCurve)(0xc0000982d0)({
//   P: (*big.Int)(0xc0000b03c0)(115792089237316195423570985008687907853269984665640564039457584007908834671663),
//   N: (*big.Int)(0xc0000b0400)(115792089237316195423570985008687907852837564279074904382605163141518161494337),
//   B: (*big.Int)(0xc0000b0440)(7),
//   Gx: (*big.Int)(0xc0000b0480)(55066263022277343669578718895168534326250603453777594175500187360389116729240),
//   Gy: (*big.Int)(0xc0000b04c0)(32670510020758816978083085130507043184471273380659243275938904335757337482424),
//   BitSize: (int) 256
//  }),
//  X: (*big.Int)(0xc0000b1aa0)(1344160861301624411922901086431771879005615956563347131047269353924650464711),
//  Y: (*big.Int)(0xc0000b1ac0)(73524953917715096899857106141372214583670064515671280443711113049610951453654)
// },
// D: (*big.Int)(0xc0000b1a40)(41116516511979929812568468771132209652243963107895293136581156908462828164432)
//})
//
// And r, s, v signature params:
//
// (*big.Int)(0xc0000b1b20)(88181292280759186801869952076472415807575357966745986437065510600744149574656)
// (*big.Int)(0xc0000b1b40)(23476722530623450948411712153618947971604430187320320363672662539909827697049)
// (*big.Int)(0xc0000b1b60)(1)
func TestSignCryptoParams(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(privKey)

	msg := []byte("buffalo buffalo buffalo buffalo buffalo buffalos")

	sig, err := Sign(msg, privKey)
	if err != nil {
		t.Fatal(err)
	}

	if len(sig) != crypto.SignatureLength {
		t.Fatal(fmt.Errorf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}

	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := new(big.Int).SetBytes([]byte{sig[64]})

	spew.Dump(r, s, v)
}

func TestSign(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKey := privKey.PublicKey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:])

	account := common.BytesToAddress(pubKeyBytesHash[12:])

	msg := []byte("the Web3Coach students are awesome")

	sig, err := Sign(msg, privKey)
	if err != nil {
		t.Fatal(err)
	}

	recoveredPubKey, err := Verify(msg, sig)
	if err != nil {
		t.Fatal(err)
	}

	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y)
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:])

	if account.Hex() != recoveredAccount.Hex() {
		t.Fatalf("msg was signed by account %s but signature recovery produced an account %s", account.Hex(), recoveredAccount.Hex())
	}
}

// func TestSignTxWithKeystoreAccount(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir("", "wallet_test")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer manifest.RemoveDir(tmpDir)

// 	andrej, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	babaYaga, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	tx := manifest.NewTx(manifest.NewAddress("test"), manifest.NewAddress("test2"), manifest.NewCID("QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH", ""), 0, 0)

// 	signedTx, err := SignTxWithKeystoreAccount(tx, andrej, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	ok, err := signedTx.IsAuthentic()
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	if !ok {
// 		t.Fatal("the TX was signed by 'from' account and should have been authentic")
// 	}
// }

func TestSignForgedTxWithKeystoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wallet_test")
	if err != nil {
		t.Fatal(err)
	}
	defer manifest.RemoveDir(tmpDir)

	hacker, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Error(err)
		return
	}

	babaYaga, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Error(err)
		return
	}

	forgedTx := manifest.NewTx(babaYaga, hacker, manifest.NewCID("", ""), 1, 1)

	signedTx, err := SignTxWithKeystoreAccount(forgedTx, hacker, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
	if err != nil {
		t.Error(err)
		return
	}

	ok, err := signedTx.IsAuthentic()
	if err != nil {
		t.Error(err)
		return
	}

	if ok {
		t.Fatal("the TX 'from' attribute was forged and should have not be authentic")
	}
}

func TestSealOpen(t *testing.T) {
	publicKey1, privateKey1, _ := box.GenerateKey(rand.Reader)
	publicKey2, privateKey2, _ := box.GenerateKey(rand.Reader)

	if *privateKey1 == *privateKey2 {
		t.Fatalf("private keys are equal!")
	}
	if *publicKey1 == *publicKey2 {
		t.Fatalf("public keys are equal!")
	}
	message := []byte("test message")
	var nonce [24]byte

	out := box.Seal(nil, message, &nonce, publicKey1, privateKey2)
	opened, ok := box.Open(nil, out, &nonce, publicKey2, privateKey1)
	if !ok {
		t.Fatalf("failed to open box")
	}

	if !bytes.Equal(opened, message) {
		t.Fatalf("got %x, want %x", opened, message)
	}

	for i := range out {
		out[i] ^= 0x40
		_, ok := box.Open(nil, out, &nonce, publicKey2, privateKey1)
		if ok {
			t.Fatalf("opened box with byte %d corrupted", i)
		}
		out[i] ^= 0x40
	}
}

type Account struct {
	ethereumPrivateKey   string
	encryptionPrivateKey string
	encryptionPublicKey  string
}

type Message struct {
	data string
}

var bob = Account{
	ethereumPrivateKey:   "7e5374ec2ef0d91761a6e72fdf8f6ac665519bfdf6da0a2329cf0d804514b816",
	encryptionPrivateKey: "flN07C7w2Rdhpucv349qxmVRm/322gojKc8NgEUUuBY=",
	encryptionPublicKey:  "C5YMNdqE4kLgxQhJO1MfuQcHP5hjVSXzamzd/TxlR0U=",
}

var encryptedData = EncryptedData{
	Version:        "x25519-xsalsa20-poly1305",
	Nonce:          "1dvWO7uOnBnO7iNDJ9kO9pTasLuKNlej",
	EphemPublicKey: "FBH1/pAEHOOW14Lu3FWkgV3qOEcuL78Zy+qW1RwzMXQ=",
	Ciphertext:     "f8kBcl/NCyf3sybfbwAKk/np2Bzt9lRVkZejr6uh5FgnNlH/ic62DZzy",
}

var secretMessage = "This is a test message "

func Test_GetEncryptionPublicKey(t *testing.T) {
	result := GetEncryptionPublicKey(bob.ethereumPrivateKey)
	assert.Equal(t, result, bob.encryptionPublicKey)
}

func Test_Encrypt(t *testing.T) {
	var publicKey [32]byte
	copy(publicKey[:], bob.encryptionPublicKey)
	encrypted, err := Encrypt(
		publicKey,
		[]byte(secretMessage),
		"x25519-xsalsa20-poly1305",
	)

	assert.Nil(t, err)
	assert.Equal(t, "x25519-xsalsa20-poly1305", encrypted.Version)
	assert.NotEmpty(t, encrypted.Nonce)
	assert.NotEmpty(t, encrypted.Ciphertext)
	assert.NotEmpty(t, encrypted.EphemPublicKey)

	var privKey [32]byte
	copy(privKey[:], []byte(bob.ethereumPrivateKey))
	decrypted, err := Decrypt(privKey, encrypted)
	assert.Nil(t, err)
	assert.Equal(t, secretMessage, string(decrypted))

}

func Test_Decrypt(t *testing.T) {
	var privKey [32]byte
	copy(privKey[:], []byte(bob.ethereumPrivateKey))
	decrypted, err := Decrypt(privKey, &encryptedData)
	assert.Nil(t, err)
	assert.Equal(t, secretMessage, string(decrypted))
}
func Test_Encrypt_Decrypt_Multi_Node(t *testing.T) {
	message := "Hi this is a message"
	AlicePublicKey, AlicePrivateKey, _ := box.GenerateKey(rand.Reader)
	// assert.Nil(t, AlicePublicKey)
	// fmt.Printf("Alice private %x\n", *AlicePrivateKey)
	// fmt.Printf("Alice public (x-co-ord) %x\n", *AlicePublicKey)
	BobPublicKey, BobPrivateKey, _ := box.GenerateKey(rand.Reader)
	// fmt.Printf("\nBob private %x\n", *BobPrivateKey)
	// fmt.Printf("Bob public (x-co-ord) %x\n", *BobPublicKey)
	var nonce [24]byte
	io.ReadFull(rand.Reader, nonce[:])
	msg := []byte(message)
	encrypted := box.Seal(nonce[:], msg, &nonce, BobPublicKey, AlicePrivateKey)
	var decryptNonce [24]byte
	copy(decryptNonce[:], encrypted[:24])
	decrypted, _ := box.Open(nil, encrypted[24:], &decryptNonce, AlicePublicKey, BobPrivateKey)
	// fmt.Printf("\nMessage: %s\n", message)
	// fmt.Printf("\nEncrypted: %x\n\n", encrypted)
	// fmt.Printf("Decrypted %s", string(decrypted))
	assert.Equal(t, message, string(decrypted))
}
