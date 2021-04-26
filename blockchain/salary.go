package blockchain

import (
	"fmt"
	"strconv"

	"github.com/incognitochain/incognito-chain/blockchain/committeestate"

	"github.com/incognitochain/incognito-chain/blockchain/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
)

func (blockchain *BlockChain) addShardRewardRequestToBeacon(beaconBlock *types.BeaconBlock, rewardStateDB *statedb.StateDB) error {
	for _, inst := range beaconBlock.Body.Instructions {
		if len(inst) <= 2 {
			continue
		}
		if instruction.IsConsensusInstruction(inst[0]) || inst[0] == instruction.SHARD_RECEIVE_REWARD_V3_ACTION {
			continue
		}
		if inst[0] == instruction.ACCEPT_BLOCK_REWARD_V3_ACTION {
			acceptBlockRewardIns, err := instruction.ValidateAndImportAcceptBlockRewardV3InstructionFromString(inst)
			if err != nil {
				return err
			}
			acceptBlockRewardIns.TxsFee()[common.PRVCoinID] += blockchain.getRewardAmount(acceptBlockRewardIns.ShardBlockHeight())

			for key, value := range acceptBlockRewardIns.TxsFee() {
				if value != 0 {
					err = statedb.AddShardRewardRequestV3(
						rewardStateDB,
						beaconBlock.Header.Epoch,
						acceptBlockRewardIns.ShardID(),
						acceptBlockRewardIns.SubsetID(),
						key, value)
					if err != nil {
						return err
					}
				}
			}
			continue
		}
		metaType, err := strconv.Atoi(inst[0])
		if err != nil {
			continue
		}
		if metaType == instruction.ACCEPT_BLOCK_REWARD_V1_ACTION {
			acceptedBlkRewardInfo, err := instruction.NewAcceptedBlockRewardV1FromString(inst[2])
			if err != nil {
				return err
			}

			if acceptedBlkRewardInfo.TxsFee == nil {
				acceptedBlkRewardInfo.TxsFee = map[common.Hash]uint64{}
			}
			acceptedBlkRewardInfo.TxsFee[common.PRVCoinID] += blockchain.getRewardAmount(acceptedBlkRewardInfo.ShardBlockHeight)

			for key, value := range acceptedBlkRewardInfo.TxsFee {
				if value != 0 {
					err = statedb.AddShardRewardRequest(
						rewardStateDB,
						beaconBlock.Header.Epoch,
						acceptedBlkRewardInfo.ShardID,
						key, value)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (blockchain *BlockChain) processSalaryInstructions(
	rewardStateDB *statedb.StateDB,
	beaconBlocks []*types.BeaconBlock,
	shardID byte,
) error {
	cInfos := make(map[int][]*statedb.StakerInfo)
	isInit := false
	epoch := uint64(0)
	for _, beaconBlock := range beaconBlocks {
		for _, l := range beaconBlock.Body.Instructions {
			if len(l) <= 2 {
				continue
			}
			if instruction.IsConsensusInstruction(l[0]) {
				continue
			}
			if l[0] == instruction.SHARD_RECEIVE_REWARD_V3_ACTION {
				shardReceiveRewardV3, err := instruction.ValidateAndImportShardReceiveRewardV3InstructionFromString(l)
				if err != nil {
					Logger.log.Debug(err)
					continue
				}
				if shardReceiveRewardV3.Epoch() != 0 {
					height := blockchain.GetLastBeaconHeightInEpoch(shardReceiveRewardV3.Epoch())
					var beaconConsensusRootHash common.Hash
					beaconConsensusRootHash, err = blockchain.GetBeaconConsensusRootHash(blockchain.GetBeaconBestState(), height)
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, fmt.Errorf("Beacon Consensus Root Hash of Height %+v not found ,error %+v", height, err))
					}
					beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, err)
					}
					cInfos = statedb.GetAllCommitteeStakeInfo(beaconConsensusStateDB, blockchain.GetShardIDs())
				}
				shardSubsetStakerInfo := getCommitteeToPayRewardV3(cInfos[int(shardReceiveRewardV3.ShardID())], shardReceiveRewardV3)
				err = blockchain.addShardCommitteeReward(rewardStateDB, shardID, shardReceiveRewardV3.Reward(), shardSubsetStakerInfo)
				if err != nil {
					return err
				}
				continue
			}
			shardToProcess, err := strconv.Atoi(l[1])
			if err != nil {
				continue
			}
			instType, err := strconv.Atoi(l[0])
			if err != nil {
				return NewBlockChainError(ProcessSalaryInstructionsError, err)
			}
			if shardToProcess == int(shardID) {
				switch instType {
				case metadata.BeaconRewardRequestMeta:
					beaconBlkRewardInfo, err := metadata.NewBeaconBlockRewardInfoFromStr(l[3])
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, err)
					}
					for key := range beaconBlkRewardInfo.BeaconReward {
						Logger.log.Criticalf("Add Committee Reward BeaconReward, Public Key %+v, reward %+v, token %+v", beaconBlkRewardInfo.PayToPublicKey, beaconBlkRewardInfo.BeaconReward[key], key)
						err = statedb.AddCommitteeReward(rewardStateDB, beaconBlkRewardInfo.PayToPublicKey, beaconBlkRewardInfo.BeaconReward[key], key)
						if err != nil {
							return NewBlockChainError(ProcessSalaryInstructionsError, err)
						}
					}
					continue

				case metadata.IncDAORewardRequestMeta:
					incDAORewardInfo, err := metadata.NewIncDAORewardInfoFromStr(l[3])
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, err)
					}
					keyWalletDevAccount, err := wallet.Base58CheckDeserialize(blockchain.config.ChainParams.IncognitoDAOAddress)
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, err)
					}
					for key := range incDAORewardInfo.IncDAOReward {
						tempPublicKey := base58.Base58Check{}.Encode(keyWalletDevAccount.KeySet.PaymentAddress.Pk, common.Base58Version)
						Logger.log.Criticalf("Add Committee Reward IncDAOReward, Public Key %+v, reward %+v, token %+v", tempPublicKey, incDAORewardInfo.IncDAOReward[key], key)
						err = statedb.AddCommitteeReward(rewardStateDB, tempPublicKey, incDAORewardInfo.IncDAOReward[key], key)
						if err != nil {
							return NewBlockChainError(ProcessSalaryInstructionsError, err)
						}
					}
					continue
				}
			}
			switch instType {
			case instruction.SHARD_RECEIVE_REWARD_V1_ACTION:
				shardRewardInfo, err := instruction.NewShardReceiveRewardV1FromString(l[3])
				if err != nil {
					return NewBlockChainError(ProcessSalaryInstructionsError, err)
				}
				if (!isInit) || (epoch != shardRewardInfo.Epoch) {
					isInit = true
					height := blockchain.GetLastBeaconHeightInEpoch(shardRewardInfo.Epoch)
					var beaconConsensusRootHash common.Hash
					beaconConsensusRootHash, err = blockchain.GetBeaconConsensusRootHash(blockchain.GetBeaconBestState(), height)
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, fmt.Errorf("Beacon Consensus Root Hash of Height %+v not found ,error %+v", height, err))
					}
					beaconConsensusStateDB, err := statedb.NewWithPrefixTrie(beaconConsensusRootHash, statedb.NewDatabaseAccessWarper(blockchain.GetBeaconChainDatabase()))
					if err != nil {
						return NewBlockChainError(ProcessSalaryInstructionsError, err)
					}
					cInfos = statedb.GetAllCommitteeStakeInfo(beaconConsensusStateDB, blockchain.GetShardIDs())
				}
				err = blockchain.addShardCommitteeReward(rewardStateDB, shardID, shardRewardInfo.ShardReward, cInfos[int(shardToProcess)])
				if err != nil {
					return err
				}
				continue
			}

		}
	}
	return nil
}

func getCommitteeToPayRewardV3(
	committees []*statedb.StakerInfo,
	shardReceiveRewardV3 *instruction.ShardReceiveRewardV3,
) []*statedb.StakerInfo {
	res := []*statedb.StakerInfo{}
	for i, v := range committees {
		if i%MaxSubsetCommittees == int(shardReceiveRewardV3.SubsetID()) {
			res = append(res, v)
		}
	}
	return res
}

func (blockchain *BlockChain) addShardCommitteeReward(
	rewardStateDB *statedb.StateDB,
	shardID byte,
	reward map[common.Hash]uint64,
	cStakeInfos []*statedb.StakerInfo,
) (
	err error,
) {
	committeeSize := len(cStakeInfos)
	for _, candidate := range cStakeInfos {
		if common.GetShardIDFromLastByte(candidate.RewardReceiver().Pk[common.PublicKeySize-1]) == shardID {
			for key, value := range reward {
				tempPK := base58.Base58Check{}.Encode(candidate.RewardReceiver().Pk, common.Base58Version)
				Logger.log.Criticalf("Add Committee Reward ShardCommitteeReward, Public Key %+v, reward %+v, token %+v", tempPK, value/uint64(committeeSize), key)
				err = statedb.AddCommitteeReward(rewardStateDB, tempPK, value/uint64(committeeSize), key)
				if err != nil {
					return NewBlockChainError(ProcessSalaryInstructionsError, err)
				}
			}
		}
	}
	return nil
}

func calculateRewardV3(
	maxBeaconBlockCreation uint64,
	splitRewardRuleProcessor committeestate.SplitRewardRuleProcessor,
	numberOfActiveShards, maxSubsetsCommittee int,
	beaconHeight uint64,
	epoch uint64,
	rewardStateDB *statedb.StateDB,
	isSplitRewardForCustodian bool,
	percentCustodianRewards uint64,
) (map[common.Hash]uint64,
	[][]map[common.Hash]uint64,
	map[common.Hash]uint64,
	map[common.Hash]uint64, error,
) {
	allCoinID := statedb.GetAllTokenIDForReward(rewardStateDB, epoch)
	blocksPerYear := getNoBlkPerYear(maxBeaconBlockCreation)
	percentForIncognitoDAO := getPercentForIncognitoDAO(beaconHeight, blocksPerYear)
	totalRewardForShardSubset := make([][]map[common.Hash]uint64, numberOfActiveShards)
	totalRewards := make([][]map[common.Hash]uint64, numberOfActiveShards)
	totalRewardForBeacon := map[common.Hash]uint64{}
	totalRewardForIncDAO := map[common.Hash]uint64{}
	totalRewardForCustodian := map[common.Hash]uint64{}
	var err error

	for shardID := 0; shardID < numberOfActiveShards; shardID++ {
		totalRewardForShardSubset[shardID] = make([]map[common.Hash]uint64, maxSubsetsCommittee)
		totalRewards[shardID] = make([]map[common.Hash]uint64, maxSubsetsCommittee)
		for subsetID := 0; subsetID < maxSubsetsCommittee; subsetID++ {
			if totalRewards[shardID][subsetID] == nil {
				totalRewards[shardID][subsetID] = map[common.Hash]uint64{}
			}
			if totalRewardForShardSubset[shardID][subsetID] == nil {
				totalRewardForShardSubset[shardID][subsetID] = map[common.Hash]uint64{}
			}

			for _, coinID := range allCoinID {
				totalRewards[shardID][subsetID][coinID], err = statedb.GetRewardOfShardByEpochV3(
					rewardStateDB, epoch,
					byte(shardID), byte(subsetID), coinID)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				if totalRewards[shardID][subsetID][coinID] == 0 {
					delete(totalRewards[shardID][subsetID], coinID)
				}
			}

			env := committeestate.NewSplitRewardEnvironmentV3(
				byte(shardID),
				byte(subsetID),
				byte(maxSubsetsCommittee),
				beaconHeight,
				totalRewards[shardID][subsetID],
				isSplitRewardForCustodian,
				percentCustodianRewards,
				percentForIncognitoDAO,
			)

			rewardForBeacon, rewardForShardSubset, rewardForDAO, rewardForCustodian, err := splitRewardRuleProcessor.SplitReward(env)
			if err != nil {
				return nil, nil, nil, nil, err
			}

			plusMap(rewardForBeacon, totalRewardForBeacon)
			plusMap(rewardForShardSubset, totalRewardForShardSubset[shardID][subsetID])
			plusMap(rewardForDAO, totalRewardForIncDAO)
			plusMap(rewardForCustodian, totalRewardForCustodian)
		}
	}

	return totalRewardForBeacon, totalRewardForShardSubset, totalRewardForIncDAO, totalRewardForCustodian, nil
}

func calculateReward(
	maxBeaconBlockCreation uint64,
	splitRewardRuleProcessor committeestate.SplitRewardRuleProcessor,
	numberOfActiveShards int,
	beaconHeight uint64,
	epoch uint64,
	rewardStateDB *statedb.StateDB,
	isSplitRewardForCustodian bool,
	percentCustodianRewards uint64,
) (map[common.Hash]uint64,
	[]map[common.Hash]uint64,
	map[common.Hash]uint64,
	map[common.Hash]uint64, error,
) {
	allCoinID := statedb.GetAllTokenIDForReward(rewardStateDB, epoch)
	blocksPerYear := getNoBlkPerYear(maxBeaconBlockCreation)
	percentForIncognitoDAO := getPercentForIncognitoDAO(beaconHeight, blocksPerYear)
	totalRewardForShard := make([]map[common.Hash]uint64, numberOfActiveShards)
	totalRewards := make([]map[common.Hash]uint64, numberOfActiveShards)
	totalRewardForBeacon := map[common.Hash]uint64{}
	totalRewardForIncDAO := map[common.Hash]uint64{}
	totalRewardForCustodian := map[common.Hash]uint64{}
	var err error

	for id := 0; id < numberOfActiveShards; id++ {
		if totalRewards[id] == nil {
			totalRewards[id] = map[common.Hash]uint64{}
		}
		if totalRewardForShard[id] == nil {
			totalRewardForShard[id] = map[common.Hash]uint64{}
		}

		for _, coinID := range allCoinID {
			totalRewards[id][coinID], err = statedb.GetRewardOfShardByEpoch(rewardStateDB, epoch, byte(id), coinID)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			if totalRewards[id][coinID] == 0 {
				delete(totalRewards[id], coinID)
			}
		}

		env := committeestate.NewSplitRewardEnvironmentV1(
			byte(id),
			beaconHeight,
			totalRewards[id],
			isSplitRewardForCustodian,
			percentCustodianRewards,
			percentForIncognitoDAO,
			numberOfActiveShards,
		)
		rewardForBeacon, rewardForShard, rewardForDAO, rewardForCustodian, err := splitRewardRuleProcessor.SplitReward(env)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		plusMap(rewardForBeacon, totalRewardForBeacon)
		plusMap(rewardForShard, totalRewardForShard[id])
		plusMap(rewardForDAO, totalRewardForIncDAO)
		plusMap(rewardForCustodian, totalRewardForCustodian)
	}

	return totalRewardForBeacon, totalRewardForShard, totalRewardForIncDAO, totalRewardForCustodian, nil
}

func (blockchain *BlockChain) buildRewardInstructionByEpoch(
	curView *BeaconBestState,
	blkHeight, epoch uint64,
	isSplitRewardForCustodian bool,
	percentCustodianRewards uint64,
) ([][]string, map[common.Hash]uint64, error) {

	//Declare variables
	var resInst [][]string
	var err error
	var instRewardForBeacons [][]string
	var instRewardForIncDAO [][]string
	var instRewardForShards [][]string

	beaconBestView := blockchain.BeaconChain.GetBestView().(*BeaconBestState)

	totalRewardForBeacon := make(map[common.Hash]uint64)
	totalRewardForShard := make([]map[common.Hash]uint64, beaconBestView.ActiveShards)
	totalRewardForShardSubset := make([][]map[common.Hash]uint64, beaconBestView.ActiveShards)
	totalRewardForCustodian := make(map[common.Hash]uint64)
	totalRewardForIncDAO := make(map[common.Hash]uint64)

	if curView.BeaconHeight >= blockchain.config.ChainParams.ConsensusV4Height {
		totalRewardForBeacon,
			totalRewardForShardSubset,
			totalRewardForIncDAO,
			totalRewardForCustodian,
			err = calculateRewardV3(
			uint64(blockchain.config.ChainParams.MaxBeaconBlockCreation.Seconds()),
			curView.beaconCommitteeState.(committeestate.SplitRewardRuleProcessor),
			curView.ActiveShards,
			MaxSubsetCommittees,
			blkHeight, epoch,
			curView.GetBeaconRewardStateDB(),
			isSplitRewardForCustodian, percentCustodianRewards,
		)

		instRewardForShards, err = blockchain.buildInstructionRewardForShardsV3(epoch, totalRewardForShardSubset)
		if err != nil {
			return nil, nil, err
		}

	} else {
		totalRewardForBeacon,
			totalRewardForShard,
			totalRewardForIncDAO,
			totalRewardForCustodian,
			err = calculateReward(uint64(blockchain.config.ChainParams.MaxBeaconBlockCreation.Seconds()),
			curView.beaconCommitteeState.(committeestate.SplitRewardRuleProcessor),
			curView.ActiveShards, blkHeight, epoch,
			curView.GetBeaconRewardStateDB(),
			isSplitRewardForCustodian, percentCustodianRewards,
		)

		instRewardForShards, err = blockchain.buildInstructionRewardForShards(epoch, totalRewardForShard)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(totalRewardForBeacon) > 0 {
		instRewardForBeacons, err = curView.buildInstRewardForBeacons(epoch, totalRewardForBeacon)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(totalRewardForIncDAO) > 0 {
		instRewardForIncDAO, err = blockchain.buildInstRewardForIncDAO(epoch, totalRewardForIncDAO)
		if err != nil {
			return nil, nil, err
		}
	}

	resInst = common.AppendSliceString(instRewardForBeacons, instRewardForIncDAO, instRewardForShards)
	return resInst, totalRewardForCustodian, nil
}

//buildInstRewardForBeacons create reward instruction for beacons
func (beaconBestState *BeaconBestState) buildInstRewardForBeacons(epoch uint64, totalReward map[common.Hash]uint64) ([][]string, error) {
	resInst := [][]string{}
	baseRewards := map[common.Hash]uint64{}
	for key, value := range totalReward {
		baseRewards[key] = value / uint64(len(beaconBestState.GetBeaconCommittee()))
	}
	for _, beaconpublickey := range beaconBestState.GetBeaconCommittee() {
		// indicate reward pubkey
		singleInst, err := metadata.BuildInstForBeaconReward(baseRewards, beaconpublickey.GetNormalKey())
		if err != nil {
			Logger.log.Errorf("BuildInstForBeaconReward error %+v\n Totalreward: %+v, epoch: %+v, reward: %+v\n", err, totalReward, epoch, baseRewards)
			return nil, err
		}
		resInst = append(resInst, singleInst)
	}
	return resInst, nil
}

func (blockchain *BlockChain) buildInstRewardForIncDAO(epoch uint64, totalReward map[common.Hash]uint64) ([][]string, error) {
	resInst := [][]string{}
	devRewardInst, err := metadata.BuildInstForIncDAOReward(totalReward, blockchain.config.ChainParams.IncognitoDAOAddress)
	if err != nil {
		Logger.log.Errorf("buildInstRewardForIncDAO error %+v\n Totalreward: %+v, epoch: %+v\n", err, totalReward, epoch)
		return nil, err
	}
	resInst = append(resInst, devRewardInst)
	return resInst, nil
}

func (blockchain *BlockChain) buildInstructionRewardForShardsV3(epoch uint64, totalRewards [][]map[common.Hash]uint64) ([][]string, error) {
	resInst := [][]string{}

	for shardID, v := range totalRewards {
		for subsetID, reward := range v {
			if len(reward) > 0 {
				shardSubsetReward := instruction.NewShardReceiveRewardV3WithValue(reward, epoch, byte(shardID), byte(subsetID))
				shardSubsetRewardInst := shardSubsetReward.String()
				resInst = append(resInst, shardSubsetRewardInst)
			}
		}
	}

	return resInst, nil
}

func (blockchain *BlockChain) buildInstructionRewardForShards(epoch uint64, totalRewards []map[common.Hash]uint64) ([][]string, error) {
	resInst := [][]string{}
	for i, reward := range totalRewards {
		if len(reward) > 0 {
			shardRewardInst, err := instruction.NewShardReceiveRewardV1WithValue(reward, epoch, byte(i))
			if err != nil {
				Logger.log.Errorf("BuildInstForShardReward error %+v\n Totalreward: %+v, epoch: %+v\n; shard:%+v", err, reward, epoch, byte(i))
				return nil, err
			}
			resInst = append(resInst, shardRewardInst...)
		}
	}
	return resInst, nil
}

func (blockchain *BlockChain) buildWithDrawTransactionResponse(view *ShardBestState, txRequest *metadata.Transaction, blkProducerPrivateKey *privacy.PrivateKey, shardID byte) (metadata.Transaction, error) {
	if (*txRequest).GetMetadataType() != metadata.WithDrawRewardRequestMeta {
		return nil, errors.New("Can not understand this request!")
	}
	requestDetail := (*txRequest).GetMetadata().(*metadata.WithDrawRewardRequest)
	tempPublicKey := base58.Base58Check{}.Encode(requestDetail.PaymentAddress.Pk, common.Base58Version)
	amount, err := statedb.GetCommitteeReward(blockchain.GetBestStateShardRewardStateDB(shardID), tempPublicKey, requestDetail.TokenID)
	if (amount == 0) || (err != nil) {
		return nil, errors.New("Not enough reward")
	}
	responseMeta, err := metadata.NewWithDrawRewardResponse(requestDetail, (*txRequest).Hash())
	if err != nil {
		return nil, err
	}
	return blockchain.InitTxSalaryByCoinID(
		&requestDetail.PaymentAddress,
		amount,
		blkProducerPrivateKey,
		view.GetCopiedTransactionStateDB(),
		blockchain.GetBeaconBestState().GetBeaconFeatureStateDB(),
		responseMeta,
		requestDetail.TokenID,
		common.GetShardIDFromLastByte(requestDetail.PaymentAddress.Pk[common.PublicKeySize-1]))
}

func (blockchain *BlockChain) getRewardAmount(blkHeight uint64) uint64 {
	blockBeaconInterval := blockchain.config.ChainParams.MinBeaconBlockInterval.Seconds()
	blockInYear := getNoBlkPerYear(uint64(blockBeaconInterval))
	n := (blkHeight - 1) / blockInYear
	reward := uint64(blockchain.config.ChainParams.BasicReward)
	for ; n > 0; n-- {
		reward *= 91
		reward /= 100
	}
	return reward
}

func getNoBlkPerYear(blockCreationTimeSeconds uint64) uint64 {
	return (365.25 * 24 * 60 * 60) / blockCreationTimeSeconds
}

func getPercentForIncognitoDAO(blockHeight, blkPerYear uint64) int {
	year := (blockHeight - 1) / blkPerYear
	if year > (UpperBoundPercentForIncDAO - LowerBoundPercentForIncDAO) {
		return LowerBoundPercentForIncDAO
	} else {
		return UpperBoundPercentForIncDAO - int(year)
	}
}

// plusMap(src, dst): dst = dst + src
func plusMap(src, dst map[common.Hash]uint64) {
	if src != nil {
		for key, value := range src {
			dst[key] += value
		}
	}
}
