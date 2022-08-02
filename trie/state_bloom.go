package trie

import (
	"encoding/binary"
	"os"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/steakknife/bloomfilter"
)

// stateBloomHasher is a wrapper around a byte blob to satisfy the interface API
// requirements of the bloom library used. It's used to convert a trie hash or
// contract code hash into a 64 bit mini hash.
type stateBloomHasher []byte

func (f stateBloomHasher) Write(p []byte) (n int, err error) { panic("not implemented") }
func (f stateBloomHasher) Sum(b []byte) []byte               { panic("not implemented") }
func (f stateBloomHasher) Reset()                            { panic("not implemented") }
func (f stateBloomHasher) BlockSize() int                    { panic("not implemented") }
func (f stateBloomHasher) Size() int                         { return 8 }
func (f stateBloomHasher) Sum64() uint64                     { return binary.BigEndian.Uint64(f) }

// stateBloom is a bloom filter used during the state convesion(snapshot->state).
// The keys of all generated entries will be recorded here so that in the pruning
// stage the entries belong to the specific version can be avoided for deletion.
//
// The false-positive is allowed here. The "false-positive" entries means they
// actually don't belong to the specific version but they are not deleted in the
// pruning. The downside of the false-positive allowance is we may leave some "dangling"
// nodes in the disk. But in practice the it's very unlike the dangling node is
// state root. So in theory this pruned state shouldn't be visited anymore. Another
// potential issue is for fast sync. If we do another fast sync upon the pruned
// database, it's problematic which will stop the expansion during the syncing.
//
// After the entire state is generated, the bloom filter should be persisted into
// the disk. It indicates the whole generation procedure is finished.
type StateBloom struct {
	bloom *bloomfilter.Filter
}

// newStateBloomWithSize creates a brand new state bloom for state generation.
// The bloom filter will be created by the passing bloom filter size. According
// to the https://hur.st/bloomfilter/?n=600000000&p=&m=2048MB&k=4, the parameters
// are picked so that the false-positive rate for mainnet is low enough.
func NewStateBloomWithSize(size uint64) (*StateBloom, error) {
	bloom, err := bloomfilter.New(size*1024*1024*8, 4)
	if err != nil {
		return nil, err
	}
	return &StateBloom{bloom: bloom}, nil
}

// NewStateBloomFromDisk loads the state bloom from the given file.
// In this case the assumption is held the bloom filter is complete.
func NewStateBloomFromDisk(filename string) (*StateBloom, error) {
	bloom, _, err := bloomfilter.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &StateBloom{bloom: bloom}, nil
}

// Commit flushes the bloom filter content into the disk and marks the bloom
// as complete.
func (bloom *StateBloom) Commit(filename, tempname string) error {
	// Write the bloom out into a temporary file
	_, err := bloom.bloom.WriteFile(tempname)
	if err != nil {
		return err
	}
	// Ensure the file is synced to disk
	f, err := os.OpenFile(tempname, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Move the teporary file into it's final location
	return os.Rename(tempname, filename)
}

// Put implements the KeyValueWriter interface. But here only the key is needed.
func (bloom *StateBloom) Put(key []byte, value []byte) error {
	_, err := common.Hash{}.NewHash(key)
	if err != nil {
		return err
	}
	bloom.bloom.Add(stateBloomHasher(key))
	return nil
}

// Delete removes the key from the key-value data store.
func (bloom *StateBloom) Delete(key []byte) error { panic("not supported") }

// Contain is the wrapper of the underlying contains function which
// reports whether the key is contained.
// - If it says yes, the key may be contained
// - If it says no, the key is definitely not contained.
func (bloom *StateBloom) Contain(key []byte) (bool, error) {
	return bloom.bloom.Contains(stateBloomHasher(key)), nil
}
