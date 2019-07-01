package netsync

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/peer"
	"github.com/incognitochain/incognito-chain/pubsub"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wire"
	libp2p "github.com/libp2p/go-libp2p-peer"
	"sync/atomic"
	"testing"
	"time"
)

var (
	bc = &blockchain.BlockChain{}
	pb = pubsub.NewPubSubManager()
	txPool = &mempool.TxPool{}
	server = &Server{}
	consensus = NewConsensus()
	shardToBeaconPool = mempool.GetShardToBeaconPool()
	crossShardPool = make(map[byte]blockchain.CrossShardPool)
	msgBFTPropose = &wire.MessageBFTPropose{
		Layer:      "shard",
		ShardID:    0,
		Block:      []byte{123, 34, 65, 103, 103, 114, 101, 103, 97, 116, 101, 100, 83, 105, 103, 34, 58, 34, 34, 44, 34, 82, 34, 58, 34, 34, 44, 34, 86, 97, 108, 105, 100, 97, 116, 111, 114, 115, 73, 100, 120, 34, 58, 110, 117, 108, 108, 44, 34, 80, 114, 111, 100, 117, 99, 101, 114, 83, 105, 103, 34, 58, 34, 49, 66, 52, 110, 54, 54, 99, 65, 119, 109, 80, 55, 106, 50, 68, 65, 88, 101, 57, 57, 101, 122, 100, 82, 80, 87, 85, 115, 56, 77, 65, 106, 104, 115, 71, 109, 100, 75, 77, 101, 112, 80, 106, 74, 82, 115, 82, 84, 114, 102, 67, 82, 85, 75, 84, 99, 68, 118, 99, 74, 77, 106, 120, 90, 52, 115, 100, 83, 85, 90, 67, 69, 101, 49, 102, 68, 120, 75, 115, 66, 88, 85, 56, 98, 97, 53, 98, 122, 49, 49, 52, 90, 85, 119, 34, 44, 34, 66, 111, 100, 121, 34, 58, 123, 34, 73, 110, 115, 116, 114, 117, 99, 116, 105, 111, 110, 115, 34, 58, 91, 93, 44, 34, 67, 114, 111, 115, 115, 84, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 115, 34, 58, 123, 125, 44, 34, 84, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 115, 34, 58, 91, 93, 125, 44, 34, 72, 101, 97, 100, 101, 114, 34, 58, 123, 34, 80, 114, 111, 100, 117, 99, 101, 114, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123, 34, 80, 107, 34, 58, 34, 65, 53, 90, 117, 78, 52, 97, 116, 66, 110, 105, 70, 89, 97, 52, 120, 117, 122, 48, 66, 53, 101, 103, 47, 70, 80, 51, 105, 68, 77, 108, 57, 55, 101, 90, 43, 55, 97, 90, 73, 48, 106, 115, 65, 34, 44, 34, 84, 107, 34, 58, 34, 65, 116, 54, 98, 72, 84, 100, 47, 107, 83, 67, 57, 78, 47, 78, 116, 78, 118, 70, 108, 67, 89, 47, 116, 47, 82, 90, 118, 119, 85, 108, 113, 112, 98, 52, 90, 43, 102, 77, 105, 101, 70, 74, 66, 34, 125, 44, 34, 83, 104, 97, 114, 100, 73, 68, 34, 58, 48, 44, 34, 86, 101, 114, 115, 105, 111, 110, 34, 58, 49, 44, 34, 80, 114, 101, 118, 66, 108, 111, 99, 107, 72, 97, 115, 104, 34, 58, 34, 102, 55, 55, 55, 51, 55, 102, 100, 49, 51, 57, 98, 56, 56, 99, 100, 98, 99, 57, 102, 52, 52, 48, 102, 101, 57, 52, 50, 97, 101, 102, 54, 48, 50, 56, 99, 56, 51, 57, 54, 51, 97, 100, 50, 50, 100, 54, 101, 55, 57, 99, 51, 53, 97, 101, 101, 98, 49, 102, 98, 51, 55, 56, 49, 34, 44, 34, 72, 101, 105, 103, 104, 116, 34, 58, 49, 51, 52, 50, 44, 34, 82, 111, 117, 110, 100, 34, 58, 50, 44, 34, 69, 112, 111, 99, 104, 34, 58, 57, 48, 44, 34, 84, 105, 109, 101, 115, 116, 97, 109, 112, 34, 58, 49, 53, 54, 49, 55, 51, 51, 52, 57, 55, 44, 34, 84, 120, 82, 111, 111, 116, 34, 58, 34, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 83, 104, 97, 114, 100, 84, 120, 82, 111, 111, 116, 34, 58, 34, 50, 54, 101, 55, 100, 50, 49, 102, 98, 54, 100, 102, 102, 101, 53, 57, 56, 97, 97, 57, 49, 101, 55, 56, 48, 57, 57, 53, 56, 56, 53, 100, 97, 101, 56, 50, 49, 50, 97, 97, 99, 57, 56, 51, 57, 52, 102, 98, 57, 56, 52, 50, 99, 99, 55, 97, 98, 54, 52, 102, 48, 53, 52, 51, 34, 44, 34, 67, 114, 111, 115, 115, 84, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 82, 111, 111, 116, 34, 58, 34, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 73, 110, 115, 116, 114, 117, 99, 116, 105, 111, 110, 115, 82, 111, 111, 116, 34, 58, 34, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 67, 111, 109, 109, 105, 116, 116, 101, 101, 82, 111, 111, 116, 34, 58, 34, 102, 49, 57, 51, 101, 53, 102, 51, 101, 53, 98, 57, 97, 97, 100, 52, 57, 57, 49, 101, 48, 53, 101, 99, 50, 52, 51, 101, 56, 50, 49, 54, 54, 52, 99, 98, 53, 102, 53, 101, 102, 54, 53, 49, 101, 49, 99, 49, 55, 98, 55, 48, 101, 54, 101, 99, 101, 48, 98, 51, 102, 54, 48, 102, 34, 44, 34, 80, 101, 110, 100, 105, 110, 103, 86, 97, 108, 105, 100, 97, 116, 111, 114, 82, 111, 111, 116, 34, 58, 34, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 67, 114, 111, 115, 115, 83, 104, 97, 114, 100, 115, 34, 58, 34, 34, 44, 34, 66, 101, 97, 99, 111, 110, 72, 101, 105, 103, 104, 116, 34, 58, 52, 52, 56, 57, 44, 34, 66, 101, 97, 99, 111, 110, 72, 97, 115, 104, 34, 58, 34, 49, 57, 52, 51, 52, 100, 53, 50, 101, 55, 98, 99, 102, 101, 98, 99, 101, 50, 57, 57, 57, 56, 49, 48, 56, 101, 50, 57, 102, 50, 50, 102, 98, 49, 53, 52, 48, 100, 101, 102, 54, 49, 52, 51, 54, 102, 101, 101, 50, 55, 49, 101, 98, 56, 56, 51, 48, 56, 97, 101, 101, 48, 49, 51, 34, 44, 34, 84, 111, 116, 97, 108, 84, 120, 115, 70, 101, 101, 34, 58, 123, 125, 125, 125},
		ContentSig: "1PzyQea5pRRjfLkEK8W3oNGbXSGH3SWYrXFPYvHsnLtcs6Fmdpne5fuiz4SfatzDpqLZZFuLc5Sr2oSLsan6rhYYjFfZj6",
		Pubkey:     "17yV5NTyFPm73sHa7tK3mdKbMJmVavdrvZzhCynAexcg81BYfQe",
		Timestamp:  1561733497,
	}
	//{BlkHash:dfa89f9a1d93bef29935b79c4f480da1d0bbe7f81713d338c774051bef3aa240 Ri:[3 69 69 47 30 22 85 174 141 181 207 120 94 218 195 112 26 182 133 60 106 134 232 55 197 41 114 91 210 246 71 119 236] Pubkey:17tN32moCx4QdmV9n9suPxqSCvrQPnujVNgvjwvFrKPTUU4r6rj ContentSig:1VuJx44CHUavQKPCktJxiHH89o31ih3HA7XCuCZN5hNQVje48BXcsrBvZzMEXcuwhovqAtKVTAC2mFhXHSxmcrXaZRDTcd Timestamp:1561733497}
	msgBFTAgree = &wire.MessageBFTAgree{
		Ri:         []byte{3, 69, 69, 47, 30, 22, 85, 174, 141, 181, 207, 120, 94, 218, 195, 112, 26, 182, 133, 60, 106, 134, 232, 55, 197, 41, 114, 91, 210, 246, 71, 119, 236},
		Pubkey:     "17tN32moCx4QdmV9n9suPxqSCvrQPnujVNgvjwvFrKPTUU4r6rj",
		ContentSig: "1VuJx44CHUavQKPCktJxiHH89o31ih3HA7XCuCZN5hNQVje48BXcsrBvZzMEXcuwhovqAtKVTAC2mFhXHSxmcrXaZRDTcd",
		Timestamp:  1561733497,
	}
	// &{BestStateHash:8289d23a4d2b7bd9df1821f9074901f252fda8cae44b53a8ec91cf5863400078 Round:1 Pubkey:17tN32moCx4QdmV9n9suPxqSCvrQPnujVNgvjwvFrKPTUU4r6rj ContentSig:1UMCTHhncrgK54Uyh1q5s5A5jxKegrMkB3e7rhisv8P6VZ1CD4cE5AhrtJ8XB3zUt9wjNkFCoN2Z7oXMMgRwWBsAVfcxjQ Timestamp:1561733485}
	msgBFTReady = &wire.MessageBFTReady{
		Round:      1,
		Pubkey:     "17tN32moCx4QdmV9n9suPxqSCvrQPnujVNgvjwvFrKPTUU4r6rj",
		ContentSig: "1UMCTHhncrgK54Uyh1q5s5A5jxKegrMkB3e7rhisv8P6VZ1CD4cE5AhrtJ8XB3zUt9wjNkFCoN2Z7oXMMgRwWBsAVfcxjQ",
		Timestamp:  1561733485,
	}
	// &{BestStateHash:8289d23a4d2b7bd9df1821f9074901f252fda8cae44b53a8ec91cf5863400078 Round:1 Pubkey:17tN32moCx4QdmV9n9suPxqSCvrQPnujVNgvjwvFrKPTUU4r6rj ContentSig:1UMCTHhncrgK54Uyh1q5s5A5jxKegrMkB3e7rhisv8P6VZ1CD4cE5AhrtJ8XB3zUt9wjNkFCoN2Z7oXMMgRwWBsAVfcxjQ Timestamp:1561733485}
	msgBFTReq = &wire.MessageBFTReq{
		Round:      1,
		Pubkey:     "17tN32moCx4QdmV9n9suPxqSCvrQPnujVNgvjwvFrKPTUU4r6rj",
		ContentSig: "1UMCTHhncrgK54Uyh1q5s5A5jxKegrMkB3e7rhisv8P6VZ1CD4cE5AhrtJ8XB3zUt9wjNkFCoN2Z7oXMMgRwWBsAVfcxjQ",
		Timestamp:  1561733485,
	}
	// &{CommitSig:12GuGZd6TRq6XiqZdewNmAaAdonQhJEXcno7BMKKwk48xxhYNu9FzqzrLd6ypykHyNJuQeTrKNM7gyB34gpDfrwXeaQWDYs R:15urAahVkbs1hK91CQKLnzmt6xx6Fb1rS82gdoy2xZjaxQ9hXo7 ValidatorsIdx:[0 1 2] Pubkey:17yV5NTyFPm73sHa7tK3mdKbMJmVavdrvZzhCynAexcg81BYfQe ContentSig:19J3wNPPdsn4HmWqLh5UAGnoc5AGkEb4y6Enz9iwuAkguLRUf3zcBVCGBMZgn7GtgBhXVic1cNVWC8p725LpKU7Tq1wPc9 Timestamp:1561733485}
	msgBFTCommit = &wire.MessageBFTCommit{
		CommitSig:     "12GuGZd6TRq6XiqZdewNmAaAdonQhJEXcno7BMKKwk48xxhYNu9FzqzrLd6ypykHyNJuQeTrKNM7gyB34gpDfrwXeaQWDYs",
		R:             "15urAahVkbs1hK91CQKLnzmt6xx6Fb1rS82gdoy2xZjaxQ9hXo7",
		ValidatorsIdx: []int{0, 1, 2},
		Pubkey:        "17yV5NTyFPm73sHa7tK3mdKbMJmVavdrvZzhCynAexcg81BYfQe",
		ContentSig:    "19J3wNPPdsn4HmWqLh5UAGnoc5AGkEb4y6Enz9iwuAkguLRUf3zcBVCGBMZgn7GtgBhXVic1cNVWC8p725LpKU7Tq1wPc9",
		Timestamp:     1561733485,
	}
	crossShardBlock    = blockchain.CrossShardBlock{}
	shardToBeaconBlock = blockchain.ShardToBeaconBlock{}
	shardBlock         = blockchain.ShardBlock{}
	beaconBlock        = blockchain.BeaconBlock{}
	msgGetBlockShard   = &wire.MessageGetBlockShard{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		ShardID:          0,
		SenderID:         "",
		Timestamp:        1561733485,
	}
	msgGetBlockShardWithHash   = &wire.MessageGetBlockShard{
		FromPool:         true,
		ByHash:           true,
		BySpecificHeight: false,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		ShardID:          0,
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetBlockShardWithSenderID   = &wire.MessageGetBlockShard{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		ShardID:          0,
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetShardToBeacon   = &wire.MessageGetShardToBeacon{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		ShardID:          0,
		SenderID:         "",
		Timestamp:        1561733485,
	}
	msgGetShardToBeaconWithHash   = &wire.MessageGetShardToBeacon{
		FromPool:         true,
		ByHash:           true,
		BySpecificHeight: false,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		ShardID:          0,
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetShardToBeaconWithSenderID   = &wire.MessageGetShardToBeacon{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		ShardID:          0,
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetCrossShard   = &wire.MessageGetCrossShard{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		FromShardID:          0,
		ToShardID: 1,
		SenderID:         "",
		Timestamp:        1561733485,
	}
	msgGetCrossShardWithHash   = &wire.MessageGetCrossShard{
		FromPool:         true,
		ByHash:           true,
		BySpecificHeight: false,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		FromShardID:          0,
		ToShardID: 1,
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetCrossShardWithSenderID   = &wire.MessageGetCrossShard{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		FromShardID:          0,
		ToShardID: 1,
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetBlockBeacon = &wire.MessageGetBlockBeacon{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		SenderID:         "",
		Timestamp:        1561733485,
	}
	msgGetBlockBeaconWithHash = &wire.MessageGetBlockBeacon{
		FromPool:         true,
		ByHash:           true,
		BySpecificHeight: false,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{},
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgGetBlockBeaconWithSenderID = &wire.MessageGetBlockBeacon{
		FromPool:         true,
		ByHash:           false,
		BySpecificHeight: true,
		BlkHashes:        []common.Hash{},
		BlkHeights:       []uint64{1, 2, 3},
		SenderID:         "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		Timestamp:        1561733485,
	}
	msgPing = &wire.MessagePing{Timestamp: time.Now()}
	msgPeerState = &wire.MessagePeerState{
		Beacon: blockchain.ChainState{
			Height: 0,
			BlockHash: common.HashH([]byte{1}),
			BestStateHash: common.HashH([]byte{1}),
		},
		Shards: make(map[byte]blockchain.ChainState),
		ShardToBeaconPool: make(map[byte][]uint64),
		Timestamp: 1561733485,
		SenderID: "",
	}
	msgPeerStateWithSenderID = &wire.MessagePeerState{
		Beacon: blockchain.ChainState{
			Height: 0,
			BlockHash: common.HashH([]byte{1}),
			BestStateHash: common.HashH([]byte{1}),
		},
		Shards: make(map[byte]blockchain.ChainState),
		ShardToBeaconPool: make(map[byte][]uint64),
		Timestamp: 1561733485,
		SenderID: "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	}
	base58CheckDataTx = "13AHwQTSToFvjZFGjxKZbVKL6Mq2sLaGyevhwFUGMvdNd7745xSXhzJ2q2ra6d8pmb8CgieHM4zgbMJSVpsxJFhprUzUu9Ba7FWYcdmHCkqviF1LhysmBugQP7mEGQVo6PJrFp7Xim72ndfwPovTLhWrqYa8475MAM8XMKMEtYRvNXjPghMJuYvN9yxPPWzXUFQGnw3temoR3heaXMCwH4ZGMVWVaom2prZZwQqdKQkC3Yi2E7hjntDny2oCVsvyHpCv7nAZganw9ZqXc2H4V15GNnvTKWUBRxZo8tRK79tnyoRH23SNtPrc7hG3tJc8gtLoZ2FLQP2UrGPi88XpnzNagTix3ydZ3HUMu29bBRJALKdREn7jsMPECN8EjgZayx2XuZNZGJ1Tqzdq7dosUB1TfKfdXk9rVQXuLqxSM7sszST2kdYQ4sLB5YnpcePtFpX3SsR2fbBB7NWNzdq8JroWiLcAcxmH5jVMtkXTkrdLYtEdHDDSm1RQgPEzTkKT3479Hoge5ocDeVBdEqp8v21vbTLvBYWW7cjy3UH9KucqCRWWN7Tiztbnujs3nkazAAHBLFgSSbmVHD9MGKScqnB249ECGqeoHwCnuTTC2GPehyFb3N67mR9bHNFfJz1MG7qwAt3CDVACR68FqCrBpEkkk1ko2jhedYe2ZAmDAWgL8HWvXK112Ji3ucwyaNm3CvjGDWpyyYAs7TjkF58RtMGojwGCutHNoNHwkEtdGoekRoHfVrTQgmgfsxEsAPbL4JcCNKzjMReDAwGVNXu4kWF6KwFtyD5z6jQyYZZX9aEm1fcMmTvFcmX1DBUjsW4BnELPZR6iYeJkikL7bYZZPLmfzKenD17jFLe4F5xhwFcBPN2kxjTFsLKG6VT2EKeShJGzuWgNAbBVBjgjECk6jrZC44HaXgUTdRcgr3MEfp1JAgegCaKJTaXn3eacHqPQrDovMoJ1wijwok4DNFWhA2Ano75udeZHZjHEvytsdEtL1VV74Rs6dFjz591EHvGqvCVFx9qgTRHmSxRDkGox2rYD5LFXSDsL9AZ2aLaoisysT21Dc81WNuG36QhgRjpZ9LZeaCVL8ffjDV5Wbe8uqedN8uZy9P6ZXothXgCjBiAxsDgyHLAbMqND742Sq3wh2N1mTgTRg75S8ijN1smca3U7Snj2iELXRhLrwr6zR8aUxWe4Tdy19knpbEwBoUQcd5i9pwii3q67nYSKLYQZyPnJC5K1tmhskFp7Sxi1Em4wxm3WrQ9aRj2oGLSaKimQ8ECyD8nKSDRynotXC1mFJB2MaMujDUKkEQ9SiVVg2sDvVy6EHGXvQbJtxFu9LiNYa9s23ro4qMAmNL7GaRVew7ss6Rh2f5D6mhBpEG58DewYwNfQnS43xmhDPQ4x55GGDCbxBApRcQd5pqCxF6HmLvipCy3DKm2F59y9QyKNVr1eJzgyRiXUNZFnaSvXdMKcAwBakNjoDSmFzLTQ711WFN731eCr8KtzVR3hpu9U3mptYwRRZXD8oL7FwZt4LaytcvB6x2Rz7ZkQcRFh6wMcEzVugwZvLPAoFyA4ZhcTsFCW8RKgJ1aEUHg79Rjti8p8QffqYaMapr2zKiqqcd4CxA8ssEy1DCNtXDmyMVBEeyQfr9ELeMk8XMTFYGKYMfF6tSDLQWn4gq3beVXC7vD3H2qGk2WKM8sY6jZC4rmJL1arRrN8NARMLwnkqTMWrtWb5Vmj7YDVR92nxdKFGK7Fabhyz6cQP3SxEAHbMfo4B6C1hTv1XebCp4pcBBLsurndpoDgSkTUFdHBFygg5t3jNByF7eJHoRoDQd892wg53zjY13rQ8QbrMJav"
	base58CheckDataTxToken = "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"
	base58CheckDataTxTokenPrivacy = "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"
)
type Consensus struct {
	ch chan interface{}
}

func NewConsensus() *Consensus {
	return &Consensus{
		ch: make(chan interface{}),
	}
}
type Server struct {}
func (server *Server) PushMessageToPeer(wire.Message, libp2p.ID) error {
	return nil
}
func (server *Server) PushMessageToAll(wire.Message) error {
	return nil
}
var _ = func() (_ struct{}) {
	fmt.Println("This runs before init()!")
	bc.Init(&blockchain.Config{})
	bc.IsTest = true
	txPool.Init(&mempool.Config{
		PubSubManager: pb,
	})
	txPool.IsTest = true
	for i := 0; i < 255; i++ {
		crossShardPool[byte(i)] = mempool.GetCrossShardPool(byte(i))
	}
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func TestNetSyncStart(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	go pb.Start()
	netSync.Start()
	if netSync.started != 1 {
		t.Fatal("Netsync should already start")
	}
	netSync.Start()
	// Test forever loop when start
	go netSync.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.ShardRoleTopic, 2))
	<-time.Tick(1 * time.Second)
	netSync.config.roleInCommitteesMtx.Lock()
	if netSync.config.RoleInCommittees != 2 {
		netSync.config.roleInCommitteesMtx.Unlock()
		t.Fatal("Netsync role not received by pubsub manager")
	}
	netSync.config.roleInCommitteesMtx.Unlock()
	
	shardBlock := blockchain.ShardBlock{}
	shardBlock.Header.Height = 2
	beaconBlock := blockchain.BeaconBlock{}
	beaconBlock.Header.Height = 2
	go netSync.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.NewShardblockTopic, &shardBlock))
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("s" + shardBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	go netSync.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.NewBeaconBlockTopic, &beaconBlock))
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("b" + beaconBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	// test transaction worker
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTx)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	var tx transaction.Tx
	err = json.Unmarshal(rawTxBytes, &tx)
	go netSync.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.TransactionHashEnterNodeTopic, *tx.Hash()))
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheTx(*tx.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.Stop()
}
func TestNetSyncStop(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	defer func() {
		if r := recover(); r != nil {
			t.Skipped()
		} else {
			t.Fatal("Should panic when send to closed chan")
		}
	}()
	netSync.Start()
	netSync.cMessage <- "test"
	netSync.Stop()
	if netSync.shutdown != 1 {
		t.Fatal("Netsync should already start")
	}
	netSync.Stop()
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- "test"
}
func TestNetSyncHandleTxWithRole(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 0
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTx)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	var tx transaction.Tx
	err = json.Unmarshal(rawTxBytes, &tx)
	msg := &wire.MessageTx{Transaction: &tx}
	if !netSync.HandleTxWithRole(msg.Transaction) {
		t.Fatal("NetSync should accept this transaction")
	}
	netSync.config.RoleInCommittees = -1
	netSync.config.RelayShard = []byte{0}
	if !netSync.HandleTxWithRole(msg.Transaction) {
		t.Fatal("NetSync should accept this transaction")
	}
}
func TestNetSyncHandleCacheTx(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	hash := common.HashH([]byte{0})
	res := netSync.HandleCacheTx(hash)
	if res {
		t.Fatal("Hash should not be in cache")
	}
	res = netSync.HandleCacheTx(hash)
	if !res {
		t.Fatal("Hash should be in cache")
	}
}
func TestNetSyncHandleCacheTxHash(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	hash := common.HashH([]byte{0})
	netSync.HandleCacheTxHash(hash)
	res := netSync.HandleCacheTx(hash)
	if !res {
		t.Fatal("Hash should be in cache")
	}
}

func TestNetSyncHandleMessageTx(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 0
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTx)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	var tx transaction.Tx
	err = json.Unmarshal(rawTxBytes, &tx)
	msg := &wire.MessageTx{Transaction: &tx}
	netSync.Start()
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheTx(*msg.Transaction.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.config.RoleInCommittees = -1
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	netSync.Stop()
}

func TestNetSyncHandleMessageTxToken(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 1
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTxToken)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	tx := transaction.TxCustomToken{}
	err = json.Unmarshal(rawTxBytes, &tx)
	if err != nil {
		t.Fatal("Error umarshall tx")
	}
	msg := &wire.MessageTxToken{Transaction: &tx}
	netSync.Start()
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheTx(*msg.Transaction.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.Stop()
	
}

func TestNetSyncHandleMessageTxPrivacyToken(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 1
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTxTokenPrivacy)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	tx := transaction.TxCustomTokenPrivacy{}
	err = json.Unmarshal(rawTxBytes, &tx)
	msg := &wire.MessageTxPrivacyToken{Transaction: &tx}
	netSync.Start()
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- msg
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheTx(*msg.Transaction.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.Stop()
}
func TestHandleCacheBlock(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	hash := common.HashH([]byte{0})
	res := netSync.HandleCacheBlock(hash.String())
	if res {
		t.Fatal("Hash should not be in cache")
	}
	res = netSync.HandleCacheBlock(hash.String())
	if !res {
		t.Fatal("Hash should be in cache")
	}
}
func TestNetSyncHandleMessageBeaconBlock(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	block := blockchain.BeaconBlock{}
	block.Header.Height = 2
	netSync.Start()
	netSync.cMessage <- &wire.MessageBlockBeacon{Block: &block}
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("b" + block.Hash().String())
	if !res {
		t.Fatal("Block should be in pool")
	}
	netSync.Stop()
}
func TestNetSyncHandleMessageShardBlock(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	block := blockchain.ShardBlock{}
	block.Header.Height = 2
	netSync.Start()
	netSync.cMessage <- &wire.MessageBlockShard{Block: &block}
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("s" + block.Hash().String())
	if !res {
		t.Fatal("Block should be in pool")
	}
	netSync.Stop()
}
func TestNetSyncHandleMessageShardToBeacon(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	block := blockchain.ShardToBeaconBlock{}
	block.Header.Height = 2
	netSync.Start()
	netSync.cMessage <- &wire.MessageShardToBeacon{Block: &block}
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("s2b" + block.Hash().String())
	if !res {
		t.Fatal("Block should be in pool")
	}
	netSync.Stop()
}
func TestNetSyncHandleMessageCrossShard(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	block := blockchain.CrossShardBlock{}
	block.Header.Height = 2
	netSync.Start()
	netSync.cMessage <- &wire.MessageCrossShard{Block: &block}
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("c" + block.Hash().String())
	if !res {
		t.Fatal("Block should be in pool")
	}
	netSync.Stop()
}
func TestNetSyncQueueTx(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 0
	pr := &peer.Peer{}
	done := make(chan struct{})
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTx)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	var tx transaction.Tx
	err = json.Unmarshal(rawTxBytes, &tx)
	msg := &wire.MessageTx{Transaction: &tx}
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueTx(pr, msg, done)
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheTx(*msg.Transaction.Hash())
	if res {
		t.Error("Transaction should NOT be in cache")
	}
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueTx(pr, msg, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheTx(*msg.Transaction.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.Stop()
}
func TestNetSyncQueueTxToken(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 0
	pr := &peer.Peer{}
	done := make(chan struct{})
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTxToken)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	var tx transaction.TxCustomToken
	err = json.Unmarshal(rawTxBytes, &tx)
	msg := &wire.MessageTxToken{Transaction: &tx}
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueTxToken(pr, msg, done)
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheTx(*msg.Transaction.Hash())
	if res {
		t.Error("Transaction should NOT be in cache")
	}
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueTxToken(pr, msg, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheTx(*msg.Transaction.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.Stop()
}

func TestNetSyncQueueTxPrivacyToken(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.config.RoleInCommittees = 0
	pr := &peer.Peer{}
	done := make(chan struct{})
	rawTxBytes, _, err := base58.Base58Check{}.Decode(base58CheckDataTxTokenPrivacy)
	if err != nil {
		t.Fatal("Error parse tx", err)
	}
	var tx transaction.TxCustomTokenPrivacy
	err = json.Unmarshal(rawTxBytes, &tx)
	msg := &wire.MessageTxPrivacyToken{Transaction: &tx}
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueTxPrivacyToken(pr, msg, done)
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheTx(*msg.Transaction.Hash())
	if res {
		t.Error("Transaction should NOT be in cache")
	}
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueTxPrivacyToken(pr, msg, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheTx(*msg.Transaction.Hash())
	if !res {
		t.Error("Transaction should be in cache")
	}
	netSync.Stop()
}

func TestNetSyncQueueBlock(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	pr := &peer.Peer{}
	done := make(chan struct{})
	crossShardBlock.Header.Height = 2
	msgCrossShardBlock := &wire.MessageCrossShard{Block: &crossShardBlock}
	shardToBeaconBlock.Header.Height = 2
	msgShardToBeaconBlock := &wire.MessageShardToBeacon{Block: &shardToBeaconBlock}
	shardBlock.Header.Height = 2
	msgShardBlock := &wire.MessageBlockShard{Block: &shardBlock}
	beaconBlock.Header.Height = 2
	msgBeaconBlock := &wire.MessageBlockBeacon{Block: &beaconBlock}
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueBlock(pr, msgCrossShardBlock, done)
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("c" + crossShardBlock.Header.Hash().String())
	if res {
		t.Error("Block should NOT be in cache")
	}
	go func() {
		<-done
	}()
	netSync.QueueBlock(pr, msgShardToBeaconBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("s2b" + shardToBeaconBlock.Header.Hash().String())
	if res {
		t.Error("Block should NOT be in cache")
	}
	go func() {
		<-done
	}()
	netSync.QueueBlock(pr, msgShardBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("s" + shardBlock.Header.Hash().String())
	if res {
		t.Error("Block should NOT be in cache")
	}
	go func() {
		<-done
	}()
	netSync.QueueBlock(pr, msgBeaconBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("b" + beaconBlock.Header.Hash().String())
	if res {
		t.Error("Block should NOT be in cache")
	}
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueBlock(pr, msgCrossShardBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("c" + crossShardBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	netSync.QueueBlock(pr, msgShardToBeaconBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("s2b" + shardToBeaconBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	netSync.QueueBlock(pr, msgShardBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("s" + shardBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	netSync.QueueBlock(pr, msgBeaconBlock, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("b" + beaconBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	netSync.Stop()
}

func TestNetSyncQueueGetBlockShard(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	pr := &peer.Peer{}
	done := make(chan struct{})
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueGetBlockShard(pr, msgGetBlockShard, done)
	<-time.Tick(1 * time.Second)
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueGetBlockShard(pr, msgGetBlockShard, done)
	<-time.Tick(1 * time.Second)
	netSync.QueueGetBlockShard(pr, msgGetBlockShardWithSenderID, done)
	<-time.Tick(1 * time.Second)
	netSync.QueueGetBlockShard(pr, msgGetBlockShardWithHash, done)
	<-time.Tick(1 * time.Second)
	netSync.Stop()
}
func TestNetSyncQueueGetBlockBeacon(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	pr := &peer.Peer{}
	done := make(chan struct{})
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueGetBlockBeacon(pr, msgGetBlockBeacon, done)
	<-time.Tick(1 * time.Second)
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueGetBlockBeacon(pr, msgGetBlockBeacon, done)
	<-time.Tick(1 * time.Second)
	netSync.QueueGetBlockBeacon(pr, msgGetBlockBeaconWithSenderID, done)
	<-time.Tick(1 * time.Second)
	netSync.QueueGetBlockBeacon(pr, msgGetBlockBeaconWithHash, done)
	<-time.Tick(1 * time.Second)
	netSync.Stop()
}
func TestNetSyncHandleMessageGetShardToBeacon(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
		ShardToBeaconPool: shardToBeaconPool,
	})
	consensus.ch = make(chan interface{})
	// start netsyc
	netSync.Start()
	netSync.cMessage <- msgGetShardToBeacon
	//<-time.Tick(1 * time.Second)
	netSync.cMessage <- msgGetShardToBeaconWithHash
	//<-time.Tick(1 * time.Second)
	netSync.cMessage <- msgGetShardToBeaconWithSenderID
	<-time.Tick(3 * time.Second)
	netSync.Stop()
}
func TestNetSyncHandleMessageGetCrossShard(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
		CrossShardPool: crossShardPool,
	})
	consensus.ch = make(chan interface{})
	// start netsyc
	netSync.Start()
	netSync.cMessage <- msgGetCrossShard
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- msgGetCrossShardWithHash
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- msgGetCrossShardWithSenderID
	<-time.Tick(1 * time.Second)
	netSync.Stop()
}
func TestNetSyncQueueMessage(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	pr := &peer.Peer{}
	done := make(chan struct{})
	// no start net sync
	if atomic.AddInt32(&netSync.shutdown, 1) != 1 {
		t.Fatal("Netsync is not shutdown")
	}
	go func() {
		<-done
	}()
	netSync.QueueMessage(pr, msgPing, done)
	<-time.Tick(1 * time.Second)
	res := netSync.HandleCacheBlock("s" + shardBlock.Header.Hash().String())
	if res {
		t.Error("Block should NOT be in cache")
	}
	if atomic.AddInt32(&netSync.shutdown, -1) != 0 {
		t.Fatal("Netsync is shutdown")
	}
	// start netsyc
	netSync.Start()
	netSync.QueueMessage(pr, msgPing, done)
	<-time.Tick(1 * time.Second)
	res = netSync.HandleCacheBlock("s" + shardBlock.Header.Hash().String())
	if !res {
		t.Error("Block should be in cache")
	}
	netSync.Stop()
}
func (consensus *Consensus) OnBFTMsg(msg wire.Message) {
	switch msg.(type) {
	case *wire.MessageBFTPropose:
		consensus.ch <- msg.(*wire.MessageBFTPropose)
		return
	case *wire.MessageBFTAgree:
		consensus.ch <- msg.(*wire.MessageBFTAgree)
		return
	case *wire.MessageBFTCommit:
		consensus.ch <- msg.(*wire.MessageBFTCommit)
		return
	case *wire.MessageBFTReady:
		consensus.ch <- msg.(*wire.MessageBFTReady)
		return
	case *wire.MessageBFTReq:
		consensus.ch <- msg.(*wire.MessageBFTReq)
		return
	default:
		consensus.ch <- msg
		return
	}
}
func TestNetSyncHandleMessageBFTMsgError(t *testing.T) {
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain:    bc,
		PubSubManager: pb,
		Server:        server,
		TxMemPool:     txPool,
		Consensus:     consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.Start()
	// fail to verify sanity
	go func() {
		netSync.cMessage <- &wire.MessageBFTPropose{}
	}()
	<-time.Tick(1 * time.Second)
	netSync.Stop()
}
func TestNetSyncHandleMessageBFTMsg(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	go netSync.HandleMessageBFTMsg(msgPing)
	now := time.Now()
out:
	for {
		select {
		case temp := <-consensus.ch:
			_, ok := temp.(*wire.MessagePing)
			if !ok {
				t.Fatal("Wrong Message Type")
			} else {
				break out
			}
		default:
			if time.Since(now).Seconds() >= time.Duration(30*time.Second).Seconds() {
				t.Fatal("Timeout")
			}
		}
	}
	netSync.Start()
	go func() {
		netSync.cMessage <- msgBFTPropose
	}()
	<-time.Tick(1 * time.Second)
	now = time.Now()
out2:
	for {
		select {
		case temp := <-consensus.ch:
			_, ok := temp.(*wire.MessageBFTPropose)
			if !ok {
				t.Fatal("Wrong Message Type, should be MessageBFTPropose")
			} else {
				break out2
			}
		default:
			if time.Since(now).Seconds() >= time.Duration(30*time.Second).Seconds() {
				t.Fatal("Timeout")
			}
		}
	}
	go func() {
		hash, _ := common.NewHashFromStr("dfa89f9a1d93bef29935b79c4f480da1d0bbe7f81713d338c774051bef3aa240")
		msgBFTAgree.BlkHash = *hash
		netSync.cMessage <- msgBFTAgree
	}()
	<-time.Tick(1 * time.Second)
	now = time.Now()
out3:
	for {
		select {
		case temp := <-consensus.ch:
			_, ok := temp.(*wire.MessageBFTAgree)
			if !ok {
				t.Fatal("Wrong Message Type, should be MessageBFTAgree")
			} else {
				break out3
			}
		default:
			if time.Since(now).Seconds() >= time.Duration(30*time.Second).Seconds() {
				t.Fatal("Timeout")
			}
		}
	}
out4:
	for {
		go func() {
			netSync.cMessage <- msgBFTCommit
		}()
		<-time.Tick(1 * time.Second)
		now = time.Now()
		select {
		case temp := <-consensus.ch:
			_, ok := temp.(*wire.MessageBFTCommit)
			if !ok {
				t.Fatal("Wrong Message Type, should be MessageBFTCommit")
			} else {
				break out4
			}
		default:
			if time.Since(now).Seconds() >= time.Duration(30*time.Second).Seconds() {
				t.Fatal("Timeout")
			}
		}
	}
	go func() {
		hash, _ := common.NewHashFromStr("8289d23a4d2b7bd9df1821f9074901f252fda8cae44b53a8ec91cf5863400078")
		msgBFTReady.BestStateHash = *hash
		netSync.cMessage <- msgBFTReady
	}()
	<-time.Tick(1 * time.Second)
	now = time.Now()
out5:
	for {
		select {
		case temp := <-consensus.ch:
			_, ok := temp.(*wire.MessageBFTReady)
			if !ok {
				t.Fatal("Wrong Message Type, should be MessageBFTReady")
			} else {
				break out5
			}
		default:
			if time.Since(now).Seconds() >= time.Duration(30*time.Second).Seconds() {
				t.Fatal("Timeout")
			}
		}
	}
	go func() {
		hash, _ := common.NewHashFromStr("8289d23a4d2b7bd9df1821f9074901f252fda8cae44b53a8ec91cf5863400078")
		msgBFTReq.BestStateHash = *hash
		netSync.cMessage <- msgBFTReq
	}()
	<-time.Tick(1 * time.Second)
	now = time.Now()
out6:
	for {
		select {
		case temp := <-consensus.ch:
			_, ok := temp.(*wire.MessageBFTReq)
			if !ok {
				t.Fatal("Wrong Message Type, should be MessageBFTReq")
			} else {
				break out6
			}
		default:
			if time.Since(now).Seconds() >= time.Duration(30*time.Second).Seconds() {
				t.Fatal("Timeout")
			}
		}
	}
}

func TestHandleMessagePeerState(t *testing.T){
	netSync := NetSync{}.New(&NetSyncConfig{
		BlockChain: bc,
		PubSubManager: pb,
		Server: server,
		TxMemPool: txPool,
		Consensus: consensus,
	})
	consensus.ch = make(chan interface{})
	netSync.Start()
	netSync.cMessage <- msgPeerState
	<-time.Tick(1 * time.Second)
	netSync.cMessage <- msgPeerStateWithSenderID
	<-time.Tick(1 * time.Second)
	netSync.Stop()
}

