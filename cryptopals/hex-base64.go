package main

import (
	"bufio"
	"cmp"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
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

func solveSingleByteXOR(hexString string) string {
	type Result struct {
		key   int
		text  string
		score int
	}
	var decoded []Result

	for i := range 256 {
		byteHexString, _ := hex.DecodeString(hexString)
		xorResult := make([]byte, len(byteHexString))
		for j, b := range byteHexString {
			xorResult[j] = b ^ byte(i)
		}
		decodedText := string(xorResult)

		decoded = append(decoded, Result{key: i, text: decodedText, score: score(decodedText)})
	}

	slices.SortFunc(decoded, func(a, b Result) int {
		return cmp.Compare(b.score, a.score)
	})

	return decoded[0].text
}

func score(text string) int {
	score := 0
	for _, char := range []string{"e", "E", "t", "T", "a", "A", " "} {
		score += strings.Count(text, char)
	}
	return score
}

func DetectSingleCharacterXOR() {
	file, _ := os.Open("challenge4.txt")
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Something wrong with file")
	}

	var results []string
	for _, line := range lines {
		results = append(results, solveSingleByteXOR(line))
	}

	maxScore := 0
	var answer string
	for _, result := range results {
		score := score(result)
		if score > maxScore {
			answer = result
			maxScore = score
		}
	}

	fmt.Println(maxScore, answer)
}

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

func main() {
	fmt.Println(RepeatingKeyXOR("Burning 'em, if you ain't quick and nimble\nI go crazy when I hear a cymbal", "ICE"))
}
