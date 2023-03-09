package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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

var genesisBlock = Block{
	Index:    0,
	Txs:      "",
	Hash:     "0000000000000000000000000000000000000000",
	Nonce:    21,
	PrevHash: "",
	Proof:    []byte(""),
}

var operations uint32 = 0
var results []int
var blockchain Blockchain
var difficulty int = 1
var operationCount int = 0

const (
	OPS_PER_BLOCK = 10000000
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func opChanMonitor(c chan int) {
	for {
		<-c
		operationCount += 1
		if operationCount%OPS_PER_BLOCK == 0 {
			tryBlock()
		}
	}
}

func rChanMonitor(c chan object.Result) {
	for {
		writeToDisk(<-c)
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
		path := "/data"
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatalln(err)
		}
		blockchain = append(blockchain, genesisBlock)
		bytes, err := json.MarshalIndent(blockchain, "", "  ")
		if err != nil {
			log.Fatalln("Failed to initialize blockchain", err)
		}
		ioutil.WriteFile("/data/blockchain.json", bytes, 0644)
		log.Println("Initialized blockchain with genesis block")
	}
	content, err = ioutil.ReadFile("/data/blockchain.json")

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
	opChan := make(chan int)
	go opChanMonitor(opChan)
	rChan := make(chan object.Result)
	go rChanMonitor(rChan)
	dat, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln("Reading File Failed")
	}
	input := string(dat)

	repl.Eval(input, rChan, opChan)

	log.Println("Operations executed: ", operationCount)
}

func tryBlock() {
	latestBlock := getLatestBlock()
	hash := []byte(latestBlock.Hash)
	attestation := generateAttestationWithHash(hash)
	block := generateBlock(attestation)

	log.Println("Found a block with hash: ", block.Hash)

	if validateHash(block.Hash) {
		log.Println("Block satisfies the dificulty requirement, broadcasting to the network...")
		broadcast(block)
	}
}

func generateAttestation() []byte {
	buf := make([]byte, 10)
	binary.LittleEndian.PutUint32(buf, operations)
	byteArray := []byte{buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6], buf[7], buf[8], buf[8], buf[9]}
	log.Println("Operations: ", operations)
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

func writeToDisk(res object.Result) {
	filename := "/worker/results.json"

	err := checkFile(filename)
	if err != nil {
		log.Println(err)
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
	}

	data := []object.Result{}

	json.Unmarshal(file, &data)

	data = append(data, res)

	// Preparing the data to be marshalled and written.
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}

	err = ioutil.WriteFile(filename, dataBytes, 0644)
	if err != nil {
		log.Println(err)
	}

}

func checkFile(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			return err
		}
	}
	return nil
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
}
