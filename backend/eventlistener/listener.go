package eventlistener

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"spikeshield/db"
	"spikeshield/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EventListener monitors and syncs contract events to the database
type EventListener struct {
	client          *ethclient.Client
	contractAddress common.Address
	contractABI     abi.ABI
	pollInterval    time.Duration
}

// TokenListener listens to ERC20 Transfer events for a specific token and updates balances
type TokenListener struct {
	client       *ethclient.Client
	tokenAddress common.Address
	contractABI  abi.ABI
	pollInterval time.Duration
	decimals     int
}

// PolicyPurchasedEvent represents the PolicyPurchased event from the contract
type PolicyPurchasedEvent struct {
	PolicyId   *big.Int
	Premium    *big.Int
	Coverage   *big.Int
	ExpiryTime *big.Int
}

// PayoutExecutedEvent represents the PayoutExecuted event from the contract
type PayoutExecutedEvent struct {
	PolicyId *big.Int
	Amount   *big.Int
}

// NewEventListener creates a new event listener instance
func NewEventListener(rpcURL, contractAddr string, pollInterval time.Duration) (*EventListener, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	// Parse contract ABI
	contractABI, err := abi.JSON(strings.NewReader(getInsurancePoolABI()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	return &EventListener{
		client:          client,
		contractAddress: common.HexToAddress(contractAddr),
		contractABI:     contractABI,
		pollInterval:    pollInterval,
	}, nil
}

// NewTokenListener creates a listener for an ERC20 token Transfer events
func NewTokenListener(rpcURL, tokenAddr string, pollInterval time.Duration, decimals int) (*TokenListener, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	contractABI, err := abi.JSON(strings.NewReader(getERC20ABI()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}

	return &TokenListener{
		client:       client,
		tokenAddress: common.HexToAddress(tokenAddr),
		contractABI:  contractABI,
		pollInterval: pollInterval,
		decimals:     decimals,
	}, nil
}

// Start begins listening for contract events
func (el *EventListener) Start(ctx context.Context) error {
	utils.LogInfo("🎧 Event listener started for contract: %s", el.contractAddress.Hex())

	ticker := time.NewTicker(el.pollInterval)
	defer ticker.Stop()

	// Initial sync
	if err := el.syncEvents(ctx); err != nil {
		utils.LogError("Initial event sync failed: %v", err)
	}

	// Poll for new events
	for {
		select {
		case <-ctx.Done():
			utils.LogInfo("Event listener stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := el.syncEvents(ctx); err != nil {
				utils.LogError("Event sync failed: %v", err)
			}
		}
	}
}

// Close closes the Ethereum client connection
func (el *EventListener) Close() {
	if el.client != nil {
		el.client.Close()
	}
}

// Close closes token listener client
func (tl *TokenListener) Close() {
	if tl.client != nil {
		tl.client.Close()
	}
}

// Start begins listening for token Transfer events
func (tl *TokenListener) Start(ctx context.Context) error {
	utils.LogInfo("🎧 Token listener started for token: %s", tl.tokenAddress.Hex())

	ticker := time.NewTicker(tl.pollInterval)
	defer ticker.Stop()

	// Initial sync
	if err := tl.syncEvents(ctx); err != nil {
		utils.LogError("Initial token event sync failed: %v", err)
	}

	// Poll for new events
	for {
		select {
		case <-ctx.Done():
			utils.LogInfo("Token listener stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := tl.syncEvents(ctx); err != nil {
				utils.LogError("Token event sync failed: %v", err)
			}
		}
	}
}

// syncEvents fetches and processes new events since last sync
func (el *EventListener) syncEvents(ctx context.Context) error {
	// Get last synced block
	lastBlock, err := db.GetLastSyncedBlock(el.contractAddress)
	if err != nil {
		return fmt.Errorf("failed to get last synced block: %w", err)
	}

	// Get current block number
	currentBlock, err := el.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block: %w", err)
	}

	if lastBlock >= currentBlock {
		return nil // No new blocks
	}

	// If this is the first sync, start from recent blocks (last 1000 blocks or from lastBlock)
	if lastBlock == 0 {
		if currentBlock > 1000 {
			lastBlock = currentBlock - 1000
		}
	}

	utils.LogInfo("Syncing events from block %d to %d", lastBlock+1, currentBlock)

	// Fetch events in chunks to avoid RPC limits
	chunkSize := uint64(1000)
	for fromBlock := lastBlock + 1; fromBlock <= currentBlock; fromBlock += chunkSize {
		toBlock := fromBlock + chunkSize - 1
		if toBlock > currentBlock {
			toBlock = currentBlock
		}

		if err := el.fetchAndProcessEvents(ctx, fromBlock, toBlock); err != nil {
			return fmt.Errorf("failed to fetch events [%d-%d]: %w", fromBlock, toBlock, err)
		}
	}

	// Update last synced block
	if err := db.UpdateLastSyncedBlock(el.contractAddress, currentBlock); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	return nil
}

// syncEvents fetches and processes Transfer events for the token
func (tl *TokenListener) syncEvents(ctx context.Context) error {
	// Get last synced block
	lastBlock, err := db.GetLastSyncedBlock(tl.tokenAddress)
	if err != nil {
		return fmt.Errorf("failed to get last synced block: %w", err)
	}

	// Get current block number
	currentBlock, err := tl.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block: %w", err)
	}

	if lastBlock >= currentBlock {
		return nil // No new blocks
	}

	// If this is the first sync, start from recent blocks (last 1000 blocks or from lastBlock)
	if lastBlock == 0 {
		if currentBlock > 1000 {
			lastBlock = currentBlock - 1000
		}
	}

	utils.LogInfo("Syncing token events from block %d to %d", lastBlock+1, currentBlock)

	// Fetch events in chunks to avoid RPC limits
	chunkSize := uint64(1000)
	for fromBlock := lastBlock + 1; fromBlock <= currentBlock; fromBlock += chunkSize {
		toBlock := fromBlock + chunkSize - 1
		if toBlock > currentBlock {
			toBlock = currentBlock
		}

		if err := tl.fetchAndProcessEvents(ctx, fromBlock, toBlock); err != nil {
			return fmt.Errorf("failed to fetch token events [%d-%d]: %w", fromBlock, toBlock, err)
		}
	}

	// Update last synced block
	if err := db.UpdateLastSyncedBlock(tl.tokenAddress, currentBlock); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	return nil
}

// fetchAndProcessEvents fetches events from a block range and processes them
func (el *EventListener) fetchAndProcessEvents(ctx context.Context, fromBlock, toBlock uint64) error {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{el.contractAddress},
	}

	logs, err := el.client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	for _, vLog := range logs {
		if err := el.processLog(vLog); err != nil {
			utils.LogError("Failed to process log (tx: %s, index: %d): %v", vLog.TxHash.Hex(), vLog.Index, err)
			continue
		}
	}

	if len(logs) > 0 {
		utils.LogInfo("Processed %d events from blocks %d-%d", len(logs), fromBlock, toBlock)
	}

	return nil
}

// fetchAndProcessEvents for token
func (tl *TokenListener) fetchAndProcessEvents(ctx context.Context, fromBlock, toBlock uint64) error {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{tl.tokenAddress},
	}

	logs, err := tl.client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	for _, vLog := range logs {
		if err := tl.processLog(vLog); err != nil {
			utils.LogError("Failed to process token log (tx: %s, index: %d): %v", vLog.TxHash.Hex(), vLog.Index, err)
			continue
		}
	}

	if len(logs) > 0 {
		utils.LogInfo("Processed %d token events from blocks %d-%d", len(logs), fromBlock, toBlock)
	}

	return nil
}

// FullResync removed: full log-scan approach deprecated in favor of on-chain
// enumeration helpers added into contracts (getHoldersBalances / getAllPolicyOwners)

// processLog processes a single log entry
func (el *EventListener) processLog(vLog types.Log) error {
	eventSignature := vLog.Topics[0].Hex()

	switch eventSignature {
	case el.contractABI.Events["PolicyPurchased"].ID.Hex():
		return el.handlePolicyPurchased(vLog)
	case el.contractABI.Events["PayoutExecuted"].ID.Hex():
		return el.handlePayoutExecuted(vLog)
	default:
		// Unknown event, skip
		return nil
	}
}

// processLog handles token Transfer events
func (tl *TokenListener) processLog(vLog types.Log) error {
	eventSignature := vLog.Topics[0].Hex()

	if eventSignature == tl.contractABI.Events["Transfer"].ID.Hex() {
		return tl.handleTransfer(vLog)
	}
	return nil
}

// handleTransfer processes Transfer events and updates balances in DB
func (tl *TokenListener) handleTransfer(vLog types.Log) error {
	// Parse indexed from and to addresses from topics
	if len(vLog.Topics) < 3 {
		return fmt.Errorf("invalid Transfer log topics")
	}

	from := common.HexToAddress(vLog.Topics[1].Hex())
	to := common.HexToAddress(vLog.Topics[2].Hex())

	// Parse non-indexed data (value)
	var event struct {
		Value *big.Int
	}
	if err := tl.contractABI.UnpackIntoInterface(&event, "Transfer", vLog.Data); err != nil {
		return fmt.Errorf("failed to unpack Transfer event: %w", err)
	}

	// Convert value to float using token decimals
	denom := math.Pow10(tl.decimals)
	amountFloat := new(big.Float).Quo(new(big.Float).SetInt(event.Value), new(big.Float).SetFloat64(denom))
	amountVal, _ := amountFloat.Float64()

	utils.LogInfo("🔁 Transfer: from=%s to=%s amount=%f", from.Hex(), to.Hex(), amountVal)

	tokenHex := tl.tokenAddress.Hex()

	// Update 'from' balance (skip zero address)
	zero := common.Address{}
	if from != zero {
		if err := db.UpsertBalanceFromChain(tl.client, utils.AppConfig, tokenHex, from.Hex()); err != nil {
			utils.LogError("Failed to refresh on-chain balance for from=%s: %v", from.Hex(), err)
		}
	}

	// Update 'to' balance (skip zero address)
	if to != zero {
		if err := db.UpsertBalanceFromChain(tl.client, utils.AppConfig, tokenHex, to.Hex()); err != nil {
			utils.LogError("Failed to refresh on-chain balance for to=%s: %v", to.Hex(), err)
		}
	}

	return nil
}

// handlePolicyPurchased processes PolicyPurchased events
func (el *EventListener) handlePolicyPurchased(vLog types.Log) error {
	var event PolicyPurchasedEvent

	// Parse indexed user address from topics[1]
	user := common.HexToAddress(vLog.Topics[1].Hex())

	// Parse non-indexed data
	err := el.contractABI.UnpackIntoInterface(&event, "PolicyPurchased", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack PolicyPurchased event: %w", err)
	}

	utils.LogInfo("📝 PolicyPurchased: user=%s, policyId=%s, premium=%s, coverage=%s",
		user.Hex(), event.PolicyId.String(), event.Premium.String(), event.Coverage.String())

	// Store event in database
	return db.InsertPolicyFromEvent(user, event.Premium, event.Coverage, event.ExpiryTime, vLog)
}

// handlePayoutExecuted processes PayoutExecuted events
func (el *EventListener) handlePayoutExecuted(vLog types.Log) error {
	var event PayoutExecutedEvent

	// Parse indexed user address from topics[1]
	user := common.HexToAddress(vLog.Topics[1].Hex())

	// Parse non-indexed data
	err := el.contractABI.UnpackIntoInterface(&event, "PayoutExecuted", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack PayoutExecuted event: %w", err)
	}

	utils.LogInfo("💰 PayoutExecuted: user=%s, policyId=%s, amount=%s",
		user.Hex(), event.PolicyId.String(), event.Amount.String())

	// Store event in database
	return db.InsertPayoutFromEvent(user, event.PolicyId, event.Amount, vLog)
}

// getInsurancePoolABI returns the ABI string for the InsurancePool contract
func getInsurancePoolABI() string {
	return `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "internalType": "address", "name": "user", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "policyId", "type": "uint256"},
				{"indexed": false, "internalType": "uint256", "name": "premium", "type": "uint256"},
				{"indexed": false, "internalType": "uint256", "name": "coverage", "type": "uint256"},
				{"indexed": false, "internalType": "uint256", "name": "expiryTime", "type": "uint256"}
			],
			"name": "PolicyPurchased",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "internalType": "address", "name": "user", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "policyId", "type": "uint256"},
				{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"}
			],
			"name": "PayoutExecuted",
			"type": "event"
		}
	]`
}

// getERC20ABI returns minimal ERC20 ABI for Transfer event and balanceOf
func getERC20ABI() string {
	return `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "internalType": "address", "name": "from", "type": "address"},
				{"indexed": true, "internalType": "address", "name": "to", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "value", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		},
		{
			"inputs": [{"internalType": "address", "name": "account", "type": "address"}],
			"name": "balanceOf",
			"outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
			"stateMutability": "view",
			"type": "function"
		}
	]`
}
