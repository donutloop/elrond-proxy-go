package process

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/ElrondNetwork/elrond-proxy-go/data"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)


const bulkSizeThreshold = 800000 // 0.8MB
type responseErrorHandler func(res *esapi.Response) error

type snapshotIndexer struct {
	es *elasticsearch.Client
	startDate time.Time
}

func NewSnapshotIndexer()  (*snapshotIndexer, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"https://search-mex-distribution-db-egzjxb77fzmpbs6drhqyc3n32q.eu-west-3.es.amazonaws.com"},
		//Username: "",
		//Password: "",
	})
	if err != nil {
		return nil, err
	}

	return &snapshotIndexer{
		es: es,
		startDate: time.Unix(1618779601, 0),
	}, nil
}

func (si *snapshotIndexer) IndexSnapshot(snapshotList []*data.SnapshotItem, timestamp string) error {
	// 1. Compute snapshot index by week number
	checkpoint := time.Now().UTC()
	if timestamp != "" {
		bigTimestamp, ok := big.NewInt(0).SetString(timestamp, 10)
		if !ok {
			return errors.New("invalid timestamp provided for snapshot checkpoint")
		}
		checkpoint = time.Unix(bigTimestamp.Int64(), 0)
	}
	diff := checkpoint.Sub(si.startDate)
	dayNumber := (int(diff.Hours() / 24) + 1) % 7
	weekNumber := int(diff.Hours() / 24 / 7) + 1
	indexName := "snapshot-week-" + strconv.Itoa(weekNumber)

	for index, _ := range snapshotList {
		snapshotList[index].DayOfTheWeek = dayNumber
	}

	// 2. Check/create index
	if !si.indexExists(indexName) {
		err := si.createIndex(indexName)
		if err != nil {
			return err
		}
	}

	// 3. Start indexing
	buffSlice, err := serializeSnapshotItems(snapshotList)
	if err != nil {
		return err
	}
	for idx := range buffSlice {
		err = si.doBulkRequest(&buffSlice[idx], indexName)
		if err != nil {
			log.Warn("indexer: indexing bulk of accounts",
				"error", err.Error())
			return err
		}
	}

	return nil
}

func (si *snapshotIndexer) IndexUndelegatedValues(snapshotList []*data.Delegator, index int) error {
	indexName := "undelegated-week-1"
	snapshotItems := make([]*data.SnapshotItem, len(snapshotList))

	for i := 0; i < len(snapshotList); i++ {
		snapshotItems[i] = &data.SnapshotItem{
			Address: snapshotList[i].DelegatorAddress,
			Unstaked: snapshotList[i].UndelegatedTotal,
			DayOfTheWeek: (index + 1) % 7,
		}
	}

	if !si.indexExists(indexName) {
		err := si.createIndex(indexName)
		if err != nil {
			return err
		}
	}

	// 3. Start indexing
	buffSlice, err := serializeSnapshotItems(snapshotItems)
	if err != nil {
		return err
	}
	for idx := range buffSlice {
		err = si.doBulkRequest(&buffSlice[idx], indexName)
		if err != nil {
			log.Warn("indexer: indexing bulk of accounts",
				"error", err.Error())
			return err
		}
	}

	return nil
}

func (si *snapshotIndexer) IndexMexValues(mexValues []*data.MexItem) error {
	indexName := "mex-week-1"
	if !si.indexExists(indexName) {
		err := si.createIndex(indexName)
		if err != nil {
			return err
		}
	}

	// 3. Start indexing
	buffSlice, err := serializeMexValues(mexValues)
	if err != nil {
		return err
	}
	for idx := range buffSlice {
		err = si.doBulkRequest(&buffSlice[idx], indexName)
		if err != nil {
			log.Warn("indexer: indexing bulk of accounts",
				"error", err.Error())
			return err
		}
	}

	return nil
}

// IndexExists checks if a given index already exists
func (si *snapshotIndexer) indexExists(index string) bool {
	res, err := si.es.Indices.Exists([]string{index})
	return exists(res, err)
}

// CreateIndex creates an elasticsearch index
func (si *snapshotIndexer) createIndex(index string) error {
	res, err := si.es.Indices.Create(index)
	if err != nil {
		return err
	}

	return parseResponse(res, nil, elasticDefaultErrorResponseHandler)
}

// DoBulkRequest will do a bulk of request to elastic server
func (si *snapshotIndexer) doBulkRequest(buff *bytes.Buffer, index string) error {
	reader := bytes.NewReader(buff.Bytes())

	res, err := si.es.Bulk(reader, si.es.Bulk.WithIndex(index))
	if err != nil {
		log.Warn("elasticClient.DoBulkRequest",
			"indexer do bulk request no response", err.Error())
		return err
	}

	return parseResponse(res, nil, elasticDefaultErrorResponseHandler)
}

/**
 * parseResponse will check and load the elastic/kibana api response into the destination objectsMap. Custom errorHandler
 *  can be passed for special requests that want to handle StatusCode != 200. Every responseErrorHandler
 *  implementation should call loadResponseBody or consume the response body in order to be able to
 *  reuse persistent TCP connections: https://github.com/elastic/go-elasticsearch#usage
 */
func parseResponse(res *esapi.Response, dest interface{}, errorHandler responseErrorHandler) error {
	defer func() {
		if res != nil && res.Body != nil {
			err := res.Body.Close()
			if err != nil {
				log.Warn("elasticClient.parseResponse",
					"could not close body", err.Error())
			}
		}
	}()

	if errorHandler == nil {
		errorHandler = elasticDefaultErrorResponseHandler
	}

	if res.StatusCode != http.StatusOK {
		return errorHandler(res)
	}

	err := loadResponseBody(res.Body, dest)
	if err != nil {
		log.Warn("elasticClient.parseResponse",
			"could not load response body:", err.Error())
		return err
	}

	return nil
}

func loadResponseBody(body io.ReadCloser, dest interface{}) error {
	if body == nil {
		return nil
	}
	if dest == nil {
		_, err := io.Copy(ioutil.Discard, body)
		return err
	}

	err := json.NewDecoder(body).Decode(dest)
	return err
}

func exists(res *esapi.Response, err error) bool {
	defer func() {
		if res != nil && res.Body != nil {
			_, _ = io.Copy(ioutil.Discard, res.Body)
			err = res.Body.Close()
			if err != nil {
				log.Warn("elasticClient.exists", "could not close body: ", err.Error())
			}
		}
	}()

	if err != nil {
		log.Warn("elasticClient.IndexExists", "could not check index on the elastic nodes:", err.Error())
		return false
	}

	switch res.StatusCode {
	case http.StatusOK:
		return true
	case http.StatusNotFound:
		return false
	default:
		log.Warn("elasticClient.exists", "invalid status code returned by the elastic nodes:", res.StatusCode)
		return false
	}
}

func elasticDefaultErrorResponseHandler(res *esapi.Response) error {
	responseBody := map[string]interface{}{}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("%w cannot read elastic response body bytes", err)
	}

	err = json.Unmarshal(bodyBytes, &responseBody)
	if err != nil {
		return err
	}

	if res.IsError() {
		if errIsAlreadyExists(responseBody) {
			return nil
		}
	}
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
		return nil
	}

	return fmt.Errorf("error while parsing the response: code returned: %v, body: %v, bodyBytes: %v",
		res.StatusCode, responseBody, bodyBytes)
}

func errIsAlreadyExists(response map[string]interface{}) bool {
	alreadyExistsMessage := "resource_already_exists_exception"
	errKey := "error"
	typeKey := "type"

	errMapI, ok := response[errKey]
	if !ok {
		return false
	}

	errMap, ok := errMapI.(map[string]interface{})
	if !ok {
		return false
	}

	existsString, ok := errMap[typeKey].(string)
	if !ok {
		return false
	}

	return existsString == alreadyExistsMessage
}

func serializeSnapshotItems(snapshotItems []*data.SnapshotItem) ([]bytes.Buffer, error) {
	var buff bytes.Buffer
	buffSlice := make([]bytes.Buffer, 0)
	for _, snapshotItem := range snapshotItems {
		meta, serializedData, errPrepareAcc := prepareSerializedSnapshots(snapshotItem)
		if errPrepareAcc != nil {
			return nil, errPrepareAcc
		}

		// append a newline for each element
		serializedData = append(serializedData, "\n"...)

		buffLenWithCurrentItem := buff.Len() + len(meta) + len(serializedData)
		if buffLenWithCurrentItem > bulkSizeThreshold && buff.Len() != 0 {
			buffSlice = append(buffSlice, buff)
			buff = bytes.Buffer{}
		}

		buff.Grow(len(meta) +len(serializedData))
		_, err := buff.Write(meta)
		if err != nil {
			log.Warn("elastic search: serialize bulk accounts, write meta", "error", err.Error())
			return nil, err
		}
		_, err = buff.Write(serializedData)
		if err != nil {
			log.Warn("elastic search: serialize bulk snapshotItems, write serialized account", "error", err.Error())
			return nil, err
		}
	}

	// check if the last buffer contains data
	if buff.Len() != 0 {
		buffSlice = append(buffSlice, buff)
	}

	return buffSlice, nil
}


func serializeMexValues(mexValues []*data.MexItem) ([]bytes.Buffer, error) {
	var buff bytes.Buffer
	buffSlice := make([]bytes.Buffer, 0)
	for _, mexValue := range mexValues {
		meta, serializedData, errPrepareAcc := prepareSerializedMexValues(mexValue)
		if errPrepareAcc != nil {
			return nil, errPrepareAcc
		}

		// append a newline for each element
		serializedData = append(serializedData, "\n"...)

		buffLenWithCurrentItem := buff.Len() + len(meta) + len(serializedData)
		if buffLenWithCurrentItem > bulkSizeThreshold && buff.Len() != 0 {
			buffSlice = append(buffSlice, buff)
			buff = bytes.Buffer{}
		}

		buff.Grow(len(meta) +len(serializedData))
		_, err := buff.Write(meta)
		if err != nil {
			log.Warn("elastic search: serialize bulk mex items, write meta", "error", err.Error())
			return nil, err
		}
		_, err = buff.Write(serializedData)
		if err != nil {
			log.Warn("elastic search: serialize bulk mex items, write serialized account", "error", err.Error())
			return nil, err
		}
	}

	// check if the last buffer contains data
	if buff.Len() != 0 {
		buffSlice = append(buffSlice, buff)
	}

	return buffSlice, nil
}

func prepareSerializedSnapshots(item *data.SnapshotItem) ([]byte, []byte, error) {
	meta := []byte(fmt.Sprintf(`{ "index" : {  } }%s`, "\n"))
	serializedData, err := json.Marshal(item)
	if err != nil {
		return nil, nil, err
	}

	return meta, serializedData, nil
}

func prepareSerializedMexValues(item *data.MexItem) ([]byte, []byte, error) {
	meta := []byte(fmt.Sprintf(`{ "index" : {  } }%s`, "\n"))
	serializedData, err := json.Marshal(item)
	if err != nil {
		return nil, nil, err
	}

	return meta, serializedData, nil
}