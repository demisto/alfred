package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
)

// Encrypt encrypts the given text with the password using AES and SHA256
func Encrypt(plaintext, pass string) (string, error) {
	key := []byte(pass)
	pt := []byte(plaintext)
	// First, lets hmac the data
	mac := hmac.New(sha256.New, key)
	mac.Write(pt)
	chksum := mac.Sum(nil)
	ciph, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// Now, let's append the checksum and pad
	padding := ciph.BlockSize() - (len(pt)+len(chksum))%ciph.BlockSize()
	// Our plain is message + mac + padding
	plainbytes := make([]byte, len(pt)+len(chksum)+padding)
	copy(plainbytes, pt)
	copy(plainbytes[len(pt):], chksum)
	pad := plainbytes[len(pt)+len(chksum):]
	for i := range pad {
		pad[i] = byte(padding)
	}

	// Our cipher is IV + message + mac + padding
	cipherbytes := make([]byte, ciph.BlockSize()+len(plainbytes))
	iv := cipherbytes[:ciph.BlockSize()]
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(ciph, iv)
	mode.CryptBlocks(cipherbytes[ciph.BlockSize():], plainbytes)
	// Finally, base64 the result
	return base64.StdEncoding.EncodeToString(cipherbytes), nil
}

// Decrypt decrypts the previously encrypted text using pass
func Decrypt(ciphertext, pass string) (string, error) {
	key := []byte(pass)
	cipherbytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	ciph, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// IV is in the beginning
	mode := cipher.NewCBCDecrypter(ciph, cipherbytes[:ciph.BlockSize()])
	plainbytes := make([]byte, len(cipherbytes)-ciph.BlockSize())
	mode.CryptBlocks(plainbytes, cipherbytes[ciph.BlockSize():])
	// After decrypting, let's check padding
	padding := int(plainbytes[len(plainbytes)-1])
	// Let's make sure that checksum is correct
	mac := hmac.New(sha256.New, key)
	cleanbytes := plainbytes[:len(plainbytes)-mac.Size()-padding]
	mac.Write(cleanbytes)
	chksum := mac.Sum(nil)
	if !hmac.Equal(plainbytes[len(cleanbytes):len(cleanbytes)+mac.Size()], chksum) {
		return "", errors.New("Could not validate cleartext")
	}
	return string(cleanbytes), nil
}

// EncryptJSON encrypts the given object by serializing to JSON and using Encrypt
func EncryptJSON(v interface{}, pass string) (string, error) {
	b := new(bytes.Buffer)
	encoder := json.NewEncoder(b)
	err := encoder.Encode(v)
	if err != nil {
		return "", err
	}
	return Encrypt(b.String(), pass)
}

// DecryptJSON decrypts to object
func DecryptJSON(ciphertext, pass string, v interface{}) error {
	plaintext, err := Decrypt(ciphertext, pass)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewBufferString(plaintext))
	return decoder.Decode(v)
}

const (
	alphanum = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`
	alpha    = `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`
)

// SecureRandomString generate a secure random string of size that can be alpha or alphanum
func SecureRandomString(size int, alphaOnly bool) string {
	bytes := make([]byte, size)
	var dict string
	if alphaOnly {
		dict = alpha
	} else {
		dict = alphanum
	}
	for i := range bytes {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(dict))))
		bytes[i] = dict[int(v.Int64())]
	}
	return string(bytes)
}
