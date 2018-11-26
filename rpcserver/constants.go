package rpcserver

// rpc cmd method
const (
	GetNetworkInfo     = "getnetworkinfo"
	GetConnectionCount = "getconnectioncount"
	GetAllPeers        = "getallpeers"
	GetRawMempool      = "getrawmempool"
	GetMempoolEntry    = "getmempoolentry"
	EstimateFee        = "estimatefee"
	GetGenerate        = "getgenerate"
	GetMiningInfo      = "getmininginfo"

	GetBestBlock      = "getbestblock"
	GetBestBlockHash  = "getbestblockhash"
	GetBlocks         = "getblocks"
	RetrieveBlock     = "retrieveblock"
	GetBlockChainInfo = "getblockchaininfo"
	GetBlockCount     = "getblockcount"
	GetBlockHash      = "getblockhash"

	ListTransactions                    = "listtransactions"
	CreateRawTransaction                = "createtransaction"
	SendRawTransaction                  = "sendtransaction"
	CreateAndSendTransaction            = "createandsendtransaction"
	CreateAndSendCustomTokenTransaction = "createandsendcustomtokentransaction"
	SendRawCustomTokenTransaction       = "sendrawcustomtokentransaction"
	CreateRawCustomTokenTransaction     = "createrawcustomtokentransaction"
	CreateAndSendLoanRequest            = "createandsendloanrequest"
	CreateAndSendLoanResponse           = "createandsendloanresponse"
	CreateAndSendLoanPayment            = "createandsendloanpayment"
	CreateAndSendLoanWithdraw           = "createandsendloanwithdraw"
	GetMempoolInfo                      = "getmempoolinfo"
	GetCommitteeCandidateList           = "getcommitteecandidate"
	RetrieveCommitteeCandidate          = "retrievecommitteecandidate"
	GetBlockProducerList                = "getblockproducer"
	ListUnspentCustomToken              = "listunspentcustomtoken"
	GetTransactionByHash                = "gettransactionbyhash"
	ListCustomToken                     = "listcustomtoken"
	CustomToken                         = "customtoken"
	CheckHashValue                      = "checkhashvalue"
	GetListCustomTokenBalance           = "getlistcustomtokenbalance"

	GetHeader = "getheader"

	// Wallet rpc cmd
	ListAccounts           = "listaccounts"
	GetAccount             = "getaccount"
	GetAddressesByAccount  = "getaddressesbyaccount"
	GetAccountAddress      = "getaccountaddress"
	DumpPrivkey            = "dumpprivkey"
	ImportAccount          = "importaccount"
	RemoveAccount          = "removeaccount"
	ListUnspent            = "listunspent"
	GetBalance             = "getbalance"
	GetBalanceByPrivatekey = "getbalancebyprivatekey"
	GetReceivedByAccount   = "getreceivedbyaccount"
	SetTxFee               = "settxfee"
	EncryptData            = "encryptdata"

	// multisig for board spending
	CreateSignatureOnCustomTokenTx = "createsignatureoncustomtokentx"
	GetListDCBBoard                = "getlistdcbboard"
	GetListCBBoard                 = "getlistcbboard"
	GetListGOVBoard                = "getlistgovboard"

	// vote
	SendRawVoteBoardDCBTx                = "sendrawvoteboarddcbtx"
	CreateRawVoteDCBBoardTx              = "createrawvotedcbboardtx"
	CreateAndSendVoteDCBBoardTransaction = "createandsendvotedcbboardtransaction"
	SendRawVoteBoardGOVTx                = "sendrawvoteboardgovtx"
	CreateRawVoteGOVBoardTx              = "createrawvotegovboardtx"
	CreateAndSendVoteGOVBoardTransaction = "createandsendvotegovboardtransaction"

	// gov
	GetBondTypes = "getBondTypes"
)
