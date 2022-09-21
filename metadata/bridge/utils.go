package bridge

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	rCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/config"
	"github.com/incognitochain/incognito-chain/dataaccessobject/rawdbv2"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	metadataCommon "github.com/incognitochain/incognito-chain/metadata/common"
	"github.com/incognitochain/incognito-chain/metadata/evmcaller"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/key"
	"github.com/incognitochain/incognito-chain/utils"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/pkg/errors"
)

type EVMProof struct {
	BlockHash rCommon.Hash `json:"BlockHash"`
	TxIndex   uint         `json:"TxIndex"`
	Proof     []string     `json:"Proof"`
}

type EVMInfo struct {
	ContractAddress   string
	Prefix            string
	IsTxHashIssued    func(stateDB *statedb.StateDB, uniqTx []byte) (bool, error)
	ListTxUsedInBlock [][]byte
}

func GetEVMInfoByNetworkID(networkID uint8, ac *metadataCommon.AccumulatedValues) (*EVMInfo, error) {
	res := &EVMInfo{}
	switch networkID {
	case common.ETHNetworkID:
		res.ListTxUsedInBlock = ac.UniqETHTxsUsed
		res.ContractAddress = config.Param().EthContractAddressStr
		res.Prefix = utils.EmptyString
		res.IsTxHashIssued = statedb.IsETHTxHashIssued
	case common.BSCNetworkID:
		res.ListTxUsedInBlock = ac.UniqBSCTxsUsed
		res.ContractAddress = config.Param().BscContractAddressStr
		res.Prefix = common.BSCPrefix
		res.IsTxHashIssued = statedb.IsBSCTxHashIssued
	case common.PLGNetworkID:
		res.ListTxUsedInBlock = ac.UniqPLGTxsUsed
		res.ContractAddress = config.Param().PlgContractAddressStr
		res.Prefix = common.PLGPrefix
		res.IsTxHashIssued = statedb.IsPLGTxHashIssued
	case common.FTMNetworkID:
		res.ListTxUsedInBlock = ac.UniqFTMTxsUsed
		res.ContractAddress = config.Param().FtmContractAddressStr
		res.Prefix = common.FTMPrefix
		res.IsTxHashIssued = statedb.IsFTMTxHashIssued
	default:
		return nil, errors.New("Invalid networkID")
	}
	return res, nil
}

func IsBridgeAggMetaType(metaType int) bool {
	switch metaType {
	case metadataCommon.BridgeAggModifyParamMeta:
		return true
	case metadataCommon.BridgeAggConvertTokenToUnifiedTokenRequestMeta:
		return true
	case metadataCommon.BridgeAggConvertTokenToUnifiedTokenResponseMeta:
		return true
	case metadataCommon.IssuingUnifiedTokenRequestMeta:
		return true
	case metadataCommon.IssuingUnifiedTokenResponseMeta:
		return true
	case metadataCommon.BurningUnifiedTokenRequestMeta:
		return true
	case metadataCommon.BurningUnifiedTokenResponseMeta:
		return true
	case metadataCommon.BridgeAggAddTokenMeta:
		return true
	case metadataCommon.BurnForCallConfirmMeta, metadataCommon.BurnForCallRequestMeta, metadataCommon.BurnForCallResponseMeta:
		return true 
	case metadataCommon.IssuingReshieldResponseMeta:
		return true
	default:
		return false
	}
}

func IsBridgeTxHashUsedInBlock(uniqTx []byte, uniqTxsUsed [][]byte) bool {
	for _, item := range uniqTxsUsed {
		if bytes.Equal(uniqTx, item) {
			return true
		}
	}
	return false
}

func GetShardIDFromPaymentAddress(paymentAddress key.PaymentAddress) (byte, error) {
	// calculate shard ID
	lastByte := paymentAddress.Pk[len(paymentAddress.Pk)-1]
	shardID := common.GetShardIDFromLastByte(lastByte)
	return shardID, nil
}

func GetShardIDFromPaymentAddressStr(addressStr string) (byte, error) {
	keyWallet, err := wallet.Base58CheckDeserialize(addressStr)
	if err != nil {
		return byte(0), err
	}
	if len(keyWallet.KeySet.PaymentAddress.Pk) == 0 {
		return byte(0), errors.New("Payment address' public key must not be empty")
	}
	// calculate shard ID
	return GetShardIDFromPaymentAddress(keyWallet.KeySet.PaymentAddress)
}

func ExtractIssueEVMDataFromReceipt(
	txReceipt *types.Receipt,
	contractAddress string, prefix string,
	expectedIncAddrStr string,
) (*big.Int, string, []byte, error) {
	if txReceipt == nil {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: bridge tx receipt is null")
	}

	logMap, err := PickAndParseLogMapFromReceiptByContractAddr(txReceipt, contractAddress, "Deposit")
	if err != nil {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: an error occurred while parsing log map from receipt: %v", err)
	}
	if logMap == nil {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: could not find log map out from receipt")
	}

	logMapBytes, _ := json.Marshal(logMap)
	metadataCommon.Logger.Log.Warn("INFO: eth logMap json - ", string(logMapBytes))

	// the token might be ETH/ERC20 BNB/BEP20
	tokenAddr, ok := logMap["token"].(rCommon.Address)
	if !ok {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: could not parse evm token id from log map.")
	}
	extTokenID := append([]byte(prefix), tokenAddr.Bytes()...)

	incAddrStr, ok := logMap["incognitoAddress"].(string)
	if !ok {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: could not parse incognito address from bridge log map.")
	}
	if expectedIncAddrStr != "" && incAddrStr != expectedIncAddrStr {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: different incognito address from bridge log map.")
	}
	if expectedIncAddrStr == "" {
		key, err := wallet.Base58CheckDeserialize(incAddrStr)
		if err != nil || len(key.KeySet.PaymentAddress.Pk) == 0 {
			return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: invalid incognito address from bridge log map.")
		}
	}

	amt, ok := logMap["amount"].(*big.Int)
	if !ok {
		return nil, utils.EmptyString, nil, fmt.Errorf("WARNING: could not parse amount from bridge log map.")
	}
	return amt, incAddrStr, extTokenID, nil
}

type DepositEventData struct {
	Amount          *big.Int
	ReceiverStr     string
	ExternalTokenID []byte
	IncTxID         []byte
	ShardID         byte
	IsOneTime       bool
}

func ExtractRedepositEVMDataFromReceipt(
	txReceipt *types.Receipt,
	contractAddress string, prefix string,
) ([]DepositEventData, error) {
	if txReceipt == nil {
		return nil, fmt.Errorf("bridge tx receipt is null")
	}
	if len(txReceipt.Logs) == 0 {
		metadataCommon.Logger.Log.Errorf("WARNING: LOG data is invalid.")
		return nil, nil
	}

	var result []DepositEventData
	for _, log := range txReceipt.Logs {
		if log, err := ParseEVMLogDataByEventName(log, contractAddress, "Redeposit"); err != nil {
			metadataCommon.Logger.Log.Errorf("WARNING: try parse Redeposit event - %v", err)
		} else {
			logMapBytes, _ := json.Marshal(log)
			metadataCommon.Logger.Log.Warn("INFO: eth redeposit log json - ", string(logMapBytes))

			// the token might be ETH/ERC20 BNB/BEP20
			tokenAddr, ok := log["token"].(rCommon.Address)
			if !ok {
				return nil, fmt.Errorf("could not parse evm token id from log map")
			}
			extTokenID := append([]byte(prefix), tokenAddr.Bytes()...)

			incAddr, ok := log["redepositIncAddress"].([]byte)
			if !ok {
				return nil, fmt.Errorf("could not parse incognito address from bridge log map")
			}
			var recv privacy.OTAReceiver
			err := recv.SetBytes(incAddr)
			if err != nil {
				return nil, fmt.Errorf("invalid incognito receiver from bridge reshield log map.")
			}
			incAddrStr, _ := recv.String()

			amt, ok := log["amount"].(*big.Int)
			if !ok {
				return nil, fmt.Errorf("could not parse amount from bridge log map.")
			}
			incTxID, ok := log["itx"].([32]byte)
			if !ok {
				return nil, fmt.Errorf("could not parse itx from bridge log map.")
			}

			result = append(result, DepositEventData{
				Amount:          amt,
				ReceiverStr:     incAddrStr,
				ExternalTokenID: extTokenID,
				IncTxID:         incTxID[:],
				ShardID:         recv.GetShardID(),
				IsOneTime:       true,
			})
		}
	}

	return result, nil
}

func VerifyTokenPair(
	stateDBs map[int]*statedb.StateDB,
	ac *metadataCommon.AccumulatedValues,
	incTokenID common.Hash,
	token []byte,
) error {
	canProcess, err := ac.CanProcessTokenPair(token, incTokenID)
	if err != nil {
		return fmt.Errorf("WARNING: an error occurred while checking it can process for token pair on the current block or not: %v", err)
	}
	if !canProcess {
		return fmt.Errorf("WARNING: pair of incognito token id & bridge's id is invalid in current block")
	}
	privacyTokenExisted, err := statedb.CheckTokenIDExisted(stateDBs, incTokenID)
	if err != nil {
		return fmt.Errorf("WARNING: Cannot find tokenID %s", incTokenID.String())
	}
	isValid, err := statedb.CanProcessTokenPair(stateDBs[common.BeaconChainID], token, incTokenID, privacyTokenExisted)
	if err != nil {
		return fmt.Errorf("WARNING: an error occured while checking it can process for token pair on the previous blocks or not: %v", err)
	}
	if !isValid {
		return fmt.Errorf("WARNING: pair of incognito token id %s & bridge's id %x is invalid with previous blocks", incTokenID, token)
	}
	return nil
}

func FindExternalTokenID(stateDB *statedb.StateDB, incTokenID common.Hash, prefix string) ([]byte, error) {
	// Convert to external tokenID
	tokenID, err := findExternalTokenID(stateDB, &incTokenID)
	if err != nil {
		return nil, err
	}

	if len(tokenID) < common.ExternalBridgeTokenLength {
		return nil, errors.New("invalid external token id")
	}

	prefixLen := len(prefix)
	if (prefixLen > 0 && !bytes.Equal([]byte(prefix), tokenID[:prefixLen])) || len(tokenID) != (common.ExternalBridgeTokenLength+prefixLen) {
		return nil, errors.New(fmt.Sprintf("invalid prefix in external tokenID %v", tokenID))
	}
	return tokenID, nil
}

// TrimNetworkPrefix is a helper that removes the 3-byte network prefix from the full 23-byte external address (for burning confirm etc.); 
// within the bridgeagg vault we only use prefixed addresses
func TrimNetworkPrefix(fullTokenID []byte, prefix string) ([]byte, error) {
	if !bytes.HasPrefix(fullTokenID, []byte(prefix)) {
		return nil, fmt.Errorf("invalid prefix in external tokenID %x, expect %s", fullTokenID, prefix)
	}
	result := fullTokenID[len(prefix):]
	if len(result) != common.ExternalBridgeTokenLength {
		return nil, fmt.Errorf("invalid length %d for un-prefixed external address", len(result))
	}
	return result, nil
}

// findExternalTokenID finds the external tokenID for a bridge token from database
func findExternalTokenID(stateDB *statedb.StateDB, tokenID *common.Hash) ([]byte, error) {
	allBridgeTokensBytes, err := statedb.GetAllBridgeTokens(stateDB)
	if err != nil {
		return nil, err
	}
	var allBridgeTokens []*rawdbv2.BridgeTokenInfo
	err = json.Unmarshal(allBridgeTokensBytes, &allBridgeTokens)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, token := range allBridgeTokens {
		if token.TokenID.IsEqual(tokenID) && len(token.ExternalTokenID) > 0 {
			return token.ExternalTokenID, nil
		}
	}
	return nil, errors.New("invalid tokenID")
}

func VerifyProofAndParseEVMReceipt(
	blockHash rCommon.Hash,
	txIndex uint,
	proofStrs []string,
	hosts []string,
	minEVMConfirmationBlocks int,
	networkPrefix string,
	checkEVMHarkFork bool,
) (*types.Receipt, error) {
	// get evm header result
	evmHeaderResult, err := evmcaller.GetEVMHeaderResult(blockHash, hosts, minEVMConfirmationBlocks, networkPrefix)
	if err != nil {
		metadataCommon.Logger.Log.Errorf("Can not get EVM header result - Error: %+v", err)
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt, err)
	}

	// check fork
	if evmHeaderResult.IsForked {
		metadataCommon.Logger.Log.Errorf("EVM Block hash %s is not in main chain", blockHash.String())
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt,
			fmt.Errorf("EVM Block hash %s is not in main chain", blockHash.String()))
	}

	// check min confirmation blocks
	if !evmHeaderResult.IsFinalized {
		metadataCommon.Logger.Log.Errorf("EVM block hash %s is not enough confirmation blocks %v", blockHash.String(), minEVMConfirmationBlocks)
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt,
			fmt.Errorf("EVM block hash %s is not enough confirmation blocks %v", blockHash.String(), minEVMConfirmationBlocks))
	}

	keybuf := new(bytes.Buffer)
	keybuf.Reset()
	rlp.Encode(keybuf, txIndex)

	nodeList := new(light.NodeList)
	for _, proofStr := range proofStrs {
		proofBytes, err := base64.StdEncoding.DecodeString(proofStr)
		if err != nil {
			return nil, err
		}
		nodeList.Put([]byte{}, proofBytes)
	}
	proof := nodeList.NodeSet()
	val, _, err := trie.VerifyProof(evmHeaderResult.Header.ReceiptHash, keybuf.Bytes(), proof)
	if err != nil {
		errMsg := fmt.Sprintf("WARNING: EVM issuance proof verification failed: %v", err)
		metadataCommon.Logger.Log.Warn(errMsg)
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt, err)
	}

	// if iReq.Type == IssuingETHRequestMeta || iReq.Type == IssuingPRVERC20RequestMeta || iReq.Type == IssuingPLGRequestMeta {
	if checkEVMHarkFork {
		if len(val) == 0 {
			return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt, errors.New("the encoded receipt is empty"))
		}

		// hardfork london with new transaction type => 0x02 || RLP([...SenderPayload, ...SenderSignature, ...GasPayerPayload, ...GasPayerSignature])
		if val[0] == AccessListTxType || val[0] == DynamicFeeTxType {
			val = val[1:]
		}
	}

	// Decode value from VerifyProof into Receipt
	constructedReceipt := new(types.Receipt)
	err = rlp.DecodeBytes(val, constructedReceipt)
	if err != nil {
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt, err)
	}

	if constructedReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.IssuingEvmRequestVerifyProofAndParseReceipt, errors.New("The constructedReceipt's status is not success"))
	}

	return constructedReceipt, nil
}

func GetEVMInfoByMetadataType(metadataType int, networkID uint) ([]string, string, int, bool, error) {
	var hosts []string
	var networkPrefix string
	minConfirmationBlocks := metadataCommon.EVMConfirmationBlocks
	checkEVMHardFork := false
	isETHNetwork := false
	isBSCNetwork := false
	isPLGNetwork := false
	isFTMNetwork := false

	if metadataType == metadataCommon.IssuingETHRequestMeta || metadataType == metadataCommon.IssuingPRVERC20RequestMeta || (metadataType == metadataCommon.IssuingUnifiedTokenRequestMeta && networkID == common.ETHNetworkID) {
		isETHNetwork = true
	}
	if metadataType == metadataCommon.IssuingBSCRequestMeta || metadataType == metadataCommon.IssuingPRVBEP20RequestMeta || (metadataType == metadataCommon.IssuingUnifiedTokenRequestMeta && networkID == common.BSCNetworkID) {
		isBSCNetwork = true
	}
	if metadataType == metadataCommon.IssuingPLGRequestMeta || (metadataType == metadataCommon.IssuingUnifiedTokenRequestMeta && networkID == common.PLGNetworkID) {
		isPLGNetwork = true
	}
	if metadataType == metadataCommon.IssuingFantomRequestMeta || (metadataType == metadataCommon.IssuingUnifiedTokenRequestMeta && networkID == common.FTMNetworkID) {
		isFTMNetwork = true
	}

	if isBSCNetwork {
		evmParam := config.Param().BSCParam
		evmParam.GetFromEnv()
		hosts = evmParam.Host

		networkPrefix = common.BSCPrefix

	} else if isETHNetwork {
		evmParam := config.Param().GethParam
		evmParam.GetFromEnv()
		hosts = evmParam.Host

		// Ethereum network with default prefix (empty string)
		networkPrefix = ""
		checkEVMHardFork = true

	} else if isPLGNetwork {
		evmParam := config.Param().PLGParam
		evmParam.GetFromEnv()
		hosts = evmParam.Host

		minConfirmationBlocks = metadataCommon.PLGConfirmationBlocks
		networkPrefix = common.PLGPrefix
		checkEVMHardFork = true

	} else if isFTMNetwork {
		evmParam := config.Param().FTMParam
		evmParam.GetFromEnv()
		hosts = evmParam.Host

		minConfirmationBlocks = metadataCommon.FantomConfirmationBlocks
		networkPrefix = common.FTMPrefix
		checkEVMHardFork = true

	} else {
		return nil, "", 0, false, fmt.Errorf("Invalid metadata type for EVM shielding request metaType %v networkID %v", metadataType, networkID)
	}

	return hosts, networkPrefix, minConfirmationBlocks, checkEVMHardFork, nil
}

func PickAndParseLogMapFromReceiptByContractAddr(
	constructedReceipt *types.Receipt,
	ethContractAddressStr string,
	eventName string) (map[string]interface{}, error) {
	logLen := len(constructedReceipt.Logs)
	if logLen == 0 {
		metadataCommon.Logger.Log.Errorf("WARNING: LOG data is invalid.")
		return nil, nil
	}
	for i, log := range constructedReceipt.Logs {
		res, err := ParseEVMLogDataByEventName(log, ethContractAddressStr, eventName)
		if err != nil {
			metadataCommon.Logger.Log.Infof("WARNING: skip log #%d - %v", i, err)
		} else {
			return res, nil
		}
	}

	return nil, nil
}

func ParseEVMLogDataByEventName(log *types.Log, ethContractAddressStr string, name string) (map[string]interface{}, error) {
	abiIns, err := abi.JSON(strings.NewReader(common.AbiJson))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(rCommon.HexToAddress(ethContractAddressStr).Bytes(), log.Address.Bytes()) {
		return nil, fmt.Errorf("contract address mismatch, expect %s", ethContractAddressStr)
	}
	event, exists := abiIns.Events[name]
	if !exists {
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.UnexpectedError, fmt.Errorf("event %s not found in vault ABI", name))
	}
	evSigHash := event.Id()
	if !bytes.Equal(log.Topics[0][:], evSigHash[:]) {
		return nil, fmt.Errorf("event %s with topic %s not found in log %x", name, evSigHash.String(), log.Topics[0])
	}
	if len(log.Data) == 0 {
		return nil, fmt.Errorf("logData is empty")
	}
	dataMap := map[string]interface{}{}
	if err = abiIns.UnpackIntoMap(dataMap, name, log.Data); err != nil {
		return nil, metadataCommon.NewMetadataTxError(metadataCommon.UnexpectedError, err)
	}
	return dataMap, nil
}

func GetNetworkTypeByNetworkID(networkID uint8) (uint, error) {
	switch networkID {
	case common.ETHNetworkID, common.BSCNetworkID, common.PLGNetworkID, common.FTMNetworkID:
		return common.EVMNetworkType, nil
	default:
		return 0, errors.New("Not found networkID")
	}
}

func IsBurningConfirmMetaType(metaType int) bool {
	switch metaType {
	case metadataCommon.BurningConfirmMeta, metadataCommon.BurningConfirmMetaV2:
		return true
	case metadataCommon.BurningConfirmForDepositToSCMeta, metadataCommon.BurningConfirmForDepositToSCMetaV2:
		return true
	case metadataCommon.BurningBSCConfirmMeta, metadataCommon.BurningPBSCConfirmForDepositToSCMeta:
		return true
	case metadataCommon.BurningFantomConfirmForDepositToSCMeta, metadataCommon.BurningFantomConfirmMeta:
		return true
	case metadataCommon.BurningPLGConfirmMeta, metadataCommon.BurningPLGConfirmForDepositToSCMeta:
		return true
	case metadataCommon.BurningPRVBEP20ConfirmMeta, metadataCommon.BurningPRVERC20ConfirmMeta:
		return true
	case metadataCommon.BurnForCallConfirmMeta:
		return true
	default:
		return false
	}
}
