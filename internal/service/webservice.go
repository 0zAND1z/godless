package service

import (
	"bytes"
	"fmt"
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/johnny-morrice/godless/api"
	"github.com/johnny-morrice/godless/internal/http"
	"github.com/johnny-morrice/godless/log"
	"github.com/johnny-morrice/godless/query"
	"github.com/pkg/errors"
)

const API_ROOT = "/api"
const QUERY_API_ROOT = "/query"
const REFLECT_API_ROOT = "/reflect"

type WebService struct {
	API api.APIService
}

func (service *WebService) Handler() gohttp.Handler {
	root := mux.NewRouter()
	topLevel := root.PathPrefix(API_ROOT).Subrouter()

	reflectMux := topLevel.PathPrefix(REFLECT_API_ROOT).Subrouter()
	reflectMux.HandleFunc("/head", service.reflectHead)
	reflectMux.HandleFunc("/index", service.reflectIndex)
	reflectMux.HandleFunc("/namespace", service.reflectDumpNamespace)

	topLevel.HandleFunc(QUERY_API_ROOT, service.runQuery)

	return root
}

func (service *WebService) reflectHead(rw gohttp.ResponseWriter, req *gohttp.Request) {
	log.Info("WebService reflectHead at: %v", req.RequestURI)
	service.reflect(rw, api.REFLECT_HEAD_PATH)
}

func (service *WebService) reflectIndex(rw gohttp.ResponseWriter, req *gohttp.Request) {
	log.Info("WebService reflectIndex at: %v", req.RequestURI)
	service.reflect(rw, api.REFLECT_INDEX)
}

func (service *WebService) reflectDumpNamespace(rw gohttp.ResponseWriter, req *gohttp.Request) {
	log.Info("WebService reflectDumpNamespace at: %v", req.RequestURI)
	service.reflect(rw, api.REFLECT_DUMP_NAMESPACE)
}

func (service *WebService) reflect(rw gohttp.ResponseWriter, reflection api.APIReflectionType) {
	respch, err := service.API.Reflect(reflection)
	service.respond(rw, respch, err)
}

func (service *WebService) runQuery(rw gohttp.ResponseWriter, req *gohttp.Request) {
	log.Info("WebService runQuery at: %v", req.RequestURI)
	q, err := query.DecodeQuery(req.Body)

	if err != nil {
		invalidRequest(rw, err)
		return
	}

	respch, err := service.API.RunQuery(q)
	service.respond(rw, respch, err)
}

func invalidRequest(rw gohttp.ResponseWriter, err error) {
	log.Info("Invalid Request details: %v", err)
	reportErr := sendErr(rw, err)
	if reportErr != nil {
		log.Error("Error sending JSON error report: '%v'", reportErr)
	}
}

// TODO more coherency.
func (service *WebService) respond(rw gohttp.ResponseWriter, respch <-chan api.APIResponse, err error) {
	if err != nil {
		invalidRequest(rw, err)
		return
	}

	resp := <-respch

	err = sendMessage(rw, resp)

	if err != nil {
		log.Error("Error sending response: %v", err)
	}
}

// TODO why are we sending errors in plaintext again?
func sendErr(rw gohttp.ResponseWriter, err error) error {
	message := api.APIResponse{
		Err: err,
	}

	buff := bytes.Buffer{}
	encerr := api.EncodeAPIResponseText(message, &buff)

	if encerr != nil {
		panic(fmt.Sprintf("Bug encoding json error message: '%v'; ", encerr))
	}

	rw.WriteHeader(WEB_API_ERROR)
	rw.Header()[http.CONTENT_TYPE] = []string{http.MIME_PROTO_TEXT}
	_, senderr := rw.Write(buff.Bytes())

	if senderr != nil {
		return errors.Wrap(senderr, "sendErr failed")
	}

	return nil
}

func sendMessage(rw gohttp.ResponseWriter, resp api.APIResponse) error {
	// Encode gob into buffer first to check for encoding errors.
	// TODO is that actually a good idea?
	buff := &bytes.Buffer{}
	encerr := api.EncodeAPIResponse(resp, buff)

	if encerr != nil {
		panic(fmt.Sprintf("BUG encoding resp: %v", encerr))
	}

	rw.Header()[http.CONTENT_TYPE] = []string{http.MIME_PROTO}
	_, senderr := rw.Write(buff.Bytes())

	if senderr != nil {
		return errors.Wrap(senderr, "sendMessage failed")
	}

	return nil
}

const (
	WEB_API_SUCCESS = 200
	WEB_API_ERROR   = 500
)