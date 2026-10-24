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
		store[string(block)] = 0
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

func encryptionOracle14(prefix []byte, plaintext []byte, key []byte) []byte {
	payload := append(prefix, plaintext...)
	return encryptAESinECB(payload, key, 16)
}

func dictionaryWithPrefix(prefix string, key []byte) map[string]string {
	dict := make(map[string]string)
	for i := range 256 {
		plaintext := append([]byte(prefix), byte(i))
		encrypted := encryptAESinECB(plaintext, key, 16)[:16]
		dict[string(encrypted)] = string(i)
	}
	return dict
}

func challenge14() {

	target := []byte("Byte-at-a-time ECB decryption (Harder)")
	key := generateRandomBytes(16)
	prefix := make([]byte, rand.N(64)+1)
	crand.Read(prefix)

	// detect a full block of A
	fullAencrypted := encryptAESinECB([]byte(strings.Repeat("A", 16)), key, 16)[:16]

	var fullAStartsAt int
	var AsNeeded int
	for i := range 32 {
		attackerBytes := []byte(strings.Repeat("A", i))
		plaintext := append(attackerBytes, target...)
		encrypted := encryptionOracle14(prefix, plaintext, key)
		var found bool
		for j := 0; j < len(encrypted); j += 16 {
			block := encrypted[j:min(j+16, len(encrypted))]
			if bytes.Equal(block, fullAencrypted) {
				fullAStartsAt = j
				AsNeeded = i
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	decrypted := ""

	for i := range 1 {
		dict := dictionaryWithPrefix(strings.Repeat("A", 15-i)+decrypted, key)
		attackerBytes := []byte(strings.Repeat("A", AsNeeded-1))
		plaintext := append(attackerBytes, target...)

		enc := encryptionOracle14(prefix, plaintext, key)
		firstBlock := enc[fullAStartsAt : fullAStartsAt+16]
		decrypted += dict[string(firstBlock)]
		fmt.Println(dict[string(firstBlock)])
	}

}

func main() {
	challenge14()
}
