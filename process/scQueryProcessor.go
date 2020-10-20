package process

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-proxy-go/data"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// SCQueryServicePath defines the get values path at which the nodes answer
const SCQueryServicePath = "/vm-values/query"

// SCQueryProcessor is able to process smart contract queries
type SCQueryProcessor struct {
	proc            Processor
	pubKeyConverter core.PubkeyConverter
}

// NewSCQueryProcessor creates a new instance of SCQueryProcessor
func NewSCQueryProcessor(proc Processor, pubKeyConverter core.PubkeyConverter) (*SCQueryProcessor, error) {
	if check.IfNil(proc) {
		return nil, ErrNilCoreProcessor
	}
	if check.IfNil(pubKeyConverter) {
		return nil, ErrNilPubKeyConverter
	}

	return &SCQueryProcessor{
		proc:            proc,
		pubKeyConverter: pubKeyConverter,
	}, nil
}

// ExecuteQuery resolves the request by sending the request to the right observer and replies back the answer
func (scQueryProcessor *SCQueryProcessor) ExecuteQuery(query *data.SCQuery) (*vmcommon.VMOutput, error) {
	addressBytes, err := scQueryProcessor.pubKeyConverter.Decode(query.ScAddress)
	if err != nil {
		return nil, err
	}

	shardID, err := scQueryProcessor.proc.ComputeShardId(addressBytes)
	if err != nil {
		return nil, err
	}

	observers, err := scQueryProcessor.proc.GetObservers(shardID)
	if err != nil {
		return nil, err
	}

	for _, observer := range observers {
		request := scQueryProcessor.createRequestFromQuery(query)
		response := &data.ResponseVmValue{}

		httpStatus, err := scQueryProcessor.proc.CallPostRestEndPoint(observer.Address, SCQueryServicePath, request, response)
		isObserverDown := httpStatus == http.StatusNotFound || httpStatus == http.StatusRequestTimeout
		isOk := httpStatus == http.StatusOK
		responseHasExplicitError := len(response.Error) > 0

		if isObserverDown {
			log.LogIfError(err)
			continue
		}

		if isOk {
			log.Info("SCQueryProcessor.ExecuteQuery, isOk = true", "observer", observer.Address, "shard", shardID)
			return response.Data.Data, nil
		}

		if responseHasExplicitError {
			log.Info("SCQueryProcessor.ExecuteQuery, responseHasExplicitError = true", "observer", observer.Address, "shard", shardID, "error", response.Error)
			return nil, fmt.Errorf(response.Error)
		}

		return nil, err
	}

	return nil, ErrSendingRequest
}

func (scQueryProcessor *SCQueryProcessor) createRequestFromQuery(query *data.SCQuery) data.VmValueRequest {
	request := data.VmValueRequest{}
	request.Address = query.ScAddress
	request.FuncName = query.FuncName
	request.Args = make([]string, len(query.Arguments))
	for i, argument := range query.Arguments {
		argumentAsHex := hex.EncodeToString(argument)
		request.Args[i] = argumentAsHex
	}

	return request
}
