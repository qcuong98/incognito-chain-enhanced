package lvdb

import (
	"bytes"
	"encoding/binary"
	"encoding/json"

	"github.com/ninjadotorg/cash-prototype/blockchain"
	"github.com/ninjadotorg/cash-prototype/common"
	"github.com/ninjadotorg/cash-prototype/database"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (db *db) StoreBlock(v interface{}, chainID byte) error {
	h, ok := v.(hasher)
	if !ok {
		return database.NewDatabaseError(database.NotImplHashMethod, errors.New("v must implement Hash() method"))
	}
	var (
		hash = h.Hash()
		key  = append(append(chainIDPrefix, chainID), append(blockKeyPrefix, hash[:]...)...)
		// key should look like this c10{b-[blockhash]}:{b-[blockhash]}
		keyB = append(blockKeyPrefix, hash[:]...)
		// key should look like this {b-blockhash}:block
	)
	if ok, _ := db.hasValue(key); ok {
		return database.NewDatabaseError(database.BlockExisted, errors.Errorf("block %s already exists", hash.String()))
	}
	val, err := json.Marshal(v)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Marshal"))
	}
	if err := db.put(key, keyB); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.Put"))
	}
	if err := db.put(keyB, val); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.Put"))
	}
	return nil
}

func (db *db) HasBlock(hash *common.Hash) (bool, error) {
	if exists, _ := db.hasValue(db.getKey("block", hash)); exists {
		return true, nil
	}
	return false, nil
}

func (db *db) FetchBlock(hash *common.Hash) ([]byte, error) {
	block, err := db.lvdb.Get(db.getKey("block", hash), nil)
	if err != nil {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	ret := make([]byte, len(block))
	copy(ret, block)
	return ret, nil
}

func (db *db) DeleteBlock(hash *common.Hash, idx int32, chainID byte) error {
	// Delete block
	err := db.lvdb.Delete(db.getKey("block", hash), nil)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	// Delete block index
	err = db.lvdb.Delete(db.getKey("blockidx", hash), nil)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	buf := make([]byte, 5)
	binary.LittleEndian.PutUint32(buf, uint32(idx))
	buf[4] = chainID
	err = db.lvdb.Delete(buf, nil)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	return nil
}

func (db *db) StoreBestState(v interface{}, chainID byte) error {
	val, err := json.Marshal(v)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Marshal"))
	}
	key := append(bestBlockKey, chainID)
	if err := db.put(key, val); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}
	return nil
}

func (db *db) FetchBestState(chainID byte) ([]byte, error) {
	key := append(bestBlockKey, chainID)
	block, err := db.lvdb.Get(key, nil)
	if err != nil {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.get"))
	}
	return block, nil
}

func (db *db) CleanBestState() error {
	for chainID := byte(0); chainID < common.TotalValidators; chainID++ {
		key := append(bestBlockKey, chainID)
		err := db.lvdb.Delete(key, nil)
		if err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
		}
	}
	return nil
}

func (db *db) StoreBlockIndex(h *common.Hash, idx int32, chainID byte) error {
	buf := make([]byte, 5)
	binary.LittleEndian.PutUint32(buf, uint32(idx))
	buf[4] = chainID
	//{i-[hash]}:index-chainid
	if err := db.lvdb.Put(db.getKey("blockidx", h), buf, nil); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}
	//{index-chainid}:[hash]
	if err := db.lvdb.Put(buf, h[:], nil); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}
	return nil
}

func (db *db) GetIndexOfBlock(h *common.Hash) (int32, byte, error) {
	b, err := db.lvdb.Get(db.getKey("blockidx", h), nil)
	//{i-[hash]}:index-chainid
	if err != nil {
		return 0, 0, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.get"))
	}

	var idx int32
	var chainID byte
	if err := binary.Read(bytes.NewReader(b[:4]), binary.LittleEndian, &idx); err != nil {
		return 0, 0, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "binary.Read"))
	}
	if err = binary.Read(bytes.NewReader(b[4:]), binary.LittleEndian, &chainID); err != nil {
		return 0, 0, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "binary.Read"))
	}
	return idx, chainID, nil
}

func (db *db) GetBlockByIndex(idx int32, chainID byte) (*common.Hash, error) {
	buf := make([]byte, 5)
	binary.LittleEndian.PutUint32(buf, uint32(idx))
	buf[4] = chainID
	// {index-chainid}: {blockhash}

	b, err := db.lvdb.Get(buf, nil)
	if err != nil {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	h := new(common.Hash)
	_ = h.SetBytes(b[:])
	return h, nil
}

func (db *db) FetchAllBlocks() (map[byte][]*common.Hash, error) {
	var keys map[byte][]*common.Hash
	for chainID := byte(0); chainID < blockchain.ChainCount; chainID++ {
		prefix := append(append(chainIDPrefix, chainID), blockKeyPrefix...)
		// prefix {c10{b-......}}
		iter := db.lvdb.NewIterator(util.BytesPrefix(prefix), nil)
		for iter.Next() {
			h := new(common.Hash)
			_ = h.SetBytes(iter.Key()[len(prefix):])
			keys[chainID] = append(keys[chainID], h)
		}
		iter.Release()
		if err := iter.Error(); err != nil {
			return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "iter.Error"))
		}
	}

	return keys, nil
}

func (db *db) FetchChainBlocks(chainID byte) ([]*common.Hash, error) {
	var keys []*common.Hash
	prefix := append(append(chainIDPrefix, chainID), blockKeyPrefix...)
	//prefix {c10{b-......}}
	iter := db.lvdb.NewIterator(util.BytesPrefix(prefix), nil)
	for iter.Next() {
		h := new(common.Hash)
		_ = h.SetBytes(iter.Key()[len(prefix):])
		keys = append(keys, h)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "iter.Error"))
	}
	return keys, nil
}
