package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/multiformats/go-multiaddr"

	"github.com/edgelesssys/ego/eclient"
)

// Blockchain is a series of validated Blocks

type Blockchain []Block

type Block struct {
	Index    int
	Txs      string
	Hash     string
	Nonce    uint32
	PrevHash string
	Proof    []byte
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

var blockchain Blockchain
var newBlock Block

var mutex = &sync.Mutex{}

var uniqueID string = ""
var difficulty int = 1

func readBlockchain() Blockchain {
	content, err := ioutil.ReadFile("./../data/blockchain.json")
	if err != nil {
		path := "./../data"
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatalln(err)
		}
		blockchain = append(blockchain, genesisBlock)
		bytes, err := json.MarshalIndent(blockchain, "", "  ")
		if err != nil {
			log.Fatalln("Failed to initialize blockchain", err)
		}
		ioutil.WriteFile("./../data/blockchain.json", bytes, 0644)
		log.Println("Initialized blockchain with genesis block")
	}
	content, err = ioutil.ReadFile("./../data/blockchain.json")
	// Now let's unmarshall the data into `payload`
	var payload Blockchain
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	// Let's print the unmarshalled data!
	blockchain = payload
	return blockchain

}

func writeBlock(newBlock Block) {
	blockchain = append(blockchain, newBlock)

	bytes, err := json.MarshalIndent(blockchain, "", "  ")
	if err != nil {
		log.Println("Error marshalling blockchain")
		return
	}
	_ = ioutil.WriteFile("./../data/blockchain.json", bytes, 0644)

}

func writeBlockchain(chain Blockchain) {
	blockchain = chain

	bytes, err := json.MarshalIndent(blockchain, "", "  ")
	if err != nil {
		log.Println("Error marshalling blockchain")
		return
	}
	_ = ioutil.WriteFile("./../data/blockchain.json", bytes, 0644)

}

func calculateWork(chain Blockchain) int {
	totalZeros := 0

	for _, block := range chain {

		hashString := block.Hash
		zeros := countLeadingZeros(hashString)

		totalZeros += zeros
	}
	return totalZeros
}

func countLeadingZeros(hash string) int {
	var leadingZeros int
	for _, c := range hash {
		if c != '0' {
			break
		}
		leadingZeros++
	}
	return leadingZeros
}

// SHA256 hashing
func calculateHash(block Block) string {
	record := strconv.Itoa(block.Index) + block.PrevHash + strconv.Itoa(int(block.Nonce)) + string(block.Proof)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func validateHash(hash string) bool {
	// Count leading zeros
	leadingZeros := countLeadingZeros(hash)
	// Check against threshold
	if leadingZeros >= difficulty {
		return true
	} else {
		return false
	}
}

func handleStream(stream network.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	//go writeData(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			log.Println(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			chain := make([]Block, 0)
			if err := json.Unmarshal([]byte(str), &chain); err != nil {
				fmt.Println("Error unmarshalling received blockchain")
				log.Println(err)
			}
			mutex.Lock()
			if calculateWork(chain) > calculateWork(blockchain) {
				log.Println("heavier chain received")
				writeBlockchain(chain)
			}
			mutex.Unlock()
		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			mutex.Lock()
			bytes, err := json.Marshal(blockchain)
			if err != nil {
				log.Println(err)
			}
			mutex.Unlock()

			mutex.Lock()
			rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
			rw.Flush()
			mutex.Unlock()

		}
	}()

}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}
	if validateHash(newBlock.Hash) != true {
		return false
	}
	if verifyAttestation(newBlock.Proof, oldBlock.Hash) != true {
		return false
	}

	return true
}

func verifyAttestation(attestation []byte, oldHash string) bool {
	report, err := eclient.VerifyRemoteReport(attestation)
	if err != nil {
		log.Println(err)
		return false
	} else {
		if hex.EncodeToString(report.UniqueID) == uniqueID {
			data := report.Data
			if validateHash(string(data[:32])) {
				if string(data[:32]) == oldHash[:32] {
					return true
				} else {
					return false
				}

			} else {
				return false
			}

		} else {
			log.Println("invalid enclave")
			return false
		}

	}
}

func main() {
	godotenv.Load("../../.env")
	uniqueID = os.Getenv("UNIQUE_ID")
	blockchain = readBlockchain()

	help := flag.Bool("help", false, "Display Help")
	cfg := parseFlags()

	if *help {
		log.Printf("Simple example for peer discovery using mDNS. mDNS is great when you have multiple peers in local LAN.")
		log.Printf("Usage: \n   Run './chat-with-mdns'\nor Run './chat-with-mdns -host [host] -port [port] -rendezvous [string] -pid [proto ID]'\n")

		os.Exit(0)
	}

	log.Printf("[*] Listening on: %s with port: %d\n", cfg.listenHost, cfg.listenPort)

	ctx := context.Background()
	r := rand.Reader

	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.listenHost, cfg.listenPort))
	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	host, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}

	// Set a function as stream handler.
	// This function is called when a peer initiates a connection and starts a stream with this peer.
	host.SetStreamHandler(protocol.ID(cfg.ProtocolID), handleStream)

	log.Printf("\n[*] Your Multiaddress Is: /ip4/%s/tcp/%v/p2p/%s\n", cfg.listenHost, cfg.listenPort, host.ID().Pretty())

	peerChan := initMDNS(host, cfg.RendezvousString)
	for { // allows multiple peers to join
		peer := <-peerChan // will block untill we discover a peer
		log.Println("Found peer:", peer, ", connecting")

		if err := host.Connect(ctx, peer); err != nil {
			log.Println("Connection failed:", err)
			continue
		}

		// open a stream, this stream will be handled by handleStream other end
		stream, err := host.NewStream(ctx, peer.ID, protocol.ID(cfg.ProtocolID))

		if err != nil {
			log.Println("Stream open failed", err)
		} else {
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			go writeData(rw)
			//go readData(rw)
			log.Println("Connected to:", peer)
		}
	}

}
