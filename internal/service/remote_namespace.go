package service

import (
	"github.com/johnny-morrice/godless/api"
	"github.com/johnny-morrice/godless/crdt"
	"github.com/johnny-morrice/godless/internal/eval"
	"github.com/johnny-morrice/godless/log"
	"github.com/johnny-morrice/godless/query"
	"github.com/pkg/errors"
)

type addResponse struct {
	path crdt.IPFSPath
	err  error
}

type addNamespace struct {
	namespace crdt.Namespace
	result    chan addResponse
}

func (add addNamespace) reply(path crdt.IPFSPath, err error) {
	go func() {
		defer close(add.result)
		add.result <- addResponse{path: path, err: err}
	}()
}

type addIndex struct {
	index  crdt.Index
	result chan addResponse
}

func (add addIndex) reply(path crdt.IPFSPath, err error) {
	go func() {
		defer close(add.result)
		add.result <- addResponse{path: path, err: err}
	}()
}

type remoteNamespace struct {
	RemoteNamespaceOptions
	namespaceTube chan addNamespace
	indexTube     chan addIndex
}

type RemoteNamespaceOptions struct {
	Store         api.RemoteStore
	HeadCache     api.HeadCache
	MemoryImage   api.MemoryImage
	IndexCache    api.IndexCache
	KeyStore      api.KeyStore
	IsPublicIndex bool
}

func MakeRemoteNamespace(options RemoteNamespaceOptions) api.RemoteNamespaceTree {
	remote := &remoteNamespace{
		RemoteNamespaceOptions: options,
		namespaceTube:          make(chan addNamespace),
		indexTube:              make(chan addIndex),
	}
	go remote.AddNamespaces()
	go remote.AddIndices()
	return remote
}

func (rn *remoteNamespace) AddNamespaces() {
	for tubeItem := range rn.namespaceTube {
		add := tubeItem
		go func() {
			path, err := rn.Store.AddNamespace(add.namespace)
			add.reply(path, err)
		}()
	}
}

func (rn *remoteNamespace) AddIndices() {
	for tubeItem := range rn.indexTube {
		add := tubeItem
		index := crdt.EmptyIndex()

		head, headErr := rn.getHead()

		if headErr != nil {
			add.reply(crdt.NIL_PATH, headErr)
		}

		if !crdt.IsNilPath(head) {
			headIndex, loadErr := rn.loadIndex(head)

			if loadErr != nil {
				add.reply(crdt.NIL_PATH, loadErr)
				continue
			}

			index = headIndex
		}

		nextIndex := index.JoinIndex(add.index)
		path, addErr := rn.addIndex(nextIndex)

		if addErr == nil {
			log.Info("Persisted index at: %v", path)
			rn.setHead(path)
		}

		add.reply(path, addErr)
	}
}

func (rn *remoteNamespace) addIndex(index crdt.Index) (crdt.IPFSPath, error) {
	const failMsg = "remoteNamespace.addIndex failed"
	indexAddr, addErr := rn.Store.AddIndex(index)

	if addErr != nil {
		return crdt.NIL_PATH, errors.Wrap(addErr, failMsg)
	}

	cacheErr := rn.IndexCache.SetIndex(indexAddr, index)
	if cacheErr != nil {
		log.Error("Failed to write index cache for: %v (%v)", indexAddr, cacheErr)
	}

	return indexAddr, nil
}

func (rn *remoteNamespace) Close() {
	close(rn.namespaceTube)
	close(rn.indexTube)
}

func (rn *remoteNamespace) Replicate(links []crdt.Link, kvq api.KvQuery) {
	runner := api.APIResponderFunc(func() api.APIResponse { return rn.joinPeerIndex(links) })
	response := runner.RunQuery()
	kvq.WriteResponse(response)
}

func (rn *remoteNamespace) joinPeerIndex(links []crdt.Link) api.APIResponse {
	const failMsg = "remoteNamespace.joinPeerIndex failed"
	failResponse := api.RESPONSE_FAIL

	log.Info("Replicating peer indices...")

	keys := rn.KeyStore.GetAllPublicKeys()

	joined := crdt.EmptyIndex()

	someFailed := false
	for _, link := range links {
		if rn.IsPublicIndex {
			log.Info("Verifying link...")
			isVerified := link.IsVerifiedByAny(keys)
			if !isVerified {
				log.Warn("Skipping unverified Index Link")
				someFailed = true
				continue
			}
			log.Info("Verified link: %v", link.Path)
		}

		peerAddr := link.Path

		theirIndex, theirErr := rn.loadIndex(peerAddr)

		if theirErr != nil {
			log.Error("Failed to replicate Index at: %v", peerAddr)
			someFailed = true
			continue
		}

		joined = joined.JoinIndex(theirIndex)
	}

	indexAddr, perr := rn.insertIndex(joined)

	if perr != nil {
		log.Error("Index replication failed")
		failResponse.Err = errors.Wrap(perr, failMsg)
		return failResponse
	}

	resp := api.RESPONSE_REPLICATE

	if someFailed {
		resp.Msg = "Update ok with load failures"
	}

	log.Info("Index replicated to: %v", indexAddr)

	return resp
}

func (rn *remoteNamespace) loadIndex(indexAddr crdt.IPFSPath) (crdt.Index, error) {
	const failMsg = "remoteNamespace.loadIndex failed"
	cached, cacheErr := rn.IndexCache.GetIndex(indexAddr)

	if cacheErr == nil {
		return cached, nil
	} else {
		log.Warn("Index cache miss for: %v", indexAddr)
	}

	index, err := rn.Store.CatIndex(indexAddr)

	if err != nil {
		return crdt.EmptyIndex(), errors.Wrap(err, failMsg)
	}

	go rn.updateIndexCache(indexAddr, index)

	return index, nil
}

// TODO there are likely to be many reflection features.  Replace switches with polymorphism.
func (rn *remoteNamespace) RunKvReflection(reflect api.APIReflectionType, kvq api.KvQuery) {
	var runner api.APIResponder
	switch reflect {
	case api.REFLECT_HEAD_PATH:
		runner = api.APIResponderFunc(rn.getReflectHead)
	case api.REFLECT_INDEX:
		runner = api.APIResponderFunc(rn.getReflectIndex)
	case api.REFLECT_DUMP_NAMESPACE:
		runner = api.APIResponderFunc(rn.dumpReflectNamespaces)
	default:
		panic("Unknown reflection command")
	}

	response := runner.RunQuery()
	kvq.WriteResponse(response)
}

// TODO Not sure if best place for these to live.
func (rn *remoteNamespace) getReflectHead() api.APIResponse {
	response := api.RESPONSE_REFLECT
	response.ReflectResponse.Type = api.REFLECT_HEAD_PATH

	myAddr, err := rn.getHead()

	if err != nil {
		response.Err = errors.Wrap(err, "remoteNamespace.getReflectHead failed")
		response.Msg = api.RESPONSE_FAIL_MSG
	} else if crdt.IsNilPath(myAddr) {
		response.Err = errors.New("No index available")
		response.Msg = api.RESPONSE_FAIL_MSG
	} else {
		response.ReflectResponse.Path = myAddr
	}

	return response
}

func (rn *remoteNamespace) getReflectIndex() api.APIResponse {
	const failMsg = "remoteNamespace.getReflectIndex failed"
	response := api.RESPONSE_REFLECT

	index, err := rn.loadCurrentIndex()

	if err != nil {
		response.Msg = api.RESPONSE_FAIL_MSG
		response.Err = errors.Wrap(err, failMsg)
		return response
	}

	response.ReflectResponse.Index = index
	response.ReflectResponse.Type = api.REFLECT_INDEX

	return response
}

func (rn *remoteNamespace) dumpReflectNamespaces() api.APIResponse {
	const failMsg = "remoteNamespace.dumpReflectNamespace failed"
	response := api.RESPONSE_REFLECT
	response.ReflectResponse.Type = api.REFLECT_DUMP_NAMESPACE

	index, err := rn.loadCurrentIndex()

	if err != nil {
		response = api.RESPONSE_FAIL
		response.Err = errors.Wrap(err, failMsg)
		response.Type = api.API_REFLECT
		return response
	}

	everything := crdt.EmptyNamespace()

	lambda := api.NamespaceTreeLambda(func(ns crdt.Namespace) api.TraversalUpdate {
		everything = everything.JoinNamespace(ns)
		return api.TraversalUpdate{More: true}
	})
	searcher := api.SignedTableSearcher{
		Reader: lambda,
		Tables: index.AllTables(),
	}

	err = rn.LoadTraverse(searcher)

	if err != nil {
		response = api.RESPONSE_FAIL
		response.Err = errors.Wrap(err, failMsg)
		response.Type = api.API_REFLECT
	}

	response.ReflectResponse.Namespace = everything
	return response
}

// RunKvQuery will block until the result can be written to kvq.
func (rn *remoteNamespace) RunKvQuery(q *query.Query, kvq api.KvQuery) {
	var runner api.APIResponder

	switch q.OpCode {
	case query.JOIN:
		log.Info("Running join...")
		visitor := eval.MakeNamespaceTreeJoin(rn, rn.KeyStore)
		q.Visit(visitor)
		runner = visitor
	case query.SELECT:
		log.Info("Running select...")
		visitor := eval.MakeNamespaceTreeSelect(rn, rn.KeyStore)
		q.Visit(visitor)
		runner = visitor
	default:
		q.OpCodePanic()
	}

	response := runner.RunQuery()
	kvq.WriteResponse(response)
}

// TODO there should be more clarity on who locks and when.
func (rn *remoteNamespace) JoinTable(tableKey crdt.TableName, table crdt.Table) error {
	const failMsg = "remoteNamespace.JoinTable failed"

	joined := crdt.EmptyNamespace().JoinTable(tableKey, table)

	addr, nsErr := rn.insertNamespace(joined)

	if nsErr != nil {
		return errors.Wrap(nsErr, failMsg)
	}

	signed, signErr := crdt.SignedLink(addr, rn.KeyStore.GetAllPrivateKeys())

	if signErr != nil {
		return errors.Wrap(signErr, failMsg)
	}

	index := crdt.EmptyIndex().JoinTable(tableKey, signed)

	_, indexErr := rn.insertIndex(index)

	if indexErr != nil {
		return errors.Wrap(indexErr, failMsg)
	}

	return nil
}

func (rn *remoteNamespace) LoadTraverse(searcher api.NamespaceSearcher) error {
	const failMsg = "remoteNamespace.LoadTraverse failed"

	index, indexerr := rn.loadCurrentIndex()

	if indexerr != nil {
		return errors.Wrap(indexerr, failMsg)
	}

	tableAddrs := searcher.Search(index)

	return rn.traverseTableNamespaces(tableAddrs, searcher)
}

func (rn *remoteNamespace) traverseTableNamespaces(tableAddrs []crdt.Link, f api.NamespaceTreeReader) error {
	nsch, cancelch := rn.namespaceLoader(tableAddrs)
	defer close(cancelch)
	for ns := range nsch {
		log.Info("Traversing another namespace...")
		update := f.ReadNamespace(ns)

		if !(update.More && update.Error == nil) {
			log.Info("Cancelling traverse...")
			cancelch <- struct{}{}
			log.Info("Cancelled traverse")
		}

		if update.Error != nil {
			log.Info("Aborting traverse with error: %v", update.Error)
			return errors.Wrap(update.Error, "traverseTableNamespaces failed")
		}
	}

	return nil
}

// Preload namespaces while the previous is analysed.
func (rn *remoteNamespace) namespaceLoader(addrs []crdt.Link) (<-chan crdt.Namespace, chan<- struct{}) {
	nsch := make(chan crdt.Namespace)
	cancelch := make(chan struct{}, 1)

	go func() {
		defer close(nsch)
		for _, a := range addrs {
			namespace, err := rn.Store.CatNamespace(a.Path)

			if err != nil {
				log.Error("remoteNamespace.namespaceLoader failed: %v", err)
				return
			}

			log.Info("Catted namespace from: %v", a)
			select {
			case <-cancelch:
				return
			case nsch <- namespace:
				break
			}
		}
	}()

	return nsch, cancelch
}

// Load chunks over IPFS
// TODO opportunity to query IPFS in parallel?
func (rn *remoteNamespace) loadCurrentIndex() (crdt.Index, error) {
	myAddr, err := rn.getHead()

	if err != nil {
		return crdt.EmptyIndex(), errors.Wrap(err, "remoteNamespace.loadCurrentIndex failed")
	} else if crdt.IsNilPath(myAddr) {
		return crdt.EmptyIndex(), errors.New("No current index")
	}

	return rn.loadIndex(myAddr)
}

func (rn *remoteNamespace) insertNamespace(namespace crdt.Namespace) (crdt.IPFSPath, error) {
	const failMsg = "remoteNamespace.persistNamespace failed"
	resultChan := make(chan addResponse)
	rn.namespaceTube <- addNamespace{namespace: namespace, result: resultChan}

	result := <-resultChan

	if result.err != nil {
		return crdt.NIL_PATH, errors.Wrap(result.err, failMsg)
	}

	return result.path, nil
}

func (rn *remoteNamespace) insertIndex(index crdt.Index) (crdt.IPFSPath, error) {
	const failMsg = "remoteNamespace.persistIndex failed"

	resultChan := make(chan addResponse)
	rn.indexTube <- addIndex{index: index, result: resultChan}

	result := <-resultChan

	if result.err != nil {
		return crdt.NIL_PATH, errors.Wrap(result.err, failMsg)
	}

	return result.path, nil
}

func (rn *remoteNamespace) updateIndexCache(addr crdt.IPFSPath, index crdt.Index) {
	err := rn.IndexCache.SetIndex(addr, index)
	if err != nil {
		log.Error("Failed to update index cache: %v", err.Error())
	}
}

func (rn *remoteNamespace) getHead() (crdt.IPFSPath, error) {
	return rn.HeadCache.GetHead()
}

func (rn *remoteNamespace) setHead(head crdt.IPFSPath) error {
	return rn.HeadCache.SetHead(head)
}
