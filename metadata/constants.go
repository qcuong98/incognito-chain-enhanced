package metadata

const (
	LoanKeyDigestLength = 32
	LoanKeyLength       = 32
)

const (
	InvalidMeta = iota
	LoanRequestMeta
	LoanResponseMeta
	LoanWithdrawMeta
	LoanUnlockMeta
	LoanPaymentMeta
	BuyFromGOVRequestMeta
	BuyFromGOVResponseMeta
	BuyBackRequestMeta
	BuyBackResponseMeta
	DividendMeta

	//Voting
	SubmitDCBProposalMeta
	VoteDCBProposalMeta
	VoteDCBBoardMeta
	AcceptDCBProposalMeta
	AcceptDCBBoardMeta

	SubmitGOVProposalMeta
	VoteGOVProposalMeta
	VoteGOVBoardMeta
	AcceptGOVProposalMeta
	AcceptGOVBoardMeta
)

const (
	MaxDivTxsPerBlock = 1000
	PayoutFrequency   = 1000 // Payout dividend every 1000 blocks
)
