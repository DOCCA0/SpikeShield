#!/bin/bash

# Generate Go bindings for InsurancePool contract
# Make sure you have abigen installed: go install github.com/ethereum/go-ethereum/cmd/abigen@latest

CONTRACTS_DIR="../../contracts/artifacts/contracts"
OUTPUT_DIR="."

echo "Generating Go bindings for InsurancePool..."

ABI_FILE="${CONTRACTS_DIR}/InsurancePool.sol/InsurancePool.json"
python3 -c "import json,sys; print(json.dumps(json.load(open('$ABI_FILE'))['abi']))" | abigen --abi - --pkg=contracts --type=InsurancePool --out="${OUTPUT_DIR}/insurancepool.go"

if [ $? -eq 0 ]; then
    echo "✅ Successfully generated insurancepool.go"
else
    echo "❌ Failed to generate bindings. Make sure abigen is installed:"
    echo "   go install github.com/ethereum/go-ethereum/cmd/abigen@latest"
    exit 1
fi

echo "Generating Go bindings for MockUSDT..."

ABI_FILE_MUSDT="${CONTRACTS_DIR}/MockUSDT.sol/MockUSDT.json"
python3 -c "import json,sys; print(json.dumps(json.load(open('$ABI_FILE_MUSDT'))['abi']))" | abigen --abi - --pkg=contracts --type=MockUSDT --out="${OUTPUT_DIR}/musdt.go"

if [ $? -eq 0 ]; then
    echo "✅ Successfully generated musdt.go"
else
    echo "❌ Failed to generate MockUSDT bindings. Make sure abigen is installed:"
    echo "   go install github.com/ethereum/go-ethereum/cmd/abigen@latest"
    exit 1
fi
