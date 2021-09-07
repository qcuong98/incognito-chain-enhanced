package pdex

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/blockchain/pdex/v2utils"
	v2 "github.com/incognitochain/incognito-chain/blockchain/pdex/v2utils"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/config"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	instruction "github.com/incognitochain/incognito-chain/instruction/pdexv3"
	"github.com/incognitochain/incognito-chain/metadata"
	metadataCommon "github.com/incognitochain/incognito-chain/metadata/common"
	metadataPdexv3 "github.com/incognitochain/incognito-chain/metadata/pdexv3"
	"github.com/incognitochain/incognito-chain/utils"
	"github.com/incognitochain/incognito-chain/wallet"
)

type stateProducerV2 struct {
	stateProducerBase
}

func (sp *stateProducerV2) addLiquidity(
	txs []metadata.Transaction,
	beaconHeight uint64,
	poolPairs map[string]*PoolPairState,
	waitingContributions map[string]rawdbv2.Pdexv3Contribution,
	nftIDs map[string]uint64,
) (
	[][]string, map[string]*PoolPairState, map[string]rawdbv2.Pdexv3Contribution, error,
) {
	res := [][]string{}
	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		metaData, _ := tx.GetMetadata().(*metadataPdexv3.AddLiquidityRequest)
		incomingContribution := *NewContributionWithMetaData(*metaData, *tx.Hash(), shardID)
		incomingContributionState := *statedb.NewPdexv3ContributionStateWithValue(
			incomingContribution, metaData.PairHash(),
		)
		_, found := nftIDs[metaData.NftID()]
		if metaData.NftID() == utils.EmptyString || !found {
			refundInst, err := instruction.NewRefundAddLiquidityWithValue(incomingContributionState).StringSlice()
			if err != nil {
				return res, poolPairs, waitingContributions, err
			}
			res = append(res, refundInst)
			continue
		}
		waitingContribution, found := waitingContributions[metaData.PairHash()]
		if !found {
			waitingContributions[metaData.PairHash()] = incomingContribution
			inst, err := instruction.NewWaitingAddLiquidityWithValue(incomingContributionState).StringSlice()
			if err != nil {
				return res, poolPairs, waitingContributions, err
			}
			res = append(res, inst)
			continue
		}
		delete(waitingContributions, metaData.PairHash())
		waitingContributionState := *statedb.NewPdexv3ContributionStateWithValue(
			waitingContribution, metaData.PairHash(),
		)
		if waitingContribution.TokenID().String() == incomingContribution.TokenID().String() ||
			waitingContribution.Amplifier() != incomingContribution.Amplifier() ||
			waitingContribution.PoolPairID() != incomingContribution.PoolPairID() ||
			waitingContribution.NftID().String() != incomingContribution.NftID().String() {
			insts, err := v2utils.BuildRefundAddLiquidityInstructions(
				waitingContributionState, incomingContributionState,
			)
			if err != nil {
				return res, poolPairs, waitingContributions, err
			}
			res = append(res, insts...)
			continue
		}
		nftHash, err := common.Hash{}.NewHashFromStr(metaData.NftID())
		if err != nil {
			return res, poolPairs, waitingContributions, err
		}

		poolPairID := utils.EmptyString
		if waitingContribution.PoolPairID() == utils.EmptyString {
			poolPairID = generatePoolPairKey(waitingContribution.TokenID().String(), metaData.TokenID(), waitingContribution.TxReqID().String())
		} else {
			poolPairID = waitingContribution.PoolPairID()
		}
		rootPoolPair, found := poolPairs[poolPairID]
		if !found || rootPoolPair == nil {
			if waitingContribution.PoolPairID() == utils.EmptyString {
				newPoolPair := initPoolPairState(waitingContribution, incomingContribution)
				tempAmt := big.NewInt(0).Mul(
					big.NewInt(0).SetUint64(waitingContribution.Amount()),
					big.NewInt(0).SetUint64(incomingContribution.Amount()),
				)
				shareAmount := big.NewInt(0).Sqrt(tempAmt).Uint64()
				err = newPoolPair.addShare(
					*nftHash,
					shareAmount, beaconHeight,
					waitingContribution.TxReqID().String(),
				)
				if err != nil {
					continue
				}
				poolPairs[poolPairID] = newPoolPair
				insts, err := v2utils.BuildMatchAddLiquidityInstructions(incomingContributionState, poolPairID, *nftHash)
				if err != nil {
					return res, poolPairs, waitingContributions, err
				}
				res = append(res, insts...)
				continue
			} else {
				insts, err := v2utils.BuildRefundAddLiquidityInstructions(
					waitingContributionState, incomingContributionState,
				)
				if err != nil {
					return res, poolPairs, waitingContributions, err
				}
				res = append(res, insts...)
				continue
			}
		}
		token0Contribution, token1Contribution := rootPoolPair.getContributionsByOrder(
			&waitingContribution, &incomingContribution,
		)
		actualToken0ContributionAmount,
			returnedToken0ContributionAmount,
			actualToken1ContributionAmount,
			returnedToken1ContributionAmount := rootPoolPair.
			computeActualContributedAmounts(&token0Contribution, &token1Contribution)

		token0ContributionState := *statedb.NewPdexv3ContributionStateWithValue(
			token0Contribution, metaData.PairHash(),
		)
		token1ContributionState := *statedb.NewPdexv3ContributionStateWithValue(
			token1Contribution, metaData.PairHash(),
		)
		if actualToken0ContributionAmount == 0 || actualToken1ContributionAmount == 0 {
			insts, err := v2utils.BuildRefundAddLiquidityInstructions(
				token0ContributionState, token1ContributionState,
			)
			if err != nil {
				return res, poolPairs, waitingContributions, err
			}
			res = append(res, insts...)
			continue
		}
		//clone props gonna change before process
		poolPair := NewPoolPairState()
		poolPair.withShares(rootPoolPair.cloneShares())
		poolPair.withState(*rootPoolPair.state.Clone())
		shareAmount, err := poolPair.addReserveDataAndCalculateShare(
			token0Contribution.TokenID().String(), token1Contribution.TokenID().String(),
			actualToken0ContributionAmount, actualToken1ContributionAmount,
		)
		if err != nil {
			Logger.log.Debug("err:", err)
			continue
		}
		err = poolPair.addShare(
			*nftHash,
			shareAmount, beaconHeight,
			waitingContribution.TxReqID().String(),
		)
		if err != nil {
			Logger.log.Debug("err:", err)
			continue
		}
		insts, err := v2utils.BuildMatchAndReturnAddLiquidityInstructions(
			token0ContributionState, token1ContributionState,
			shareAmount, returnedToken0ContributionAmount,
			actualToken0ContributionAmount,
			returnedToken1ContributionAmount,
			actualToken1ContributionAmount,
			*nftHash,
		)
		if err != nil {
			return res, poolPairs, waitingContributions, err
		}
		//assign which props not change after process
		poolPair.withOrderBook(rootPoolPair.orderbook)
		poolPairs[poolPairID] = poolPair

		res = append(res, insts...)
	}
	return res, poolPairs, waitingContributions, nil
}

func (sp *stateProducerV2) mintPDEXGenesis() ([][]string, error) {
	daoPaymentAddressStr := config.Param().IncognitoDAOAddress
	keyWallet, err := wallet.Base58CheckDeserialize(daoPaymentAddressStr)
	if err != nil {
		return [][]string{}, errors.New("Could not deserialize DAO payment address")
	}
	if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
		return [][]string{}, errors.New("DAO payment address is invalid")
	}

	shardID := common.GetShardIDFromLastByte(keyWallet.KeySet.PaymentAddress.Pk[common.PublicKeySize-1])

	mintingPDEXGenesisContent := metadataPdexv3.MintPDEXGenesisContent{
		MintingPaymentAddress: daoPaymentAddressStr,
		MintingAmount:         uint64(GenesisMintingAmount * math.Pow(10, common.PDEXDenominatingDecimal)),
		ShardID:               shardID,
	}
	mintingPDEXGenesisContentBytes, _ := json.Marshal(mintingPDEXGenesisContent)

	inst := []string{
		strconv.Itoa(metadataCommon.Pdexv3MintPDEXGenesisMeta),
		strconv.Itoa(int(shardID)),
		metadataPdexv3.RequestAcceptedChainStatus,
		string(mintingPDEXGenesisContentBytes),
	}

	return [][]string{inst}, nil
}

func (sp *stateProducerV2) modifyParams(
	txs []metadata.Transaction,
	beaconHeight uint64,
	params *Params,
	pairs map[string]*PoolPairState,
	stakingPools map[string]*StakingPoolState,
) ([][]string, *Params, map[string]*StakingPoolState, error) {
	instructions := [][]string{}

	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		txReqID := *tx.Hash()
		metaData, ok := tx.GetMetadata().(*metadataPdexv3.ParamsModifyingRequest)
		if !ok {
			return instructions, params, stakingPools, errors.New("Can not parse params modifying metadata")
		}

		// check conditions
		metadataParams := metaData.Pdexv3Params
		newParams := Params(metadataParams)
		isValidParams, errorMsg := isValidPdexv3Params(&newParams, pairs, stakingPools)

		status := ""
		if isValidParams {
			status = metadataPdexv3.RequestAcceptedChainStatus
			params = &newParams
			stakingPools = addStakingPoolState(stakingPools, params.StakingPoolsShare)
		} else {
			status = metadataPdexv3.RequestRejectedChainStatus
		}

		inst := v2utils.BuildModifyParamsInst(
			metadataParams,
			errorMsg,
			shardID,
			txReqID,
			status,
		)
		instructions = append(instructions, inst)
	}

	return instructions, params, stakingPools, nil
}

func (sp *stateProducerV2) mintPDEX(
	mintingAmount uint64,
	params *Params,
	pairs map[string]*PoolPairState,
) ([][]string, map[string]*PoolPairState, error) {
	instructions := [][]string{}

	totalRewardShare := uint64(0)
	for _, shareAmount := range params.PDEXRewardPoolPairsShare {
		totalRewardShare += uint64(shareAmount)
	}

	for pairID, shareRewardAmount := range params.PDEXRewardPoolPairsShare {
		pair, isExisted := pairs[pairID]
		if !isExisted {
			return instructions, pairs, fmt.Errorf("Could not find pair %v for distributing PDEX reward", pairID)
		}

		// pairReward = mintingAmount * shareRewardAmount / totalRewardShare
		pairReward := new(big.Int).Mul(new(big.Int).SetUint64(mintingAmount), new(big.Int).SetUint64(uint64(shareRewardAmount)))
		pairReward = new(big.Int).Div(pairReward, new(big.Int).SetUint64(totalRewardShare))

		if !pairReward.IsUint64() {
			return instructions, pairs, fmt.Errorf("Could not calculate PDEX reward for pair %v", pairID)
		}

		(&v2utils.TradingPair{&pair.state}).AddFee(
			common.PDEXCoinID, pairReward.Uint64(), BaseLPFeesPerShare,
			0, 0, []common.Hash{})

		instructions = append(instructions, v2utils.BuildMintPDEXInst(pairID, pairReward.Uint64())...)
	}

	return instructions, pairs, nil
}

func (sp *stateProducerV2) trade(
	txs []metadata.Transaction,
	pairs map[string]*PoolPairState,
	params *Params,
) ([][]string, map[string]*PoolPairState, error) {
	result := [][]string{}
	var invalidTxs []metadataCommon.Transaction
	var fees, sellAmounts map[string]uint64
	var feeInPRVMap map[string]bool
	var err error
	txs, feeInPRVMap, fees, sellAmounts, invalidTxs, err = getWeightedFee(txs, pairs, params)
	if err != nil {
		return result, pairs, fmt.Errorf("Error converting fee %v", err)
	}
	sort.SliceStable(txs, func(i, j int) bool {
		// compare the fee / sellAmount ratio by comparing products
		fi := big.NewInt(0).SetUint64(fees[txs[i].Hash().String()])
		fi.Mul(fi, big.NewInt(0).SetUint64(sellAmounts[txs[j].Hash().String()]))
		fj := big.NewInt(0).SetUint64(fees[txs[j].Hash().String()])
		fj.Mul(fj, big.NewInt(0).SetUint64(sellAmounts[txs[i].Hash().String()]))

		// sort descending
		return fi.Cmp(fj) == 1
	})

	for _, tx := range txs {
		currentTrade, ok := tx.GetMetadata().(*metadataPdexv3.TradeRequest)
		if !ok {
			return result, pairs, errors.New("Cannot parse trade metadata")
		}
		// sender & receiver shard must be the same
		refundInstructions, err := getRefundedTradeInstructions(currentTrade,
			feeInPRVMap[tx.Hash().String()], *tx.Hash(), byte(tx.GetValidationEnv().ShardID()))
		if err != nil {
			return result, pairs, fmt.Errorf("Error preparing trade refund %v", err)
		}

		// get relevant, cloned data from state for the trade path
		reserves, orderbookList, tradeDirections, tokenToBuy, err :=
			tradePathFromState(currentTrade.TokenToSell, currentTrade.TradePath, pairs)
		tradeOutputReceiver, exists := currentTrade.Receiver[tokenToBuy]
		// anytime the trade handler fails, add a refund instruction
		if err != nil || !exists {
			Logger.log.Warnf("Error preparing trade path: %v", err)
			result = append(result, refundInstructions...)
			continue
		}

		acceptedTradeMd, _, err := v2.MaybeAcceptTrade(
			currentTrade.SellAmount, 0, currentTrade.TradePath,
			tradeOutputReceiver, reserves, tradeDirections,
			tokenToBuy, currentTrade.MinAcceptableAmount, orderbookList,
		)
		if err != nil {
			Logger.log.Warnf("Error handling trade: %v", err)
			result = append(result, refundInstructions...)
			continue
		}

		acceptedTradeMd, err = v2.TrackFee(
			currentTrade.TradingFee, feeInPRVMap[tx.Hash().String()], BaseLPFeesPerShare,
			currentTrade.TradePath, reserves, tradeDirections, orderbookList,
			acceptedTradeMd,
			params.TradingProtocolFeePercent, params.TradingStakingPoolRewardPercent, params.StakingRewardTokens,
		)
		if err != nil {
			Logger.log.Warnf("Error handling fee distribution: %v", err)
			result = append(result, refundInstructions...)
			continue
		}

		// apply state changes for trade consistency in the same block
		for index, pairID := range currentTrade.TradePath {
			changedPair := pairs[pairID]
			changedPair.state = *reserves[index]
			orderbook, _ := orderbookList[index].(*Orderbook) // type is determined; see tradePathFromState()
			changedPair.orderbook = *orderbook
			pairs[pairID] = changedPair
		}
		// "accept" instruction
		action := instruction.NewAction(
			acceptedTradeMd,
			*tx.Hash(),
			byte(tx.GetValidationEnv().ShardID()), // sender & receiver shard must be the same
		)
		result = append(result, action.StringSlice())
	}

	// refund invalid-by-fee tradeRequests
	for _, tx := range invalidTxs {
		currentTrade, ok := tx.GetMetadata().(*metadataPdexv3.TradeRequest)
		if !ok {
			return result, pairs, fmt.Errorf("Cannot parse trade metadata")
		}
		refundInstructions, err := getRefundedTradeInstructions(currentTrade,
			feeInPRVMap[tx.Hash().String()], *tx.Hash(), byte(tx.GetValidationEnv().ShardID()))
		if err != nil {
			return result, pairs, fmt.Errorf("Error preparing trade refund %v", err)
		}
		result = append(result, refundInstructions...)
	}
	Logger.log.Warnf("Trade instructions: %v", result)
	return result, pairs, nil
}

func (sp *stateProducerV2) addOrder(
	txs []metadata.Transaction,
	pairs map[string]*PoolPairState,
	nftIDs map[string]uint64,
	params *Params,
) ([][]string, map[string]*PoolPairState, error) {
	result := [][]string{}

TransactionLoop:
	for _, tx := range txs {
		currentOrderReq, ok := tx.GetMetadata().(*metadataPdexv3.AddOrderRequest)
		if !ok {
			return result, pairs, errors.New("Cannot parse AddOrder metadata")
		}
		// sender & receiver shard must be the same
		refundInstructions, err := getRefundedAddOrderInstructions(currentOrderReq,
			*tx.Hash(), byte(tx.GetValidationEnv().ShardID()))
		if err != nil {
			return result, pairs, fmt.Errorf("Error preparing trade refund %v", err)
		}

		if _, exists := nftIDs[currentOrderReq.NftID.String()]; !exists {
			Logger.log.Warnf("Cannot find nftID %s for new order", currentOrderReq.NftID.String())
			result = append(result, refundInstructions...)
			continue TransactionLoop
		}

		pair, exists := pairs[currentOrderReq.PoolPairID]
		if !exists {
			Logger.log.Warnf("Cannot find pair %s for new order", currentOrderReq.PoolPairID)
			result = append(result, refundInstructions...)
			continue TransactionLoop
		}
		if v2.IsEmptyLiquidity(pair.state) {
			Logger.log.Warnf("No liquidity in pair %s", currentOrderReq.PoolPairID)
			result = append(result, refundInstructions...)
			continue TransactionLoop
		}

		orderID := tx.Hash().String()
		orderbook := pair.orderbook
		for _, ord := range orderbook.orders {
			if ord.Id() == orderID {
				Logger.log.Warnf("Cannot add existing order ID %s", orderID)
				// on any error, append a refund instruction & continue to next tx
				result = append(result, refundInstructions...)
				continue TransactionLoop
			}
		}

		// prepare order data
		sellAmountAfterFee := currentOrderReq.SellAmount

		var tradeDirection byte
		var token0Rate, token1Rate uint64
		var token0Balance, token1Balance uint64
		if currentOrderReq.TokenToSell == pair.state.Token0ID() {
			tradeDirection = v2.TradeDirectionSell0
			// set order's rates according to request, then set selling token's balance to sellAmount
			// and buying token to 0
			token0Rate = sellAmountAfterFee
			token1Rate = currentOrderReq.MinAcceptableAmount
			token0Balance = sellAmountAfterFee
			token1Balance = 0
		} else {
			tradeDirection = v2.TradeDirectionSell1
			token1Rate = sellAmountAfterFee
			token0Rate = currentOrderReq.MinAcceptableAmount
			token1Balance = sellAmountAfterFee
			token0Balance = 0
		}

		acceptedMd := metadataPdexv3.AcceptedAddOrder{
			PoolPairID:     currentOrderReq.PoolPairID,
			OrderID:        orderID,
			NftID:          currentOrderReq.NftID,
			Token0Rate:     token0Rate,
			Token1Rate:     token1Rate,
			Token0Balance:  token0Balance,
			Token1Balance:  token1Balance,
			TradeDirection: tradeDirection,
		}

		acceptedAction := instruction.NewAction(
			&acceptedMd,
			*tx.Hash(),
			byte(tx.GetValidationEnv().ShardID()), // sender & receiver shard must be the same
		)
		result = append(result, acceptedAction.StringSlice())
	}

	Logger.log.Warnf("AddOrder instructions: %v", result)
	return result, pairs, nil
}

func (sp *stateProducerV2) withdrawOrder(
	txs []metadata.Transaction,
	pairs map[string]*PoolPairState,
) ([][]string, map[string]*PoolPairState, error) {
	result := [][]string{}
TransactionLoop:
	for _, tx := range txs {
		currentOrderReq, ok := tx.GetMetadata().(*metadataPdexv3.WithdrawOrderRequest)
		if !ok {
			return result, pairs, errors.New("Cannot parse AddOrder metadata")
		}

		// always return NFT in response
		nftReceiver, exists := currentOrderReq.Receiver[currentOrderReq.NftID]
		if !exists {
			return result, pairs, fmt.Errorf("NFT receiver not found in WithdrawOrder Request")
		}
		withdrawOutputReceiver, exists := currentOrderReq.Receiver[currentOrderReq.TokenID]
		if !exists {
			return result, pairs, fmt.Errorf("WithdrawOrder receiver not found")
		}
		recvStr, err := nftReceiver.String()
		if err != nil {
			return result, pairs, fmt.Errorf("NFT receiver invalid in WithdrawOrder Request")
		}
		mintInstruction, err := instruction.NewMintNftWithValue(
			currentOrderReq.NftID, recvStr, byte(tx.GetValidationEnv().ShardID()), *tx.Hash(),
		).StringSlice(strconv.Itoa(metadataCommon.Pdexv3WithdrawOrderRequestMeta))
		result = append(result, mintInstruction)

		// default to reject
		currentAction := instruction.NewAction(
			&metadataPdexv3.RejectedWithdrawOrder{
				PoolPairID: currentOrderReq.PoolPairID,
				OrderID:    currentOrderReq.OrderID,
			},
			*tx.Hash(),
			byte(tx.GetValidationEnv().ShardID()), // sender & receiver shard must be the same
		)

		pair, exists := pairs[currentOrderReq.PoolPairID]
		if !exists {
			Logger.log.Warnf("Cannot find pair %s for new order", currentOrderReq.PoolPairID)
			result = append(result, currentAction.StringSlice())
			continue TransactionLoop
		}

		orderID := currentOrderReq.OrderID
		for _, ord := range pair.orderbook.orders {
			if ord.Id() == orderID {
				if ord.NftID() == currentOrderReq.NftID {
					var currentBalance uint64
					withdrawAmount := currentOrderReq.Amount
					switch currentOrderReq.TokenID {
					// cap withdrawAmount & set new balance in state
					case pair.state.Token0ID():
						currentBalance = ord.Token0Balance()
						if currentBalance < withdrawAmount {
							withdrawAmount = currentBalance
						}
						ord.SetToken0Balance(currentBalance - withdrawAmount)
					case pair.state.Token1ID():
						currentBalance = ord.Token1Balance()
						if currentBalance < withdrawAmount {
							withdrawAmount = currentBalance
						}
						ord.SetToken1Balance(currentBalance - withdrawAmount)
					default:
						Logger.log.Warnf("Invalid withdraw tokenID %v for order %s",
							currentOrderReq.TokenID, orderID)
						result = append(result, currentAction.StringSlice())
						continue TransactionLoop
					}
					// apply orderbook changes for withdraw consistency in the same block
					pairs[currentOrderReq.PoolPairID] = pair

					// "accepted" metadata
					currentAction.Content = &metadataPdexv3.AcceptedWithdrawOrder{
						PoolPairID: currentOrderReq.PoolPairID,
						OrderID:    currentOrderReq.OrderID,
						Receiver:   withdrawOutputReceiver,
						TokenID:    currentOrderReq.TokenID,
						Amount:     withdrawAmount,
					}
				} else {
					Logger.log.Warnf("Incorrect NftID %v for withdrawing order %s",
						currentOrderReq.NftID, orderID)
				}
				result = append(result, currentAction.StringSlice())
				continue TransactionLoop
			}
		}

		Logger.log.Warnf("No order with ID %s found for withdrawal", orderID)
		result = append(result, currentAction.StringSlice())
	}

	Logger.log.Warnf("WithdrawOrder instructions: %v", result)
	return result, pairs, nil
}

func (sp *stateProducerV2) withdrawLPFee(
	txs []metadata.Transaction,
	pairs map[string]*PoolPairState,
) ([][]string, map[string]*PoolPairState, error) {
	instructions := [][]string{}

	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		txReqID := *tx.Hash()
		metaData, ok := tx.GetMetadata().(*metadataPdexv3.WithdrawalLPFeeRequest)
		if !ok {
			return instructions, pairs, errors.New("Can not parse withdrawal LP fee metadata")
		}

		_, isExisted := metaData.Receivers[metaData.NftID]
		if !isExisted {
			return instructions, pairs, fmt.Errorf("NFT receiver not found in WithdrawalLPFeeRequest")
		}
		addressStr, err := metaData.Receivers[metaData.NftID].String()
		if err != nil {
			return instructions, pairs, fmt.Errorf("NFT receiver invalid in WithdrawalLPFeeRequest")
		}
		mintNftInst := instruction.NewMintNftWithValue(metaData.NftID, addressStr, shardID, txReqID)
		mintNftInstStr, err := mintNftInst.StringSlice(strconv.Itoa(metadataCommon.Pdexv3WithdrawLPFeeRequestMeta))
		if err != nil {
			return instructions, pairs, fmt.Errorf("Can not parse mint NFT instruction")
		}

		instructions = append(instructions, mintNftInstStr)

		rejectInst := v2utils.BuildWithdrawLPFeeInsts(
			metaData.PoolPairID,
			metaData.NftID,
			map[common.Hash]metadataPdexv3.ReceiverInfo{},
			shardID,
			txReqID,
			metadataPdexv3.RequestRejectedChainStatus,
		)

		// check conditions
		poolPair, isExisted := pairs[metaData.PoolPairID]
		if !isExisted {
			instructions = append(instructions, rejectInst...)
			continue
		}

		share, isExisted := poolPair.shares[metaData.NftID.String()]
		if !isExisted {
			instructions = append(instructions, rejectInst...)
			continue
		}

		// compute amount of received LP fee
		reward, err := poolPair.RecomputeLPFee(metaData.NftID)
		if err != nil {
			return instructions, pairs, fmt.Errorf("Could not track LP reward: %v\n", err)
		}

		receiversInfo := map[common.Hash]metadataPdexv3.ReceiverInfo{}
		notEnoughOTA := false
		for tokenID := range reward {
			if _, isExisted := metaData.Receivers[tokenID]; !isExisted {
				notEnoughOTA = true
				break
			}
			receiversInfo[tokenID] = metadataPdexv3.ReceiverInfo{
				Address: metaData.Receivers[tokenID],
				Amount:  reward[tokenID],
			}
		}
		if notEnoughOTA {
			instructions = append(instructions, rejectInst...)
			continue
		}

		acceptedInst := v2utils.BuildWithdrawLPFeeInsts(
			metaData.PoolPairID,
			metaData.NftID,
			receiversInfo,
			shardID,
			txReqID,
			metadataPdexv3.RequestAcceptedChainStatus,
		)

		// update state after fee withdrawal
		share.tradingFees = map[common.Hash]uint64{}
		share.lastLPFeesPerShare = map[common.Hash]*big.Int{}
		for tokenID, value := range poolPair.state.LPFeesPerShare() {
			share.lastLPFeesPerShare[tokenID] = new(big.Int).Set(value)
		}

		instructions = append(instructions, acceptedInst...)
	}

	return instructions, pairs, nil
}

func (sp *stateProducerV2) withdrawProtocolFee(
	txs []metadata.Transaction,
	pairs map[string]*PoolPairState,
) ([][]string, map[string]*PoolPairState, error) {
	instructions := [][]string{}

	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		txReqID := *tx.Hash()
		metaData, ok := tx.GetMetadata().(*metadataPdexv3.WithdrawalProtocolFeeRequest)
		if !ok {
			return instructions, pairs, errors.New("Can not parse withdrawal protocol fee metadata")
		}

		rejectInst := v2utils.BuildWithdrawProtocolFeeInsts(
			metaData.PoolPairID,
			map[common.Hash]metadataPdexv3.ReceiverInfo{},
			shardID,
			txReqID,
			metadataPdexv3.RequestRejectedChainStatus,
		)

		// check conditions
		pair, isExisted := pairs[metaData.PoolPairID]
		if !isExisted {
			instructions = append(instructions, rejectInst...)
			continue
		}

		reward := pair.state.ProtocolFees()

		receiversInfo := map[common.Hash]metadataPdexv3.ReceiverInfo{}
		for tokenID := range reward {
			if _, isExisted := metaData.Receivers[tokenID]; !isExisted {
				return instructions, pairs, fmt.Errorf("Could not find receiver for token %v\n", tokenID)
			}
			receiversInfo[tokenID] = metadataPdexv3.ReceiverInfo{
				Address: metaData.Receivers[tokenID],
				Amount:  reward[tokenID],
			}
		}

		acceptedInst := v2utils.BuildWithdrawProtocolFeeInsts(
			metaData.PoolPairID,
			receiversInfo,
			shardID,
			txReqID,
			metadataPdexv3.RequestAcceptedChainStatus,
		)

		// update state after fee withdrawal
		pair.state.SetProtocolFees(map[common.Hash]uint64{})

		instructions = append(instructions, acceptedInst...)
	}

	return instructions, pairs, nil
}

func (sp *stateProducerV2) withdrawLiquidity(
	txs []metadata.Transaction, poolPairs map[string]*PoolPairState, nftIDs map[string]uint64,
	beaconHeight uint64,
) (
	[][]string,
	map[string]*PoolPairState,
	error,
) {
	res := [][]string{}
	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		metaData, _ := tx.GetMetadata().(*metadataPdexv3.WithdrawLiquidityRequest)
		txReqID := *tx.Hash()

		rejectInsts, err := v2utils.BuildRejectWithdrawLiquidityInstructions(*metaData, txReqID, shardID)
		if err != nil {
			return res, poolPairs, err
		}

		_, found := nftIDs[metaData.NftID()]
		if metaData.NftID() == utils.EmptyString || !found {
			res = append(res, rejectInsts...)
			continue
		}
		rootPoolPair, ok := poolPairs[metaData.PoolPairID()]
		if !ok || rootPoolPair == nil {
			res = append(res, rejectInsts...)
			continue
		}
		shares, ok := rootPoolPair.shares[metaData.NftID()]
		if !ok || shares == nil {
			res = append(res, rejectInsts...)
			continue
		}
		//clone props gonna change before process
		poolPair := NewPoolPairState()
		poolPair.withShares(rootPoolPair.cloneShares())
		poolPair.withState(*rootPoolPair.state.Clone())

		token0Amount, token1Amount, shareAmount, err := poolPair.deductShare(
			metaData.NftID(), metaData.ShareAmount(), beaconHeight,
		)
		if err != nil {
			res = append(res, rejectInsts...)
			continue
		}

		insts, err := v2utils.BuildAcceptWithdrawLiquidityInstructions(
			*metaData,
			poolPair.state.Token0ID(), poolPair.state.Token1ID(),
			token0Amount, token1Amount, shareAmount,
			txReqID, shardID)
		if err != nil {
			return res, poolPairs, err
		}
		res = append(res, insts...)
		//assign which props not change after process
		poolPair.withOrderBook(rootPoolPair.orderbook)
		poolPairs[metaData.PoolPairID()] = poolPair
	}
	return res, poolPairs, nil
}

func (sp *stateProducerV2) userMintNft(
	txs []metadata.Transaction,
	nftIDs map[string]uint64,
	beaconHeight, mintNftRequireAmount uint64,
) ([][]string, map[string]uint64, error) {
	res := [][]string{}
	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		metaData, _ := tx.GetMetadata().(*metadataPdexv3.UserMintNftRequest)
		txReqID := *tx.Hash()
		inst := []string{}
		var err error
		if metaData.Amount() != mintNftRequireAmount {
			inst, err = instruction.NewRejectUserMintNftWithValue(
				metaData.OtaReceiver(), metaData.Amount(), shardID, txReqID,
			).StringSlice()
			if err != nil {
				Logger.log.Debugf("Can not reject mint nftID with txHash %s", txReqID.String())
				continue
			}
		} else {
			nftID := genNFT(uint64(len(nftIDs)), beaconHeight)
			nftIDs[nftID.String()] = metaData.Amount()
			inst, err = instruction.NewAcceptUserMintNftWithValue(
				metaData.OtaReceiver(), metaData.Amount(), shardID, nftID, txReqID,
			).StringSlice()
			if err != nil {
				Logger.log.Debugf("Can not mint nftID with txHash %s", txReqID.String())
				continue
			}
		}
		res = append(res, inst)
	}
	return res, nftIDs, nil
}

func (sp *stateProducerV2) staking(
	txs []metadata.Transaction,
	nftIDs map[string]uint64,
	stakingPoolStates map[string]*StakingPoolState,
	beaconHeight uint64,
) ([][]string, map[string]*StakingPoolState, error) {
	res := [][]string{}
	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		metaData, _ := tx.GetMetadata().(*metadataPdexv3.StakingRequest)
		txReqID := *tx.Hash()
		stakingTokenHash, err := common.Hash{}.NewHashFromStr(metaData.TokenID())
		if err != nil {
			Logger.log.Infof("tx hash %s error %v", txReqID, err)
			continue
		}
		nftHash, err := common.Hash{}.NewHashFromStr(metaData.NftID())
		if err != nil {
			Logger.log.Infof("tx hash %s error %v", txReqID, err)
			continue
		}
		rootStakingPoolState, found := stakingPoolStates[metaData.TokenID()]
		if !found || rootStakingPoolState == nil {
			rejectInst, err := instruction.NewRejectStakingWithValue(
				metaData.OtaReceiver(), *stakingTokenHash, txReqID, shardID, metaData.TokenAmount(),
			).StringSlice()
			if err != nil {
				Logger.log.Infof("tx hash %s error %v", txReqID, err)
				return res, stakingPoolStates, err
			}
			res = append(res, rejectInst)
			continue
		}
		_, found = nftIDs[metaData.NftID()]
		if metaData.NftID() == utils.EmptyString || !found {
			rejectInst, err := instruction.NewRejectStakingWithValue(
				metaData.OtaReceiver(), *stakingTokenHash, txReqID, shardID, metaData.TokenAmount(),
			).StringSlice()
			if err != nil {
				Logger.log.Infof("tx hash %s error %v", txReqID, err)
				return res, stakingPoolStates, err
			}
			res = append(res, rejectInst)
			continue
		}
		stakingPoolState := rootStakingPoolState.Clone()
		err = stakingPoolState.addLiquidity(metaData.NftID(), metaData.TokenAmount(), beaconHeight)
		if err != nil {
			rejectInst, err := instruction.NewRejectStakingWithValue(
				metaData.OtaReceiver(), *stakingTokenHash, txReqID, shardID, metaData.TokenAmount(),
			).StringSlice()
			if err != nil {
				Logger.log.Infof("tx hash %s error %v", txReqID, err)
				return res, stakingPoolStates, err
			}
			res = append(res, rejectInst)
			continue
		}
		inst, err := instruction.NewAcceptStakingWtihValue(
			*nftHash, *stakingTokenHash, txReqID, shardID, metaData.TokenAmount(),
		).StringSlice()
		if err != nil {
			return res, stakingPoolStates, err
		}
		res = append(res, inst)
		stakingPoolStates[metaData.TokenID()] = stakingPoolState
	}
	return res, stakingPoolStates, nil
}
