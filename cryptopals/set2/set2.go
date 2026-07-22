package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
)

func pkcs7pad(s []byte, blocksize int) []byte {
	diff := blocksize - (len(s) % blocksize)
	padding := bytes.Repeat([]byte{byte(diff)}, diff)
	return append(s, padding...)
}

func xor(a, b []byte) []byte {
	c := make([]byte, len(a))
	for i := range a {
		c[i] = a[i] ^ b[i]
	}
	return c
}

func openFile(filepath string) []byte {
	file, _ := os.Open(filepath)
	defer file.Close()
	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Something wrong with file")
	}
	decoded, _ := base64.StdEncoding.DecodeString(sb.String())
	return decoded
}

func encryptAESinECB(plaintext, key []byte, blocksize int) []byte {
	paddedText := pkcs7pad(plaintext, blocksize)
	cipher, _ := aes.NewCipher(key)
	var encrypted []byte
	for block := range slices.Chunk(paddedText, blocksize) {
		buf := make([]byte, blocksize)
		cipher.Encrypt(buf, block)
		encrypted = append(encrypted, buf...)
	}
	// fmt.Println("Encrypted:", string(encrypted))
	return encrypted
}

func decryptAESinECB(ciphertext, key []byte, blocksize int) []byte {
	cipher, _ := aes.NewCipher(key)
	var decrypted []byte
	for block := range slices.Chunk(ciphertext, blocksize) {
		buf := make([]byte, blocksize)
		cipher.Decrypt(buf, block)
		decrypted = append(decrypted, buf...)
	}
	fmt.Println("Decrypted:", string(decrypted))
	return decrypted
}

func encryptAESinCBC(plaintext, key []byte, blocksize int, iv []byte) []byte {
	paddedText := pkcs7pad(plaintext, blocksize)
	cipher, _ := aes.NewCipher(key)
	var encrypted []byte
	prev := iv
	for block := range slices.Chunk(paddedText, blocksize) {
		buf := make([]byte, blocksize)
		cipher.Encrypt(buf, xor(prev, block))
		prev = buf
		encrypted = append(encrypted, buf...)
	}
	// fmt.Println("Encrypted:", string(encrypted))
	return encrypted
}

func decryptAESinCBC(ciphertext, key []byte, blocksize int, iv []byte) []byte {
	cipher, _ := aes.NewCipher(key)
	var decrypted []byte
	prev := iv
	for block := range slices.Chunk(ciphertext, blocksize) {
		buf := make([]byte, blocksize)
		cipher.Decrypt(buf, block)
		decrypted = append(decrypted, xor(buf, prev)...)
		prev = block
	}
	fmt.Println("Decrypted:", string(decrypted))
	return decrypted
}

func generateRandomBytes(size int) []byte {
	key := make([]byte, size)
	crand.Read(key)
	return key
}

func detectAESinECBorCBC(ciphertext []byte) string {
	store := make(map[string]int)
	for block := range slices.Chunk(ciphertext, 16) {
		_, seen := store[string(block)]
		if seen {
			return "ECB"
		}
	}
	return "CBC"
}

func encryptionOracle(plaintext []byte) []byte {
	key := generateRandomBytes(16)
	plaintext = append(generateRandomBytes(5+rand.IntN(5)), plaintext...)
	plaintext = append(plaintext, generateRandomBytes(5+rand.IntN(5))...)
	if rand.IntN(2) == 0 {
		fmt.Println("Used ECB")
		return encryptAESinECB(plaintext, key, 16)
	}
	fmt.Println("Used CBC")
	iv := generateRandomBytes(16)
	return encryptAESinCBC(plaintext, key, 16, iv)
}

func main() {
	plaintext := []byte(`
	Play that funky music Come on, Come on, let me hear
Play that funky music white boy you say it, say it
Play that funky music A little louder now
Play that funky music, white boy Come on, Come on, Come on
Play that funky music`)
	ciphertext := encryptionOracle(plaintext)
	fmt.Println(detectAESinECBorCBC(ciphertext))

}
