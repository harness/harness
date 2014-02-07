package encrypt

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"io"
)

// EncryptedField handles encrypted and decryption of
// values to and from database columns.
type EncryptedField struct {
	Cipher cipher.Block
}

// PreRead is called before a Scan operation. It is given a pointer to
// the raw struct field, and returns the value that will be given to
// the database driver.
func (e *EncryptedField) PreRead(fieldAddr interface{}) (scanTarget interface{}, err error) {
	// give a pointer to a byte buffer to grab the raw data
	return new([]byte), nil
}

// PostRead is called after a Scan operation. It is given the value returned
// by PreRead and a pointer to the raw struct field. It is expected to fill
// in the struct field if the two are different.
func (e *EncryptedField) PostRead(fieldAddr interface{}, scanTarget interface{}) error {
	ptr := scanTarget.(*[]byte)
	if ptr == nil {
		return fmt.Errorf("encrypter.PostRead: nil pointer")
	}
	raw := *ptr

	// ignore fields that aren't set at all
	if len(raw) == 0 {
		return nil
	}

	// decrypt value for gob decoding
	var err error
	raw, err = decrypt(e.Cipher, raw)
	if err != nil {
		return fmt.Errorf("Gob decryption error: %v", err)
	}

	// decode gob
	gobDecoder := gob.NewDecoder(bytes.NewReader(raw))
	if err := gobDecoder.Decode(fieldAddr); err != nil {
		return fmt.Errorf("Gob decode error: %v", err)
	}

	return nil
}

// PreWrite is called before an Insert or Update operation. It is given
// a pointer to the raw struct field, and returns the value that will be
// given to the database driver.
func (e *EncryptedField) PreWrite(field interface{}) (saveValue interface{}, err error) {
	buffer := new(bytes.Buffer)

	// gob encode
	gobEncoder := gob.NewEncoder(buffer)
	if err := gobEncoder.Encode(field); err != nil {
		return nil, fmt.Errorf("Gob encoding error: %v", err)
	}
	// and then ecrypt
	encrypted, err := encrypt(e.Cipher, buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Gob decryption error: %v", err)
	}

	return encrypted, nil
}

// encrypt is a helper function to encrypt a slice
// of bytes using the specified block cipher.
func encrypt(block cipher.Block, v []byte) ([]byte, error) {
	// if no block cipher value exists we'll assume
	// the database is running in non-ecrypted mode.
	if block == nil {
		return v, nil
	}

	value := make([]byte, len(v))
	copy(value, v)

	// Generate a random initialization vector
	iv := generateRandomKey(block.BlockSize())
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("Could not generate a valid initialization vector for encryption")
	}

	// Encrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)

	// Return iv + ciphertext.
	return append(iv, value...), nil
}

// decrypt is a helper function to decrypt a slice
// using the specified block cipher.
func decrypt(block cipher.Block, value []byte) ([]byte, error) {
	// if no block cipher value exists we'll assume
	// the database is running in non-ecrypted mode.
	if block == nil {
		return value, nil
	}

	size := block.BlockSize()
	if len(value) > size {
		// Extract iv.
		iv := value[:size]
		// Extract ciphertext.
		value = value[size:]
		// Decrypt it.
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)
		return value, nil
	}
	return nil, fmt.Errorf("Could not decrypt the value")
}

// GenerateRandomKey creates a random key of size length bytes
func generateRandomKey(strength int) []byte {
	k := make([]byte, strength)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}
