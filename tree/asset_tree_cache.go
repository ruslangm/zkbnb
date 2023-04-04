package tree

import (
	"github.com/zeromicro/go-zero/core/logx"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	bsmt "github.com/bnb-chain/zkbnb-smt"
)

// Lazy init cache for asset trees
type AssetTreeCache struct {
	initFunction      func(index, block int64) bsmt.SparseMerkleTree
	nextAccountNumber int64
	blockNumber       int64
	mainLock          sync.RWMutex
	changes           map[int64]bool
	changesLock       sync.RWMutex
	treeCache         *lru.Cache
}

type SparseMerkleTreeAdapter struct {
	sparseMerkleTree bsmt.SparseMerkleTree
	changes          map[int64]bool
	changesLock      *sync.RWMutex
}

func NewSparseMerkleTreeAdapter(tree bsmt.SparseMerkleTree, changesLock *sync.RWMutex, changes map[int64]bool) *SparseMerkleTreeAdapter {
	sparseMerkleTreeAdapter := SparseMerkleTreeAdapter{sparseMerkleTree: tree, changesLock: changesLock, changes: changes}
	return &sparseMerkleTreeAdapter
}

func (c *SparseMerkleTreeAdapter) SetWithVersion(key uint64, val []byte, newVersion bsmt.Version) error {
	return c.sparseMerkleTree.SetWithVersion(key, val, newVersion)
}

func (c *SparseMerkleTreeAdapter) MultiSetWithVersion(items []bsmt.Item, newVersion bsmt.Version) error {
	return c.sparseMerkleTree.MultiSetWithVersion(items, newVersion)
}

// Creates new AssetTreeCache
// maxSize defines the maximum size of currently initialized trees
// accountNumber defines the number of accounts to create/or next index for new account
func NewLazyTreeCache(maxSize int, accountNumber int64, blockNumber int64, f func(index, block int64) bsmt.SparseMerkleTree) *AssetTreeCache {
	cache := AssetTreeCache{initFunction: f, nextAccountNumber: accountNumber, blockNumber: blockNumber, changes: make(map[int64]bool, maxSize*10)}
	cache.treeCache, _ = lru.NewWithEvict(maxSize, cache.onDelete)
	return &cache
}

// Updates current cache with new block number and with latest account index
func (c *AssetTreeCache) UpdateCache(accountNumber, latestBlock int64) {
	c.mainLock.Lock()
	if c.nextAccountNumber < accountNumber {
		c.nextAccountNumber = accountNumber
	}
	if c.blockNumber < latestBlock {
		c.blockNumber = latestBlock
	}
	c.mainLock.Unlock()
}

// Returns index of next account
func (c *AssetTreeCache) GetNextAccountIndex() int64 {
	c.mainLock.RLock()
	defer c.mainLock.RUnlock()
	return c.nextAccountNumber + 1
}

func (c *AssetTreeCache) UpdateAccountIndex(accountIndex int64) {
	c.mainLock.Lock()
	if c.nextAccountNumber < accountIndex {
		c.nextAccountNumber = accountIndex
	}
	c.mainLock.Unlock()
}

func (c *AssetTreeCache) GetCurrentAccountIndex() int64 {
	c.mainLock.RLock()
	defer c.mainLock.RUnlock()
	return c.nextAccountNumber
}

// Get Returns asset tree based on account index
func (c *AssetTreeCache) Get(i int64) (tree bsmt.SparseMerkleTree) {
	if tmpTree, ok := c.treeCache.Get(i); ok {
		tree = tmpTree.(bsmt.SparseMerkleTree)
	} else {
		c.treeCache.ContainsOrAdd(i, c.initFunction(i, 0))
		if tmpTree, ok := c.treeCache.Get(i); ok {
			tree = tmpTree.(bsmt.SparseMerkleTree)
		}
	}
	return
}

func (c *AssetTreeCache) GetAdapter(i int64) (treeAdapter *SparseMerkleTreeAdapter) {
	c.changesLock.Lock()
	c.changes[i] = true
	c.changesLock.Unlock()
	return NewSparseMerkleTreeAdapter(c.Get(i), &c.changesLock, c.changes)
}

//Returns slice of indexes of asset trees that were changned
func (c *AssetTreeCache) GetChanges() []int64 {
	c.changesLock.Lock()
	defer c.changesLock.Unlock()
	ret := make([]int64, 0)
	for key := range c.changes {
		ret = append(ret, key)
	}
	return ret
}

//func (c *AssetTreeCache) GetChanges() []int64 {
//	c.mainLock.Lock()
//	c.changesLock.Lock()
//	defer c.mainLock.Unlock()
//	defer c.changesLock.Unlock()
//	for _, key := range c.treeCache.Keys() {
//		tree, _ := c.treeCache.Peek(key)
//		if tree.(bsmt.SparseMerkleTree).LatestVersion()-tree.(bsmt.SparseMerkleTree).RecentVersion() > 1 {
//			c.changes[key.(int64)] = true
//		}
//	}
//	ret := make([]int64, 0, len(c.changes))
//	for key := range c.changes {
//		ret = append(ret, key)
//	}
//	return ret
//}

// Cleans all saved tree changes in the cache
func (c *AssetTreeCache) CleanChanges() {
	c.changesLock.Lock()
	c.changes = make(map[int64]bool, len(c.changes))
	c.changesLock.Unlock()
}

// Internal method to that marks if changes happend to tree eviced from LRU
func (c *AssetTreeCache) onDelete(k, v interface{}) {
	logx.Infof("sparse merkle tree evicted from LRU %s", k)
	c.changesLock.Lock()
	if v.(bsmt.SparseMerkleTree).LatestVersion()-v.(bsmt.SparseMerkleTree).RecentVersion() > 1 {
		c.changes[k.(int64)] = true
	}
	c.changesLock.Unlock()
}
