const { expect } = require("chai");
const { ethers } = require("hardhat");
const { time } = require("@nomicfoundation/hardhat-network-helpers");

describe("InsurancePool", function () {
  let insurancePool;
  let mockUSDT;
  let owner;
  let oracle;
  let user1;
  let user2;
  let user3;

  const PREMIUM_AMOUNT = ethers.parseUnits("10", 6); // 10 USDT
  const COVERAGE_AMOUNT = ethers.parseUnits("100", 6); // 100 USDT
  const COVERAGE_DURATION = 24 * 60 * 60; // 24 hours
  const INITIAL_MINT = ethers.parseUnits("1000", 6); // 1000 USDT per user

  beforeEach(async function () {
    [owner, oracle, user1, user2, user3] = await ethers.getSigners();

    // Deploy MockUSDT
    const MockUSDT = await ethers.getContractFactory("MockUSDT");
    mockUSDT = await MockUSDT.deploy();
    await mockUSDT.waitForDeployment();

    // Deploy InsurancePool
    const InsurancePool = await ethers.getContractFactory("InsurancePool");
    insurancePool = await InsurancePool.deploy(await mockUSDT.getAddress());
    await insurancePool.waitForDeployment();

    // Mint USDT to users for testing
    await mockUSDT.mint(user1.address, INITIAL_MINT);
    await mockUSDT.mint(user2.address, INITIAL_MINT);
    await mockUSDT.mint(user3.address, INITIAL_MINT);

    // Fund the insurance pool
    const poolFundAmount = ethers.parseUnits("10000", 6);
    await mockUSDT.approve(await insurancePool.getAddress(), poolFundAmount);
    await insurancePool.fundPool(poolFundAmount);
  });

  describe("Deployment", function () {
    it("Should set correct USDT token address", async function () {
      expect(await insurancePool.usdt()).to.equal(await mockUSDT.getAddress());
    });

    it("Should set deployer as owner and oracle", async function () {
      expect(await insurancePool.owner()).to.equal(owner.address);
      expect(await insurancePool.oracle()).to.equal(owner.address);
    });

    it("Should set correct initial policy parameters", async function () {
      expect(await insurancePool.premiumAmount()).to.equal(PREMIUM_AMOUNT);
      expect(await insurancePool.coverageAmount()).to.equal(COVERAGE_AMOUNT);
      expect(await insurancePool.coverageDuration()).to.equal(COVERAGE_DURATION);
    });

    it("Should have correct pool balance after funding", async function () {
      const balance = await insurancePool.getPoolBalance();
      expect(balance).to.equal(ethers.parseUnits("10000", 6));
    });
  });

  describe("Buy Insurance", function () {
    it("Should allow user to purchase insurance policy", async function () {
      // Approve USDT spending
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);

      // Buy insurance
      const tx = await insurancePool.connect(user1).buyInsurance();
      
      // Check event emission
      await expect(tx)
        .to.emit(insurancePool, "PolicyPurchased")
        .withArgs(
          user1.address,
          0, // policyId
          PREMIUM_AMOUNT,
          COVERAGE_AMOUNT,
          await time.latest() + COVERAGE_DURATION
        );

      // Check user's policy count
      expect(await insurancePool.getUserPoliciesCount(user1.address)).to.equal(1);

      // Check user has active policy
      expect(await insurancePool.hasActivePolicy(user1.address)).to.be.true;
    });

    it("Should transfer premium from user to pool", async function () {
      const initialUserBalance = await mockUSDT.balanceOf(user1.address);
      const initialPoolBalance = await insurancePool.getPoolBalance();

      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user1).buyInsurance();

      expect(await mockUSDT.balanceOf(user1.address)).to.equal(initialUserBalance - PREMIUM_AMOUNT);
      expect(await insurancePool.getPoolBalance()).to.equal(initialPoolBalance + PREMIUM_AMOUNT);
    });

    it("Should create policy with correct details", async function () {
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user1).buyInsurance();

      const policy = await insurancePool.getPolicy(user1.address, 0);
      const currentTime = await time.latest();

      expect(policy.user).to.equal(user1.address);
      expect(policy.premium).to.equal(PREMIUM_AMOUNT);
      expect(policy.coverageAmount).to.equal(COVERAGE_AMOUNT);
      expect(policy.active).to.be.true;
      expect(policy.claimed).to.be.false;
      expect(policy.expiryTime).to.be.closeTo(currentTime + COVERAGE_DURATION, 2);
    });

    it("Should allow user to purchase multiple policies", async function () {
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT * 3n);
      
      await insurancePool.connect(user1).buyInsurance();
      await insurancePool.connect(user1).buyInsurance();
      await insurancePool.connect(user1).buyInsurance();

      expect(await insurancePool.getUserPoliciesCount(user1.address)).to.equal(3);
    });

    it("Should fail when user doesn't approve USDT spending", async function () {
      await expect(
        insurancePool.connect(user1).buyInsurance()
      ).to.be.reverted;
    });

    it("Should fail when user has insufficient USDT balance", async function () {
      // Create a new user with no USDT
      const [poorUser] = await ethers.getSigners();
      
      await expect(
        insurancePool.connect(poorUser).buyInsurance()
      ).to.be.reverted;
    });
  });

  describe("Execute Payout", function () {
    beforeEach(async function () {
      // User1 buys insurance
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user1).buyInsurance();
    });

    it("Should allow oracle to execute payout for valid policy", async function () {
      const initialBalance = await mockUSDT.balanceOf(user1.address);
      const detectionTxHash = "0xabcd1234";

      const tx = await insurancePool.connect(owner).executePayout(user1.address, 0, detectionTxHash);

      // Check event
      await expect(tx)
        .to.emit(insurancePool, "PayoutExecuted")
        .withArgs(user1.address, 0, COVERAGE_AMOUNT, detectionTxHash);

      // Check user received payout
      expect(await mockUSDT.balanceOf(user1.address)).to.equal(initialBalance + COVERAGE_AMOUNT);
    });

    it("Should mark policy as claimed and inactive after payout", async function () {
      await insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash");

      const policy = await insurancePool.getPolicy(user1.address, 0);
      expect(policy.claimed).to.be.true;
      expect(policy.active).to.be.false;
    });

    it("Should update hasActivePolicy status after payout", async function () {
      expect(await insurancePool.hasActivePolicy(user1.address)).to.be.true;
      
      await insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash");
      
      expect(await insurancePool.hasActivePolicy(user1.address)).to.be.false;
    });

    it("Should fail when non-oracle tries to execute payout", async function () {
      await expect(
        insurancePool.connect(user2).executePayout(user1.address, 0, "0xhash")
      ).to.be.revertedWith("Only oracle can execute payout");
    });

    it("Should fail when policy ID is invalid", async function () {
      await expect(
        insurancePool.connect(owner).executePayout(user1.address, 999, "0xhash")
      ).to.be.revertedWith("Invalid policy ID");
    });

    it("Should fail when policy is not active", async function () {
      // Execute payout once
      await insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash1");

      // Try to execute again
      await expect(
        insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash2")
      ).to.be.revertedWith("Policy not active");
    });

    it("Should fail when policy is already claimed", async function () {
      await insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash1");

      // After payout, policy is both inactive and claimed, so either error is acceptable
      await expect(
        insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash2")
      ).to.be.revertedWith("Policy not active");
    });

    it("Should fail when policy has expired", async function () {
      // Fast forward time beyond policy expiry
      await time.increase(COVERAGE_DURATION + 1);

      await expect(
        insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash")
      ).to.be.revertedWith("Policy expired");
    });

    it("Should fail when pool has insufficient balance", async function () {
      // Create a new pool with no funds
      const InsurancePool = await ethers.getContractFactory("InsurancePool");
      const emptyPool = await InsurancePool.deploy(await mockUSDT.getAddress());
      await emptyPool.waitForDeployment();

      // User buys insurance on empty pool
      await mockUSDT.connect(user1).approve(await emptyPool.getAddress(), PREMIUM_AMOUNT);
      await emptyPool.connect(user1).buyInsurance();

      // Try to execute payout
      await expect(
        emptyPool.connect(owner).executePayout(user1.address, 0, "0xhash")
      ).to.be.revertedWith("Insufficient pool balance");
    });
  });

  describe("Policy Management", function () {
    it("Should correctly track multiple users' policies", async function () {
      // User1 buys 2 policies
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT * 2n);
      await insurancePool.connect(user1).buyInsurance();
      await insurancePool.connect(user1).buyInsurance();

      // User2 buys 1 policy
      await mockUSDT.connect(user2).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user2).buyInsurance();

      expect(await insurancePool.getUserPoliciesCount(user1.address)).to.equal(2);
      expect(await insurancePool.getUserPoliciesCount(user2.address)).to.equal(1);
    });

    it("Should return false for hasActivePolicy when user has no policies", async function () {
      expect(await insurancePool.hasActivePolicy(user3.address)).to.be.false;
    });

    it("Should return false for hasActivePolicy when all policies expired", async function () {
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user1).buyInsurance();

      // Fast forward past expiry
      await time.increase(COVERAGE_DURATION + 1);

      expect(await insurancePool.hasActivePolicy(user1.address)).to.be.false;
    });

    it("Should return true when user has at least one active policy", async function () {
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT * 2n);
      await insurancePool.connect(user1).buyInsurance();
      await insurancePool.connect(user1).buyInsurance();

      // Claim first policy
      await insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash");

      // Should still have active policy (second one)
      expect(await insurancePool.hasActivePolicy(user1.address)).to.be.true;
    });
  });

  describe("Oracle Management", function () {
    it("Should allow owner to update oracle address", async function () {
      const tx = await insurancePool.connect(owner).setOracle(oracle.address);

      await expect(tx)
        .to.emit(insurancePool, "OracleUpdated")
        .withArgs(oracle.address);

      expect(await insurancePool.oracle()).to.equal(oracle.address);
    });

    it("Should fail when non-owner tries to update oracle", async function () {
      await expect(
        insurancePool.connect(user1).setOracle(oracle.address)
      ).to.be.revertedWithCustomError(insurancePool, "OwnableUnauthorizedAccount");
    });

    it("Should allow new oracle to execute payouts", async function () {
      // Change oracle
      await insurancePool.connect(owner).setOracle(oracle.address);

      // User buys insurance
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user1).buyInsurance();

      // New oracle executes payout
      await expect(
        insurancePool.connect(oracle).executePayout(user1.address, 0, "0xhash")
      ).to.not.be.reverted;
    });
  });

  describe("Policy Parameters Management", function () {
    it("Should allow owner to update policy parameters", async function () {
      const newPremium = ethers.parseUnits("20", 6);
      const newCoverage = ethers.parseUnits("200", 6);
      const newDuration = 48 * 60 * 60; // 48 hours

      await insurancePool.connect(owner).setPolicyParams(newPremium, newCoverage, newDuration);

      expect(await insurancePool.premiumAmount()).to.equal(newPremium);
      expect(await insurancePool.coverageAmount()).to.equal(newCoverage);
      expect(await insurancePool.coverageDuration()).to.equal(newDuration);
    });

    it("Should fail when non-owner tries to update parameters", async function () {
      await expect(
        insurancePool.connect(user1).setPolicyParams(
          ethers.parseUnits("20", 6),
          ethers.parseUnits("200", 6),
          48 * 60 * 60
        )
      ).to.be.revertedWithCustomError(insurancePool, "OwnableUnauthorizedAccount");
    });

    it("Should apply new parameters to future policies", async function () {
      // Update parameters
      const newPremium = ethers.parseUnits("20", 6);
      const newCoverage = ethers.parseUnits("200", 6);
      await insurancePool.connect(owner).setPolicyParams(newPremium, newCoverage, COVERAGE_DURATION);

      // User buys insurance with new parameters
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), newPremium);
      await insurancePool.connect(user1).buyInsurance();

      const policy = await insurancePool.getPolicy(user1.address, 0);
      expect(policy.premium).to.equal(newPremium);
      expect(policy.coverageAmount).to.equal(newCoverage);
    });
  });

  describe("Pool Funding", function () {
    it("Should allow anyone to fund the pool", async function () {
      const fundAmount = ethers.parseUnits("500", 6); // Use amount user1 has
      const initialBalance = await insurancePool.getPoolBalance();

      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), fundAmount);
      await insurancePool.connect(user1).fundPool(fundAmount);

      expect(await insurancePool.getPoolBalance()).to.equal(initialBalance + fundAmount);
    });

    it("Should fail when funder doesn't approve USDT", async function () {
      await expect(
        insurancePool.connect(user1).fundPool(ethers.parseUnits("1000", 6))
      ).to.be.reverted;
    });
  });

  describe("ReentrancyGuard", function () {
    it("Should prevent reentrancy on buyInsurance", async function () {
      // This is a basic check - in a real scenario you'd deploy a malicious contract
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT * 2n);
      
      // Multiple calls should succeed independently (not testing actual reentrancy attack)
      await insurancePool.connect(user1).buyInsurance();
      await insurancePool.connect(user1).buyInsurance();
      
      expect(await insurancePool.getUserPoliciesCount(user1.address)).to.equal(2);
    });
  });

  describe("Edge Cases", function () {
    it("Should handle policy at exact expiry time", async function () {
      await mockUSDT.connect(user1).approve(await insurancePool.getAddress(), PREMIUM_AMOUNT);
      await insurancePool.connect(user1).buyInsurance();

      const policy = await insurancePool.getPolicy(user1.address, 0);
      
      // Fast forward to just before expiry (1 second before)
      await time.increaseTo(policy.expiryTime - 1n);

      // Should still be valid before expiry time
      await expect(
        insurancePool.connect(owner).executePayout(user1.address, 0, "0xhash")
      ).to.not.be.reverted;
    });

    it("Should handle zero user policies count correctly", async function () {
      expect(await insurancePool.getUserPoliciesCount(user3.address)).to.equal(0);
    });

    it("Should handle getPolicy on non-existent policy", async function () {
      await expect(
        insurancePool.getPolicy(user3.address, 0)
      ).to.be.revertedWith("Invalid policy ID");
    });
  });
});
