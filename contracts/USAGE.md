# Contract Usage Guide

## Calling On-Chain Methods

### Using ethers.js
```javascript
const { ethers } = require("hardhat");

// Get contract instance
const InsurancePool = await ethers.getContractFactory("InsurancePool");
const insurance = await InsurancePool.attach("0xbe4636f196f5a081074EBE435E82D38C9991F49D");

const MUSDTPool = await ethers.getContractFactory("MockUSDT");
const musdt = await MUSDTPool.attach("0x6259CD72e5d1347143F1097bc7c97928B3acCf02");

const aaa = await insurance.xxx
const bbb = await musdt.yyy

// With signer
const [owner, user1] = await ethers.getSigners();
const poolAsUser = pool.connect(user1);
await poolAsUser.deposit(amount);
```

### Using web3.js
```javascript
const contract = new web3.eth.Contract(abi, address);

// Call (read)
const balance = await contract.methods.poolBalance().call();

// Send (write)
await contract.methods.deposit(amount).send({ from: account });
```

## Common Hardhat Commands

```bash
# Install dependencies
npm install

# Compile contracts
npx hardhat compile

# Run tests
npx hardhat test
npx hardhat test test/InsurancePool.test.js  # Single file
npx hardhat test --grep "deposit"            # Specific test

# Deploy
npx hardhat run scripts/deploy.js --network localhost
npx hardhat run scripts/deploy.js --network sepolia
npx hardhat run scripts/deploy.js --network mainnet

# Start local node
npx hardhat node

# Console
npx hardhat console --network localhost
npx hardhat console --network sepolia

# Verify contract
npx hardhat verify --network sepolia DEPLOYED_CONTRACT_ADDRESS "Constructor arg1" "arg2"

# Clean
npx hardhat clean

# Check contract size
npx hardhat size-contracts

# Gas report
REPORT_GAS=true npx hardhat test

# Coverage
npx hardhat coverage

# Flatten contract (for verification)
npx hardhat flatten contracts/InsurancePool.sol > flattened.sol
```

## Quick Scripts

### Deploy and interact
```javascript
// scripts/interact.js
async function main() {
  const [deployer] = await ethers.getSigners();
  const pool = await ethers.getContractAt("InsurancePool", "0x...");
  
  // Interact
  const tx = await pool.deposit(ethers.parseUnits("100", 6));
  await tx.wait();
  console.log("Deposited");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
```

Run: `npx hardhat run scripts/interact.js --network sepolia`

## Environment Setup

Create `.env`:
```
PRIVATE_KEY=your_private_key
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY
ETHERSCAN_API_KEY=your_etherscan_key
```
