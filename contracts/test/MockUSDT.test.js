const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("MockUSDT", function () {
  let mockUSDT;
  let owner;
  let addr1;
  let addr2;

  // Deploy contract before each test
  beforeEach(async function () {
    [owner, addr1, addr2] = await ethers.getSigners();
    
    const MockUSDT = await ethers.getContractFactory("MockUSDT");
    mockUSDT = await MockUSDT.deploy();
    await mockUSDT.waitForDeployment();
  });

  describe("Deployment", function () {
    it("Should set correct name, symbol and decimals", async function () {
      expect(await mockUSDT.name()).to.equal("Mock USDT");
      expect(await mockUSDT.symbol()).to.equal("mUSDT");
      expect(await mockUSDT.decimals()).to.equal(6);
    });

    it("Should mint 1M tokens to deployer", async function () {
      const ownerBalance = await mockUSDT.balanceOf(owner.address);
      const expectedAmount = ethers.parseUnits("1000000", 6);
      expect(ownerBalance).to.equal(expectedAmount);
    });
  });

  describe("Minting", function () {
    it("Should allow anyone to mint tokens", async function () {
      const mintAmount = ethers.parseUnits("1000", 6);
      
      await mockUSDT.connect(addr1).mint(addr1.address, mintAmount);
      
      expect(await mockUSDT.balanceOf(addr1.address)).to.equal(mintAmount);
    });
  });

  describe("Transfers", function () {
    beforeEach(async function () {
      // Mint tokens to addr1 for testing
      await mockUSDT.mint(addr1.address, ethers.parseUnits("1000", 6));
    });

    it("Should transfer tokens correctly", async function () {
      const transferAmount = ethers.parseUnits("100", 6);
      
      await mockUSDT.connect(addr1).transfer(addr2.address, transferAmount);
      
      expect(await mockUSDT.balanceOf(addr2.address)).to.equal(transferAmount);
      expect(await mockUSDT.balanceOf(addr1.address)).to.equal(
        ethers.parseUnits("900", 6)
      );
    });

    it("Should fail when transferring more than balance", async function () {
      const excessiveAmount = ethers.parseUnits("2000", 6);
      
      await expect(
        mockUSDT.connect(addr1).transfer(addr2.address, excessiveAmount)
      ).to.be.reverted;
    });
  });

  describe("Approvals", function () {
    beforeEach(async function () {
      await mockUSDT.mint(addr1.address, ethers.parseUnits("1000", 6));
    });

    it("Should approve and transferFrom correctly", async function () {
      const approveAmount = ethers.parseUnits("500", 6);
      const transferAmount = ethers.parseUnits("200", 6);
      
      await mockUSDT.connect(addr1).approve(addr2.address, approveAmount);
      await mockUSDT.connect(addr2).transferFrom(
        addr1.address,
        addr2.address,
        transferAmount
      );
      
      expect(await mockUSDT.balanceOf(addr2.address)).to.equal(transferAmount);
      expect(await mockUSDT.allowance(addr1.address, addr2.address)).to.equal(
        approveAmount - transferAmount
      );
    });
  });
});
