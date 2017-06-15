package api

import (
	"crypto"

	"github.com/johnny-morrice/godless/crdt"
)

type HeadCache interface {
	SetHead(head crdt.IPFSPath) error
	GetHead() (crdt.IPFSPath, error)
	Rollback() error
	Commit() error
}

type RequestPriorityQueue interface {
	Enqueue(request APIRequest, data interface{}) error
	Drain() <-chan interface{}
	Close() error
}

type PublicKeyId string
type PrivateKeyId string

type KeyCache interface {
	StorePrivateKey(priv crypto.PrivateKey) (PrivateKeyId, error)
	GetPrivateKey(privId PrivateKeyId) (crypto.PrivateKey, error)
	StorePublicKey(pub crypto.PublicKey) (PublicKeyId, error)
	GetPublicKey(pubId PublicKeyId) (crypto.PublicKey, error)
}