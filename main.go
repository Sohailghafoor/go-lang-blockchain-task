
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	PreviousHash string        `json:"previous_hash"`
	Hash         string        `json:"hash"`
	Nonce        int           `json:"nonce"`
}

type Transaction struct {
	Author    string  `json:"author"`
	Content   string  `json:"content"`
	Timestamp float64 `json:"timestamp"`
}

type Blockchain struct {
	UnconfirmedTransactions []Transaction `json:"unconfirmed_transactions"`
	Chain                   []Block       `json:"chain"`
	mutex                   sync.Mutex
	difficulty              int
}

type Node struct {
	Address string `json:"node_address"`
}

var (
	blockchain Blockchain
	peers      = make(map[string]bool)
	mutex      sync.Mutex
)

func main() {
	blockchain = Blockchain{
		UnconfirmedTransactions: []Transaction{},
		Chain:                   []Block{},
		difficulty:              2,
	}
	blockchain.CreateGenesisBlock()

	http.HandleFunc("/new_transaction", NewTransactionHandler)
	http.HandleFunc("/chain", GetChainHandler)
	http.HandleFunc("/mine", MineHandler)
	http.HandleFunc("/register_node", RegisterNodeHandler)
	http.HandleFunc("/register_with", RegisterWithExistingNodeHandler)
	http.HandleFunc("/add_block", AddBlockHandler)
	http.HandleFunc("/pending_tx", GetPendingTransactionsHandler)

	go func() {
		log.Println("Listening on http://localhost:8000")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals
		os.Exit(0)
	}()

	select {}
}

func (b *Blockchain) CreateGenesisBlock() {
	block := Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []Transaction{},
		PreviousHash: "0",
		Nonce:        0,
	}
	block.Hash = block.ComputeHash()
	b.Chain = append(b.Chain, block)
}

func (b *Blockchain) LastBlock() Block {
	return b.Chain[len(b.Chain)-1]
}

func (b *Blockchain) AddBlock(block Block, proof string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if block.PreviousHash != b.LastBlock().Hash {
		return fmt.Errorf("Previous hash incorrect")
	}

	if !b.IsValidProof(block, proof) {
		return fmt.Errorf("Block proof invalid")
	}

	block.Hash = proof
	b.Chain = append(b.Chain, block)

	return nil
}

func (b *Blockchain) AddNewTransaction(tx Transaction) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.UnconfirmedTransactions = append(b.UnconfirmedTransactions, tx)
}

func (b *Blockchain) ProofOfWork(block Block) string {
	block.Nonce = 0

	for !IsValidHash(block.ComputeHash(), b.difficulty) {
		block.Nonce++
	}

	return block.ComputeHash()
}

func (b *Blockchain) IsValidProof(block Block, hash string) bool {
	return IsValidHash(hash, b.difficulty) && hash == block.ComputeHash()
}

func (b *Blockchain) CheckChainValidity(chain []Block) bool {
	previousHash :=
