package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/SebastiaanWouters/verigo/object"
	"github.com/SebastiaanWouters/verigo/repl"
	"github.com/edgelesssys/ego/enclave"
)

type Blockchain []Block

type Block struct {
	Index    int
	Txs      string
	Hash     string
	Nonce    uint32
	PrevHash string
	Proof    []byte
}

type Results struct {
	ResultMap map[string]object.Object
	Proof     []byte
}

type Tx struct {
	From   string
	To     string
	Amount int
	Sig    string
}

var operations uint32 = 0
var results []int
var blockchain Blockchain
var difficulty int = 1
var operationCount int = 0

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func monitorChan(c chan int) {
	for {
		<-c
		operationCount += 1
		if operationCount%100000 == 0 {
			tryBlock()
		}
	}
}

// SHA256 hashing
func calculateBlockHash(block Block) string {
	record := strconv.Itoa(block.Index) + block.PrevHash + strconv.Itoa(int(block.Nonce)) + string(block.Proof)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func calculateStringHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func validateHash(hash string) bool {
	// Count leading zeros
	var leadingZeros int
	for _, c := range hash {
		if c != '0' {
			break
		}
		leadingZeros++
	}

	// Check against threshold
	if leadingZeros >= difficulty {
		return true
	} else {
		return false
	}
}

func getLatestBlock() Block {
	content, err := ioutil.ReadFile("/data/blockchain.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	// Now let's unmarshall the data into `payload`
	var payload Blockchain
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	// Let's print the unmarshalled data!
	blockchain = payload
	return blockchain[len(blockchain)-1]

}

func main() {
	evalFile("/worker/script.vg")
}

func evalFile(path string) {
	c := make(chan int)
	go monitorChan(c)
	rMap := object.NewResultMap()
	dat, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln("Reading File Failed")
	}
	input := string(dat)

	repl.Eval(input, rMap, c)

	fmt.Println("operations executed: ", operationCount)

	saveResults(rMap)

}

func interpret(token string) {
	time.Sleep(200 * time.Millisecond)
	var1, err := strconv.Atoi(string(token[0]))
	check(err)
	var2, err := strconv.Atoi(string(token[2]))
	check(err)
	operator := string(token[1])
	var result int
	switch operator {
	case "+":
		result = var1 + var2
		operations += 1
	case "*":
		result = var1 * var2
		operations += 1
	case "/":
		result = var1 / var2
		operations += 1
	case "-":
		result = var1 - var2
		operations += 1
	}
	if operations%1 == 0 {
		tryBlock()
	}
	results = append(results, result)
}

func tryBlock() {
	latestBlock := getLatestBlock()
	hash := []byte(latestBlock.Hash)
	attestation := generateAttestationWithHash(hash)
	block := generateBlock(attestation)

	fmt.Println("Found block with hash: ", block.Hash)

	if validateHash(block.Hash) {
		fmt.Println("Valid Block Found")
		broadcast(block)
	}
}

func generateAttestation() []byte {
	buf := make([]byte, 10)
	binary.LittleEndian.PutUint32(buf, operations)
	byteArray := []byte{buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6], buf[7], buf[8], buf[8], buf[9]}
	fmt.Println("Operations: ", operations)
	report, err := enclave.GetRemoteReport(byteArray)
	check(err)
	return report

}

func generateAttestationWithHash(hash []byte) []byte {
	report, err := enclave.GetRemoteReport(hash)
	check(err)
	return report
}

func generateBlock(attestation []byte) Block {
	nonce := rand.Uint32()
	latestBlock := getLatestBlock()
	prevHash := latestBlock.Hash
	prevIndex := latestBlock.Index
	block := Block{
		Txs:      "",
		Nonce:    nonce,
		PrevHash: prevHash,
		Index:    prevIndex + 1,
		Proof:    attestation,
	}
	block.Hash = calculateBlockHash(block)
	return block
}

func parseTokens() []string {
	readFile, err := os.Open("/worker/script.js")
	check(err)

	var tokens []string
	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		tokens = append(tokens, fileScanner.Text())
	}
	readFile.Close()
	return tokens
}

func saveResults(results *object.ResultMap) {
	s := ""
	for key, value := range results.GetAll() {
		s += key
		s += value.Inspect()
	}
	hash := calculateStringHash(s)
	fmt.Println(hash)
	attestation := generateAttestationWithHash([]byte(hash))
	r := Results{ResultMap: results.GetAll(), Proof: attestation}
	file, _ := json.MarshalIndent(r, "", " ")
	_ = ioutil.WriteFile("/worker/result.json", file, 0644)
}

func broadcast(block Block) {

	// marshall data to json (like json_encode)
	marshalled, err := json.Marshal(block)
	if err != nil {
		log.Println("impossible to marshall block: %s", err)
	}
	req, err := http.NewRequest("POST", "http://localhost:4001/newblock", bytes.NewReader(marshalled))
	if err != nil {
		log.Println("impossible to build request: %s", err)

	}

	client := http.Client{Timeout: 10 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		log.Println("impossible to send request: %s", err)
		return
	}
	log.Printf("status Code: %d", res.StatusCode)

	defer res.Body.Close()
	// read body
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("impossible to read all body of response: %s", err)
	}
	log.Printf("res body: %s", string(resBody))
}
