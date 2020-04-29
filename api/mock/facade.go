package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-proxy-go/data"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// Facade is the mock implementation of a node's router handler
type Facade struct {
	GetAccountHandler               func(address string) (*data.Account, error)
	GetTransactionsHandler          func(address string) ([]data.DatabaseTransaction, error)
	SendTransactionHandler          func(tx *data.ApiTransaction) (int, string, error)
	SendMultipleTransactionsHandler func(txs []*data.ApiTransaction) (uint64, error)
	SendUserFundsCalled             func(receiver string, value *big.Int) error
	ExecuteSCQueryHandler           func(query *data.SCQuery) (*vmcommon.VMOutput, error)
	GetHeartbeatDataHandler         func() (*data.HeartbeatResponse, error)
	ValidatorStatisticsHandler      func() (map[string]*data.ValidatorApiResponse, error)
	TransactionCostRequestHandler   func(tx *data.ApiTransaction) (string, error)
	GetShardStatusHandler           func(shardID uint32) (map[string]interface{}, error)
	GetEpochMetricsHandler          func(shardID uint32) (map[string]interface{}, error)
	GetTransactionStatusHandler     func(txHash string) (string, error)
	GetHighestBlockNonceHandler     func() (uint64, error)
	GetBlockByNonceHandler          func(nonce uint64) (data.ApiBlock, error)
}

// ValidatorStatistics is the mock implementation of a handler's ValidatorStatistics method
func (f *Facade) ValidatorStatistics() (map[string]*data.ValidatorApiResponse, error) {
	return f.ValidatorStatisticsHandler()
}

// GetShardStatus --
func (f *Facade) GetShardStatus(shardID uint32) (map[string]interface{}, error) {
	return f.GetShardStatusHandler(shardID)
}

// GetEpochMetrics --
func (f *Facade) GetEpochMetrics(shardID uint32) (map[string]interface{}, error) {
	return f.GetEpochMetricsHandler(shardID)
}

// GetAccount is the mock implementation of a handler's GetAccount method
func (f *Facade) GetAccount(address string) (*data.Account, error) {
	return f.GetAccountHandler(address)
}

// GetTransactions --
func (f *Facade) GetTransactions(address string) ([]data.DatabaseTransaction, error) {
	return f.GetTransactionsHandler(address)
}

// SendTransaction is the mock implementation of a handler's SendTransaction method
func (f *Facade) SendTransaction(tx *data.ApiTransaction) (int, string, error) {
	return f.SendTransactionHandler(tx)
}

// SendMultipleTransactions is the mock implementation of a handler's SendMultipleTransactions method
func (f *Facade) SendMultipleTransactions(txs []*data.ApiTransaction) (uint64, error) {
	return f.SendMultipleTransactionsHandler(txs)
}

// TransactionCostRequest --
func (f *Facade) TransactionCostRequest(tx *data.ApiTransaction) (string, error) {
	return f.TransactionCostRequestHandler(tx)
}

// GetTransactionStatus --
func (f *Facade) GetTransactionStatus(txHash string) (string, error) {
	return f.GetTransactionStatusHandler(txHash)
}

// SendUserFunds is the mock implementation of a handler's SendUserFunds method
func (f *Facade) SendUserFunds(receiver string, value *big.Int) error {
	return f.SendUserFundsCalled(receiver, value)
}

// ExecuteSCQuery is a mock implementation.
func (f *Facade) ExecuteSCQuery(query *data.SCQuery) (*vmcommon.VMOutput, error) {
	return f.ExecuteSCQueryHandler(query)
}

// GetHeartbeatData is the mock implementation of a handler's GetHeartbeatData method
func (f *Facade) GetHeartbeatData() (*data.HeartbeatResponse, error) {
	return f.GetHeartbeatDataHandler()
}

// GetHighestBlockNonce -
func (f *Facade) GetHighestBlockNonce() (uint64, error) {
	return f.GetHighestBlockNonceHandler()
}

// GetBlockByNonce -
func (f *Facade) GetBlockByNonce(nonce uint64) (data.ApiBlock, error) {
	return f.GetBlockByNonceHandler(nonce)
}

// WrongFacade is a struct that can be used as a wrong implementation of the node router handler
type WrongFacade struct {
}
