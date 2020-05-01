package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/ktool-backend/database"
	"github.com/YasiruR/ktool-backend/log"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const promUrl = "http://localhost:9090/api/v1/"

func setBrokerBytesIn(ctx context.Context) (err error) {
	currentTime := strconv.Itoa(int(time.Now().Unix()))
	var response BrokerBytes

	//query bytes in to the broker
	res, err := http.Get(promUrl + "query_range?query=sum%20by%20(instance)%20(rate(kafka_server_brokertopicmetrics_bytesin_total%5B1m%5D))&time=" + currentTime)
	if err != nil {
		log.Logger.ErrorContext(ctx, err, "querying broker total bytes in failed")
		return
	}

	fmt.Println("got res")

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Logger.ErrorContext(ctx, err, "reading broker total bytes in response failed")
		return
	}

	fmt.Println("read all")

	err = json.Unmarshal(content, &response)
	if err != nil {
		log.Logger.ErrorContext(ctx, err, "unmarshalling broker total bytes in response failed")
		return
	}

	fmt.Println("unmarshalled", response)

	for _, result := range response.Data.Result {
		if len(result.Value) < 2 {
			log.Logger.ErrorContext(ctx, "received insufficient values for query", result)
			continue
		}
		byteRate, err := strconv.ParseFloat(result.Value[1].(string), 64)
		if err != nil {
			log.Logger.ErrorContext(ctx, "converting byte in value to float failed", result.Value[1])
			continue
		}

		s := strings.Split(result.Metric.Instance, ":")
		if len(s) < 2 {
			log.Logger.ErrorContext(ctx, "invalid format received for instance", result.Metric.Instance)
			continue
		}
		host := s[0]

		err = database.UpdateBrokerByteInRate(ctx, byteRate, host)
		if err != nil {
			log.Logger.ErrorContext(ctx, "db query to update broker bytes in failed", result.Metric.Instance)
		}
	}

	fmt.Println("done")

	return nil
}

func setBrokerBytesOut(ctx context.Context) (err error) {
	currentTime := strconv.Itoa(int(time.Now().Unix()))
	var response BrokerBytes

	//query bytes out from the broker
	res, err := http.Get(promUrl + "query_range?query=sum%20by%20(instance)%20(rate(kafka_server_brokertopicmetrics_bytesout_total%5B1m%5D))&time=" + currentTime)
	if err != nil {
		log.Logger.ErrorContext(ctx, err, "querying broker total bytes out failed")
		return
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Logger.ErrorContext(ctx, err, "reading broker total bytes out response failed")
		return
	}

	err = json.Unmarshal(content, &response)
	if err != nil {
		log.Logger.ErrorContext(ctx, err, "unmarshalling broker total bytes out response failed")
		return
	}

	for _, result := range response.Data.Result {
		if len(result.Value) < 2 {
			log.Logger.ErrorContext(ctx, "received insufficient values for query", result)
			continue
		}
		byteRate, err := strconv.ParseFloat(result.Value[1].(string), 64)
		if err != nil {
			log.Logger.ErrorContext(ctx, "converting byte out value to float failed", result.Value[1])
			continue
		}

		s := strings.Split(result.Metric.Instance, ":")
		if len(s) < 2 {
			log.Logger.ErrorContext(ctx, "invalid format received for instance", result.Metric.Instance)
			continue
		}
		host := s[0]

		err = database.UpdateBrokerByteOutRate(ctx, byteRate, host)
		if err != nil {
			log.Logger.ErrorContext(ctx, "db query to update broker bytes out failed", result.Metric.Instance)
		}
	}

	return nil
}
