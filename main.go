package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/satori/uuid"
)

var (
	nodeID uuid.UUID
	err error
)

func main() {
	s := NewServer()

	if nodeID, err = uuid.NewV4(); err != nil {
		log.Fatal(err)
	}

	log.Println("Started listening on localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", s.Router))
}

type Blockchain struct {
	Chain        []Block       `json:"chain"`
	Transactions []Transaction `json:"transactions"`
	Nodes 		 map[string]bool
}

// NewBlock create new block and adds it to chain
func (bc *Blockchain) NewBlock(proof string, prevBlockHash string) Block {
	newBlock := Block{
		Index:         int64(len(bc.Chain) + 1),
		Timestamp:     time.Now().Unix(),
		Transactions:  bc.Transactions,
		Proof:         proof,
		PrevBlockHash: prevBlockHash,
	}

	bc.Chain = append(bc.Chain, newBlock)
	bc.Transactions = nil
	return newBlock
}

// NewTransaction adds a new transaction to the list of transactions
func (bc *Blockchain) NewTransaction(tx Transaction) int64 {
	bc.Transactions = append(bc.Transactions, tx)
	return bc.LastBlock().Index + 1
}

// LastBlock returns the last block in the chain
func (bc *Blockchain) LastBlock() Block {
	return bc.Chain[len(bc.Chain)-1]
}

func (bc *Blockchain) RegNode(addr string) bool {
	_, found := bc.Nodes[addr]
	bc.Nodes[addr] = true
	return !found
}

func InitBlockchain() *Blockchain {
	bc := &Blockchain{}
	bc.NewBlock("100", "1")

	return bc
}

type Block struct {
	Index         int64         `json:"index"`
	Timestamp     int64         `json:"timestamp"`
	Transactions  []Transaction `json:"transactions"`
	Proof         string         `json:"proof"`
	PrevBlockHash string        `json:"prevblockhash"`
}

type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
}

type Server struct {
	Bc     *Blockchain
	Router *mux.Router
}

func (s *Server) NewTx(w http.ResponseWriter, r *http.Request) {
	log.Println("newtx endpoint")
	tx := Transaction{}
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	res := struct {
		LastBlockID int64 `json:"blockid"`
	}{
		LastBlockID: s.Bc.NewTransaction(tx),
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func (s *Server) Mine(w http.ResponseWriter, r *http.Request) {
	log.Println("mine endpoint")
	lastBlock := s.Bc.LastBlock()
	lastProof := lastBlock.Proof

	proof := ProofOfWork(lastProof)
	s.Bc.NewTransaction(Transaction{
		Sender: "0",
		Recipient: nodeID.String(),
		Amount: 20,
	})

	prevHash := Hash(lastBlock)

	s.Bc.NewBlock(proof, prevHash)
}

func (s *Server) Chain(w http.ResponseWriter, r *http.Request) {
	log.Println("chain endpoint")
	res := struct {
		Chain  []Block `json:"blockchain"`
		Length int     `json:"length"`
	}{
		Chain:  s.Bc.Chain,
		Length: len(s.Bc.Chain),
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func NewServer() *Server {
	s := &Server{
		Bc: InitBlockchain(),
	}
	InitRoutes(s)

	return s
}

func InitRoutes(s *Server) {
	r := mux.NewRouter()

	r.HandleFunc("/transaction/new", s.NewTx).Methods("POST")
	r.HandleFunc("/mine", s.Mine).Methods("GET")
	r.HandleFunc("/chain", s.Chain).Methods("GET")

	s.Router = r
}

// Hash hashes the block
func Hash(b Block) string {
	jsonBlock, err := json.Marshal(&b)
	if err != nil {
		log.Fatalf("could not marshal block: %s\n", err.Error())
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(jsonBlock)))
}

func ProofOfWork(lastProof string) string {
	proof := 0

	for {
		if ValidProof(lastProof, fmt.Sprintf("%d", proof)) {
			return fmt.Sprintf("%d", proof)
		}
		proof += 1
	}

	return fmt.Sprintf("%d", proof)
}

func ValidProof(lastProof, proof string) bool {
	result := fmt.Sprintf("%s", sha256.Sum256([]byte(lastProof+proof)))
	if result[:3] == "000" {
		return true
	}

	return false
}
