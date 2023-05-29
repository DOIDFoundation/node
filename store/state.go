package store

import (
	"github.com/DOIDFoundation/node/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/iavl"
)

type StateStore struct {
	db   cosmosdb.DB
	tree *iavl.MutableTree
}

func NewStateStore(db cosmosdb.DB) (*StateStore, error) {
	tree, err := iavl.NewMutableTree(db, 128, false)
	tree.Load()
	if err != nil {
		return nil, err
	}
	return &StateStore{db: db, tree: tree}, nil
}

func (s *StateStore) Set(key []byte, value []byte) (updated bool, err error) {
	return s.tree.Set(key, value)
}

func (s *StateStore) Get(key []byte) ([]byte, error) {
	return s.tree.Get(key)
}

func (s *StateStore) Rollback() {
	s.tree.Rollback()
}

func (s *StateStore) Commit() (types.Hash, error) {
	hash, _, err := s.tree.SaveVersion()
	return hash, err
}
