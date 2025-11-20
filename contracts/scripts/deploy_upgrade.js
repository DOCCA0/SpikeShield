const { ethers, upgrades } = require("hardhat");
require("dotenv").config({ path: "../.env" });

async function main() {
  const [deployer] = await ethers.getSigners();
  
  console.log("Upgrading contract with the account:", deployer.address);

  // Replace with your deployed proxy address
  const PROXY_ADDRESS = process.env.INSURANCE_POOL_ADDRESS_PROXY;
  console.log("Proxy address:", PROXY_ADDRESS);

  // Get the new implementation
  const InsurancePoolV2 = await ethers.getContractFactory("InsurancePool");
  
  console.log("Preparing upgrade...");
  const upgraded = await upgrades.upgradeProxy(PROXY_ADDRESS, InsurancePoolV2);
  await upgraded.waitForDeployment();
  
  console.log("✅ Contract upgraded successfully!");
  
  // Get new implementation address
  const newImplementationAddress = await upgrades.erc1967.getImplementationAddress(PROXY_ADDRESS);
  console.log("New implementation address:", newImplementationAddress);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
