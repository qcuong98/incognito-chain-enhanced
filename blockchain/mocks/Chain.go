// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	common "github.com/incognitochain/incognito-chain/common"
	incdb "github.com/incognitochain/incognito-chain/incdb"

	incognitokey "github.com/incognitochain/incognito-chain/incognitokey"

	mock "github.com/stretchr/testify/mock"

	multiview "github.com/incognitochain/incognito-chain/multiview"

	time "time"

	types "github.com/incognitochain/incognito-chain/blockchain/types"
)

// Chain is an autogenerated mock type for the Chain type
type Chain struct {
	mock.Mock
}

// BestViewCommitteeFromBlock provides a mock function with given fields:
func (_m *Chain) BestViewCommitteeFromBlock() common.Hash {
	ret := _m.Called()

	var r0 common.Hash
	if rf, ok := ret.Get(0).(func() common.Hash); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	return r0
}

// CommitteeEngineVersion provides a mock function with given fields:
func (_m *Chain) CommitteeEngineVersion() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// CommitteesFromViewHashForShard provides a mock function with given fields: committeeHash, shardID
func (_m *Chain) CommitteesFromViewHashForShard(committeeHash common.Hash, shardID byte) ([]incognitokey.CommitteePublicKey, error) {
	ret := _m.Called(committeeHash, shardID)

	var r0 []incognitokey.CommitteePublicKey
	if rf, ok := ret.Get(0).(func(common.Hash, byte) []incognitokey.CommitteePublicKey); ok {
		r0 = rf(committeeHash, shardID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]incognitokey.CommitteePublicKey)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(common.Hash, byte) error); ok {
		r1 = rf(committeeHash, shardID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateNewBlock provides a mock function with given fields: version, proposer, round, startTime, committees, hash
func (_m *Chain) CreateNewBlock(version int, proposer string, round int, startTime int64, committees []incognitokey.CommitteePublicKey, hash common.Hash) (types.BlockInterface, error) {
	ret := _m.Called(version, proposer, round, startTime, committees, hash)

	var r0 types.BlockInterface
	if rf, ok := ret.Get(0).(func(int, string, int, int64, []incognitokey.CommitteePublicKey, common.Hash) types.BlockInterface); ok {
		r0 = rf(version, proposer, round, startTime, committees, hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.BlockInterface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string, int, int64, []incognitokey.CommitteePublicKey, common.Hash) error); ok {
		r1 = rf(version, proposer, round, startTime, committees, hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateNewBlockFromOldBlock provides a mock function with given fields: oldBlock, proposer, startTime, committees, hash
func (_m *Chain) CreateNewBlockFromOldBlock(oldBlock types.BlockInterface, proposer string, startTime int64, committees []incognitokey.CommitteePublicKey, hash common.Hash) (types.BlockInterface, error) {
	ret := _m.Called(oldBlock, proposer, startTime, committees, hash)

	var r0 types.BlockInterface
	if rf, ok := ret.Get(0).(func(types.BlockInterface, string, int64, []incognitokey.CommitteePublicKey, common.Hash) types.BlockInterface); ok {
		r0 = rf(oldBlock, proposer, startTime, committees, hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.BlockInterface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(types.BlockInterface, string, int64, []incognitokey.CommitteePublicKey, common.Hash) error); ok {
		r1 = rf(oldBlock, proposer, startTime, committees, hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CurrentHeight provides a mock function with given fields:
func (_m *Chain) CurrentHeight() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetActiveShardNumber provides a mock function with given fields:
func (_m *Chain) GetActiveShardNumber() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetBestView provides a mock function with given fields:
func (_m *Chain) GetBestView() multiview.View {
	ret := _m.Called()

	var r0 multiview.View
	if rf, ok := ret.Get(0).(func() multiview.View); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(multiview.View)
		}
	}

	return r0
}

// GetBestViewHash provides a mock function with given fields:
func (_m *Chain) GetBestViewHash() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetBestViewHeight provides a mock function with given fields:
func (_m *Chain) GetBestViewHeight() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetChainDatabase provides a mock function with given fields:
func (_m *Chain) GetChainDatabase() incdb.Database {
	ret := _m.Called()

	var r0 incdb.Database
	if rf, ok := ret.Get(0).(func() incdb.Database); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(incdb.Database)
		}
	}

	return r0
}

// GetChainName provides a mock function with given fields:
func (_m *Chain) GetChainName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetCommittee provides a mock function with given fields:
func (_m *Chain) GetCommittee() []incognitokey.CommitteePublicKey {
	ret := _m.Called()

	var r0 []incognitokey.CommitteePublicKey
	if rf, ok := ret.Get(0).(func() []incognitokey.CommitteePublicKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]incognitokey.CommitteePublicKey)
		}
	}

	return r0
}

// GetCommitteeSize provides a mock function with given fields:
func (_m *Chain) GetCommitteeSize() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetConsensusType provides a mock function with given fields:
func (_m *Chain) GetConsensusType() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetEpoch provides a mock function with given fields:
func (_m *Chain) GetEpoch() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetFinalView provides a mock function with given fields:
func (_m *Chain) GetFinalView() multiview.View {
	ret := _m.Called()

	var r0 multiview.View
	if rf, ok := ret.Get(0).(func() multiview.View); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(multiview.View)
		}
	}

	return r0
}

// GetFinalViewHash provides a mock function with given fields:
func (_m *Chain) GetFinalViewHash() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetFinalViewHeight provides a mock function with given fields:
func (_m *Chain) GetFinalViewHeight() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetLastBlockTimeStamp provides a mock function with given fields:
func (_m *Chain) GetLastBlockTimeStamp() int64 {
	ret := _m.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// GetLastProposerIndex provides a mock function with given fields:
func (_m *Chain) GetLastProposerIndex() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetMaxBlkCreateTime provides a mock function with given fields:
func (_m *Chain) GetMaxBlkCreateTime() time.Duration {
	ret := _m.Called()

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// GetMinBlkInterval provides a mock function with given fields:
func (_m *Chain) GetMinBlkInterval() time.Duration {
	ret := _m.Called()

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// GetPendingCommittee provides a mock function with given fields:
func (_m *Chain) GetPendingCommittee() []incognitokey.CommitteePublicKey {
	ret := _m.Called()

	var r0 []incognitokey.CommitteePublicKey
	if rf, ok := ret.Get(0).(func() []incognitokey.CommitteePublicKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]incognitokey.CommitteePublicKey)
		}
	}

	return r0
}

// GetProposerByTimeSlot provides a mock function with given fields: committeeViewHash, shardID, ts, committees
func (_m *Chain) GetProposerByTimeSlot(committeeViewHash common.Hash, shardID byte, ts int64, committees []incognitokey.CommitteePublicKey) (incognitokey.CommitteePublicKey, int, error) {
	ret := _m.Called(committeeViewHash, shardID, ts, committees)

	var r0 incognitokey.CommitteePublicKey
	if rf, ok := ret.Get(0).(func(common.Hash, byte, int64, []incognitokey.CommitteePublicKey) incognitokey.CommitteePublicKey); ok {
		r0 = rf(committeeViewHash, shardID, ts, committees)
	} else {
		r0 = ret.Get(0).(incognitokey.CommitteePublicKey)
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(common.Hash, byte, int64, []incognitokey.CommitteePublicKey) int); ok {
		r1 = rf(committeeViewHash, shardID, ts, committees)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(common.Hash, byte, int64, []incognitokey.CommitteePublicKey) error); ok {
		r2 = rf(committeeViewHash, shardID, ts, committees)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetPubKeyCommitteeIndex provides a mock function with given fields: _a0
func (_m *Chain) GetPubKeyCommitteeIndex(_a0 string) int {
	ret := _m.Called(_a0)

	var r0 int
	if rf, ok := ret.Get(0).(func(string) int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetShardID provides a mock function with given fields:
func (_m *Chain) GetShardID() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetViewByHash provides a mock function with given fields: hash
func (_m *Chain) GetViewByHash(hash common.Hash) multiview.View {
	ret := _m.Called(hash)

	var r0 multiview.View
	if rf, ok := ret.Get(0).(func(common.Hash) multiview.View); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(multiview.View)
		}
	}

	return r0
}

// InsertBlock provides a mock function with given fields: block, shouldValidate
func (_m *Chain) InsertBlock(block types.BlockInterface, shouldValidate bool) error {
	ret := _m.Called(block, shouldValidate)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.BlockInterface, bool) error); ok {
		r0 = rf(block, shouldValidate)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IsBeaconChain provides a mock function with given fields:
func (_m *Chain) IsBeaconChain() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// IsReady provides a mock function with given fields:
func (_m *Chain) IsReady() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ReplacePreviousValidationData provides a mock function with given fields: previousBlockHash, newValidationData
func (_m *Chain) ReplacePreviousValidationData(previousBlockHash common.Hash, newValidationData string) error {
	ret := _m.Called(previousBlockHash, newValidationData)

	var r0 error
	if rf, ok := ret.Get(0).(func(common.Hash, string) error); ok {
		r0 = rf(previousBlockHash, newValidationData)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetReady provides a mock function with given fields: _a0
func (_m *Chain) SetReady(_a0 bool) {
	_m.Called(_a0)
}

// SigningCommittees provides a mock function with given fields: committeeViewHash, proposerIndex, committees, shardID
func (_m *Chain) GetSigningCommitteesFromBestView(committeeViewHash common.Hash, proposerIndex int, committees []incognitokey.CommitteePublicKey, shardID byte) []incognitokey.CommitteePublicKey {
	ret := _m.Called(committeeViewHash, proposerIndex, committees, shardID)

	var r0 []incognitokey.CommitteePublicKey
	if rf, ok := ret.Get(0).(func(common.Hash, int, []incognitokey.CommitteePublicKey, byte) []incognitokey.CommitteePublicKey); ok {
		r0 = rf(committeeViewHash, proposerIndex, committees, shardID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]incognitokey.CommitteePublicKey)
		}
	}

	return r0
}

// UnmarshalBlock provides a mock function with given fields: blockString
func (_m *Chain) UnmarshalBlock(blockString []byte) (types.BlockInterface, error) {
	ret := _m.Called(blockString)

	var r0 types.BlockInterface
	if rf, ok := ret.Get(0).(func([]byte) types.BlockInterface); ok {
		r0 = rf(blockString)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.BlockInterface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(blockString)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ValidateBlockSignatures provides a mock function with given fields: block, committees
func (_m *Chain) ValidateBlockSignatures(block types.BlockInterface, committees []incognitokey.CommitteePublicKey) error {
	ret := _m.Called(block, committees)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.BlockInterface, []incognitokey.CommitteePublicKey) error); ok {
		r0 = rf(block, committees)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidatePreSignBlock provides a mock function with given fields: block, signingCommittees, committees
func (_m *Chain) ValidatePreSignBlock(block types.BlockInterface, signingCommittees []incognitokey.CommitteePublicKey, committees []incognitokey.CommitteePublicKey) error {
	ret := _m.Called(block, signingCommittees, committees)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.BlockInterface, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey) error); ok {
		r0 = rf(block, signingCommittees, committees)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
