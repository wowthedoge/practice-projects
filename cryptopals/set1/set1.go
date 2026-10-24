package main

import (
	"bufio"
	"cmp"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"math/bits"
	"os"
	"slices"
	"strings"
)

func HexToBase64(hexString string) string {
	bytes, _ := hex.DecodeString(hexString)
	return base64.StdEncoding.EncodeToString(bytes)
}

func XOR(hexString string, xor string) string {
	a, _ := hex.DecodeString(hexString)
	b, _ := hex.DecodeString(xor)

	result := make([]byte, len(a))
	for i := range a {
		result[i] = a[i] ^ b[i]
	}
	return hex.EncodeToString(result)
}

func openFile(filepath string) string {
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
	return sb.String()
}

// func solveSingleByteXOR(hexString string) string {
// 	type Result struct {
// 		key   int
// 		text  string
// 		score int
// 	}
// 	var decoded []Result

// 	for i := range 256 {
// 		byteHexString, _ := hex.DecodeString(hexString)
// 		xorResult := make([]byte, len(byteHexString))
// 		for j, b := range byteHexString {
// 			xorResult[j] = b ^ byte(i)
// 		}
// 		decodedText := string(xorResult)

// 		decoded = append(decoded, Result{key: i, text: decodedText, score: score(decodedText)})
// 	}

// 	slices.SortFunc(decoded, func(a, b Result) int {
// 		return cmp.Compare(b.score, a.score)
// 	})

// 	return decoded[0].text
// }

func score(text string) int {
	score := 0
	for _, char := range []string{"e", "E", "t", "T", "a", "A", " "} {
		score += strings.Count(text, char)
	}
	return score
}

// func DetectSingleCharacterXOR() {
// 	file, _ := os.Open("challenge4.txt")
// 	defer file.Close()

// 	var lines []string
// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		lines = append(lines, scanner.Text())
// 	}
// 	if err := scanner.Err(); err != nil {
// 		log.Fatalf("Something wrong with file")
// 	}

// 	var results []string
// 	for _, line := range lines {
// 		results = append(results, solveSingleByteXOR(line))
// 	}

// 	maxScore := 0
// 	var answer string
// 	for _, result := range results {
// 		score := score(result)
// 		if score > maxScore {
// 			answer = result
// 			maxScore = score
// 		}
// 	}

// 	fmt.Println(maxScore, answer)
// }

func RepeatingKeyXOR(text string, key string) string {

	var result []byte
	for i, char := range []byte(text) {
		// fmt.Println(string(char), string(key[i%len(key)]))
		keyChar := key[i%len(key)]
		result = append(result, byte(char)^byte(keyChar))
	}
	hexString := hex.EncodeToString(result)
	return hexString
}

func getEditDistance(s1 []byte, s2 []byte) int {
	distance := 0
	for i := range s1 {
		distance += bits.OnesCount8(s1[i] ^ s2[i])
	}
	return distance
}

func BreakRepeatingKeyXOR() {

	s := openFile("challenge6.txt")
	sBytes, _ := base64.StdEncoding.DecodeString(s)

	// Calculate normalized Edit distance for each keysize
	type Distance struct {
		keysize  int
		distance float32
	}
	var distances []Distance
	for keysize := 2; keysize <= 40; keysize++ {
		normalizedDistance := float32(0)
		comparisons := 0
		for i := 0; i+keysize*2 < len(sBytes); i += keysize {
			normalizedDistance += float32(getEditDistance(sBytes[i:i+keysize], sBytes[i+keysize:i+keysize*2])) / float32(keysize)
			comparisons += 1
		}
		distances = append(distances, Distance{keysize: keysize, distance: normalizedDistance / float32(comparisons)})
	}
	slices.SortFunc(distances, func(a, b Distance) int {
		return cmp.Compare(a.distance, b.distance)
	})
	for i := range distances {
		fmt.Println(distances[i])
	}

	// Break ciphertext into blocks of keysize length
	keysize := distances[0].keysize
	var sBlocks [][]byte
	for i := 0; i+keysize < len(sBytes); i += keysize {
		sBlocks = append(sBlocks, sBytes[i:i+keysize])
	}
	fmt.Println("length of block", len(sBlocks[0]))

	// Orgnanize such that block n has nth byte of every block
	transposed := make([][]byte, keysize)
	for charI := range keysize {
		var iBlock []byte
		for blockI := range sBlocks {
			iBlock = append(iBlock, sBlocks[blockI][charI])
		}
		transposed[charI] = iBlock
	}
	fmt.Println("num blocks", len(transposed))

	// Solve single-char XOR for each block to get key
	key := ""
	for _, block := range transposed {
		key += solveSingleByteXOR(block)
	}
	fmt.Println("key:", key)

	// Decrypt with key
	decrypted := ""
	for i, byte := range sBytes {
		decrypted += string(byte ^ key[i%keysize])
	}
	fmt.Println("Decrypted text:", decrypted)
}

func solveSingleByteXOR(bytes []byte) string {
	type Result struct {
		key   int
		text  string
		score int
	}
	var decoded []Result

	for i := range 256 {
		xorResult := make([]byte, len(bytes))
		for j, b := range bytes {
			xorResult[j] = b ^ byte(i)
		}
		decodedText := string(xorResult)

		decoded = append(decoded, Result{key: i, text: decodedText, score: score(decodedText)})
	}

	slices.SortFunc(decoded, func(a, b Result) int {
		return cmp.Compare(b.score, a.score)
	})

	return string(decoded[0].key)
}

func BreakAESinECB(key string) {
	s := openFile("challenge7.txt")
	sBytes, _ := base64.StdEncoding.DecodeString(s)

	cipher, _ := aes.NewCipher([]byte(key))
	var decrypted []byte
	// split into blocks of 16 bytes
	for block := range slices.Chunk(sBytes, 16) {
		buf := make([]byte, 16)
		cipher.Decrypt(buf, block)
		decrypted = append(decrypted, buf...)
	}
	fmt.Println(string(decrypted))
}

func DetectAESinECB() {
	file, _ := os.Open("challenge8.txt")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	type Cipher struct {
		text         string
		uniqueblocks int
	}
	mostRepeats := Cipher{text: "", uniqueblocks: 100}

	for scanner.Scan() {
		hashes := make(map[string]int)
		line := scanner.Text()
		decodedLine, _ := hex.DecodeString(line)
		for block := range slices.Chunk(decodedLine, 16) {
			hashes[string(block)] += 1
		}
		// fmt.Println(line, len(hashes))
		if len(hashes) < mostRepeats.uniqueblocks {
			mostRepeats = Cipher{text: line, uniqueblocks: len(hashes)}
		}
	}
	fmt.Println(mostRepeats.text, mostRepeats.uniqueblocks)
}

func main() {
	DetectAESinECB()
}
