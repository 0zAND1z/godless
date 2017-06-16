package service

import (
	"fmt"

	"github.com/johnny-morrice/godless/api"
	"github.com/johnny-morrice/godless/log"
)

type keyValueStore struct {
	namespace api.RemoteNamespace
	queue     api.RequestPriorityQueue
	semaphore chan struct{}
}

func LaunchKeyValueStore(ns api.RemoteNamespace, queue api.RequestPriorityQueue, queryLimit int) (api.APIService, <-chan error) {
	errch := make(chan error, 1)

	kv := &keyValueStore{
		namespace: ns,
		queue:     queue,
	}

	if queryLimit > 0 {
		kv.semaphore = make(chan struct{}, queryLimit)
	}

	go kv.executeLoop(errch)

	return kv, errch
}

func (kv *keyValueStore) executeLoop(errch chan<- error) {
	defer close(errch)
	for anything := range kv.queue.Drain() {
		thing := anything
		go func() {
			kv.lockResource()
			defer kv.unlockResource()
			if thing == nil {
				panic("Drained nil")
			}

			log.Info("API executing request")
			kvq, ok := thing.(api.KvQuery)

			if !ok {
				// errch <- fmt.Errorf("Corrupt queue found a '%v' but expected %v: %v", reflect.TypeOf(anything).Name(), reflect.TypeOf(api.KvQuery{}).Name(), anything)
				log.Error("Corrupt queue")
				errch <- fmt.Errorf("Corrupt queue")
			}

			kv.run(kvq)
		}()

	}
}

func (kv *keyValueStore) run(kvq api.KvQuery) {
	kvq.Run(kv.namespace)
}

func (kv *keyValueStore) lockResource() {
	if kv.semaphore == nil {
		return
	}

	kv.semaphore <- struct{}{}
}

func (kv *keyValueStore) unlockResource() {
	if kv.semaphore == nil {
		return
	}

	<-kv.semaphore
}

func (kv *keyValueStore) Call(request api.APIRequest) (<-chan api.APIResponse, error) {
	switch request.Type {
	case api.API_QUERY:
		return kv.runQuery(request)
	case api.API_REPLICATE:
		return kv.replicate(request)
	case api.API_REFLECT:
		return kv.reflect(request)
	default:
		return nil, fmt.Errorf("Unknown request.Type: %v", request.Type)
	}
}

func (kv *keyValueStore) enqueue(kvq api.KvQuery) {
	go kv.queue.Enqueue(kvq.Request, kvq)
}

func (kv *keyValueStore) replicate(request api.APIRequest) (<-chan api.APIResponse, error) {
	log.Info("api.APIService Replicating: %v", request.Replicate)
	kvq := api.MakeKvReplicate(request)
	kv.enqueue(kvq)

	return kvq.Response, nil
}

func (kv *keyValueStore) runQuery(request api.APIRequest) (<-chan api.APIResponse, error) {
	query := request.Query
	if log.CanLog(log.LOG_INFO) {
		text, err := query.PrettyText()
		if err == nil {
			log.Info("api.APIService running query.Query:\n%v", text)
		} else {
			log.Debug("Failed to pretty print query: %v", err)
		}

	}

	if err := query.Validate(); err != nil {
		log.Warn("Invalid query.Query")
		return nil, err
	}
	kvq := api.MakeKvQuery(request)

	kv.enqueue(kvq)

	return kvq.Response, nil
}

func (kv *keyValueStore) reflect(request api.APIRequest) (<-chan api.APIResponse, error) {
	log.Info("api.APIService running reflect request: %v", request.Replicate)
	kvq := api.MakeKvReflect(request)

	kv.enqueue(kvq)

	return kvq.Response, nil
}

func (kv *keyValueStore) CloseAPI() {
	kv.namespace.Close()
	kv.queue.Close()
}

func convertToFailure(resp *api.APIResponse, message string) {
	respType := resp.Type
	*resp = api.RESPONSE_FAIL
	resp.Type = respType
}
