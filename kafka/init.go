package kafka

import (
	"fmt"
	"github.com/YasiruR/ktool-backend/database"
	"github.com/YasiruR/ktool-backend/domain"
	"github.com/YasiruR/ktool-backend/log"
	"github.com/google/uuid"
	traceable_context "github.com/pickme-go/traceable-context"
	"github.com/rcrowley/go-metrics"
	"strconv"
)

var (
	ClusterList 			[]domain.KCluster
	RegisteredMetrics 		= 	[]string{"incoming-byte-rate", "outgoing-byte-rate", "request-rate"}
)

//taken from sarama library for histogram sample
const (
	metricsReservoirSize 	= 	1028
	metricsAlphaFactor   	= 	0.015
)

func init() {
	domain.LoggedInUserMap = make(map[int]domain.User)
}

func InitAllClusters() {
	ctx := traceable_context.WithUUID(uuid.New())
	clusterList, err := database.GetAllClusters(ctx)
	if err != nil {
		log.Logger.Fatal("initializing clusters failed")
	}

	var tempClustList []domain.KCluster

	clusterLoop:
	for _, cluster := range clusterList {
		var brokerList []string
		var clustClient domain.KCluster
		clustClient.ClusterID = cluster.ID
		clustClient.ClusterName = cluster.ClusterName

		brokers, err := database.GetBrokersByClusterId(ctx, cluster.ID)
		if err != nil {
			log.Logger.ErrorContext(ctx, "fetching brokers failed for cluster", cluster.ClusterName)
			clustClient.Available = false
			tempClustList = append(tempClustList, clustClient)
			continue
		}

		for _, broker := range brokers {
			addr := broker.Host + ":" + strconv.Itoa(broker.Port)
			brokerList = append(brokerList, addr)
		}

		config, err := InitSaramaConfig(ctx, cluster.ClusterName, "")
		if err != nil {
			log.Logger.ErrorContext(ctx, "initializing sarama config failed and may proceed with default config for consumer and client init", cluster.ClusterName)
		}

		client, err := InitClient(ctx, brokerList, config)
		if err != nil {
			log.Logger.ErrorContext(ctx, "client could not be initialized for cluster", cluster.ClusterName, err)
			clustClient.Available = false
			tempClustList = append(tempClustList, clustClient)
			continue
		}

		saramaBrokers := client.Brokers()
		clustClient.Brokers = saramaBrokers

		saramaConsumer, err := InitSaramaConsumer(ctx, brokerList, config)
		if err != nil {
			log.Logger.ErrorContext(ctx, err,"cluster config could not be initialized for cluster", cluster.ClusterName)
			clustClient.Available = false
			tempClustList = append(tempClustList, clustClient)
			continue
		}

		//to store all broker overview in a cluster
		clustClient.BrokerOverview.Brokers = make(map[int32]domain.BrokerMetrics)

		clustClient.BrokerOverview.TotalIncomingRate = metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry).RateMean()/1024
		clustClient.BrokerOverview.TotalOutgoingRate = metrics.GetOrRegisterMeter("outgoing-byte-rate", config.MetricRegistry).RateMean()/1024
		clustClient.BrokerOverview.TotalRequestRate = metrics.GetOrRegisterMeter("request-rate", config.MetricRegistry).RateMean()
		clustClient.BrokerOverview.TotalResponseRate = metrics.GetOrRegisterMeter("response-rate", config.MetricRegistry).RateMean()

		//user GetOrRegister in metrics library if this does not work, as used in sarama broker
		clustClient.BrokerOverview.TotalRequestLatency = metrics.GetOrRegisterHistogram("request-latency-in-ms", config.MetricRegistry, metrics.NewExpDecaySample(metricsReservoirSize, metricsAlphaFactor)).Mean()
		clustClient.BrokerOverview.TotalRequestSize = metrics.GetOrRegisterHistogram("request-size", config.MetricRegistry, metrics.NewExpDecaySample(metricsReservoirSize, metricsAlphaFactor)).Mean()
		clustClient.BrokerOverview.TotalResponseSize = metrics.GetOrRegisterHistogram("response-size", config.MetricRegistry, metrics.NewExpDecaySample(metricsReservoirSize, metricsAlphaFactor)).Mean()

		//open broker connections to establish metrics along with config
		for _, broker := range saramaBrokers {
			//check if broker is already connected
			connected, err := broker.Connected()
			if err != nil {
				log.Logger.ErrorContext(ctx, err,"checking broker connection with sarama failed", broker.ID(), cluster.ClusterName)
				clustClient.Available = false
				tempClustList = append(tempClustList, clustClient)
				continue
			}

			if !connected {
				err = broker.Open(config)
				if err != nil {
					log.Logger.ErrorContext(ctx, err,"connecting broker to sarama failed", broker.ID(), cluster.ClusterName)
					clustClient.Available = false
					tempClustList = append(tempClustList, clustClient)
					continue
				}
				log.Logger.TraceContext(ctx, "new broker connection opened since it was not connected previously", broker.Addr())
			}

			var brokerMetrics domain.BrokerMetrics

			//todo: unregister all these metrics on app termination and close brokers

			//brokerMetrics.IncomingByteRate = metrics.GetOrRegisterMeter(fmt.Sprintf("incoming-byte-rate-for-broker-%v", broker.ID()), config.MetricRegistry).Rate5()
			//brokerMetrics.OutgoingByteRate = metrics.GetOrRegisterMeter(fmt.Sprintf("outgoing-byte-rate-for-broker-%v", broker.ID()), config.MetricRegistry).Rate5()
			//brokerMetrics.RequestRate = metrics.GetOrRegisterMeter(fmt.Sprintf("request-rate-for-broker-%v", broker.ID()), config.MetricRegistry).Rate5()
			//brokerMetrics.ResponseRate = metrics.GetOrRegisterMeter(fmt.Sprintf("response-rate-for-broker-%v", broker.ID()), config.MetricRegistry).Rate5()
			//
			////user GetOrRegister in metrics library if this does not work, as used in sarama broker
			//brokerMetrics.RequestLatency = metrics.GetOrRegisterHistogram(fmt.Sprintf("request-latency-in-ms-for-broker-%v", broker.ID()), config.MetricRegistry, metrics.NewExpDecaySample(metricsReservoirSize, metricsAlphaFactor)).Mean()
			//brokerMetrics.RequestSize = metrics.GetOrRegisterHistogram(fmt.Sprintf("request-size-for-broker-%v", broker.ID()), config.MetricRegistry, metrics.NewExpDecaySample(metricsReservoirSize, metricsAlphaFactor)).Mean()
			//brokerMetrics.ResponseSize = metrics.GetOrRegisterHistogram(fmt.Sprintf("response-size-for-broker-%v", broker.ID()), config.MetricRegistry, metrics.NewExpDecaySample(metricsReservoirSize, metricsAlphaFactor)).Mean()

			clustClient.BrokerOverview.Brokers[broker.ID()] = brokerMetrics

			fmt.Println("incoming byte count : ", metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry).Count())
			fmt.Println("incoming byte rate mean : ", metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry).RateMean())
			fmt.Println("incoming byte rate 1 : ", metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry).Rate1())
			fmt.Println("incoming byte rate 15 : ", metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry).Rate15())

			fmt.Println("outgoing byte count : ", metrics.GetOrRegisterMeter("outgoing-byte-rate", config.MetricRegistry).Count())
			fmt.Println("outgoing byte rate mean : ", metrics.GetOrRegisterMeter("outgoing-byte-rate", config.MetricRegistry).RateMean())

			fmt.Println("request-rate count : ", metrics.GetOrRegisterMeter("request-rate", config.MetricRegistry).Count())
			fmt.Println("request-rate mean : ", metrics.GetOrRegisterMeter("request-rate", config.MetricRegistry).RateMean())
			fmt.Println("request-rate 1 : ", metrics.GetOrRegisterMeter("request-rate", config.MetricRegistry).Rate1())
			fmt.Println("request-rate 15 : ", metrics.GetOrRegisterMeter("request-rate", config.MetricRegistry).Rate15())

			fmt.Println("response-rate count : ", metrics.GetOrRegisterMeter("response-rate", config.MetricRegistry).Count())
			fmt.Println("response-rate mean : ", metrics.GetOrRegisterMeter("response-rate", config.MetricRegistry).RateMean())
			fmt.Println("response-rate 1 : ", metrics.GetOrRegisterMeter("response-rate", config.MetricRegistry).Rate1())
			fmt.Println("response-rate 15 : ", metrics.GetOrRegisterMeter("response-rate", config.MetricRegistry).Rate15())

			fmt.Println("incoming kb : ", metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry).RateMean()/1024)
			fmt.Println("outgoing kb : ", metrics.GetOrRegisterMeter("outgoing-byte-rate", config.MetricRegistry).RateMean()/1024)

			fmt.Println("producer metrics record send mean rate : ", metrics.GetOrRegisterMeter("record-send-rate", config.MetricRegistry).RateMean())
			fmt.Println("producer metrics record send rate 1 : ", metrics.GetOrRegisterMeter("record-send-rate", config.MetricRegistry).Rate1())

			fmt.Println("all metrics : ", config.MetricRegistry.GetAll())
		}


		topics, err := GetTopicList(ctx, saramaConsumer)
		if err != nil {
			log.Logger.ErrorContext(ctx, "topic list could not be fetched", cluster.ClusterName)
			clustClient.Available = false
			tempClustList = append(tempClustList, clustClient)
			continue
		}

		var numOfPartitions, numOfReplicas, numOfOfflineRepl, numOfInsyncRepl int
		for _, topic := range topics {
			var clusterTopic domain.KTopic
			clusterTopic.Name = topic
			clusterTopic.Partitions, err = saramaConsumer.Partitions(topic)
			if err != nil {
				log.Logger.Error(fmt.Sprintf("partitions could not be fetched for %v topic in %v cluster", topic, cluster.ClusterName), err)
				clustClient.Available = false
				tempClustList = append(ClusterList, clustClient)
				continue clusterLoop
			}
			numOfPartitions += len(clusterTopic.Partitions)
			clustClient.Topics = append(clustClient.Topics, clusterTopic)

			//to fetch information about replicas
			partitionLoop:
			for _, partitionID := range clusterTopic.Partitions {
				replicas, err := client.Replicas(clusterTopic.Name, partitionID)
				if err != nil {
					log.Logger.Error(fmt.Sprintf("replicas could not be fetched for %v topic and %v paritition in %v cluster", topic, partitionID, cluster.ClusterName), err)
					continue partitionLoop
				}
				numOfReplicas += len(replicas)

				insyncReplicas, err := client.InSyncReplicas(clusterTopic.Name, partitionID)
				if err != nil {
					log.Logger.Error(fmt.Sprintf("insync replicas could not be fetched for %v topic and %v paritition in %v cluster", topic, partitionID, cluster.ClusterName), err)
					continue partitionLoop
				}
				numOfInsyncRepl += len(insyncReplicas)

				offlineReplicas, err := client.OfflineReplicas(clusterTopic.Name, partitionID)
				if err != nil {
					log.Logger.Error(fmt.Sprintf("offline replicas could not be fetched for %v topic and %v paritition in %v cluster", topic, partitionID, cluster.ClusterName), err)
					continue partitionLoop
				}
				numOfOfflineRepl += len(offlineReplicas)
			}
		}

		//getting cluster controller id
		controller, err := client.Controller()
		if err != nil {
			log.Logger.ErrorContext(ctx, err, fmt.Sprintf("fetching controller id for the cluster %v failed", cluster.ClusterName))
		} else {
			clustClient.BrokerOverview.ActiveController = controller.Addr()
		}

		//updating all collected broker metrics to the cluster
		clustClient.BrokerOverview.TotalBrokers = len(saramaBrokers)
		clustClient.BrokerOverview.TotalPartitions = numOfPartitions
		clustClient.BrokerOverview.TotalTopics = len(topics)
		clustClient.BrokerOverview.TotalReplicas = numOfReplicas
		clustClient.BrokerOverview.UnderReplicatedPartitions = numOfReplicas - numOfInsyncRepl
		clustClient.BrokerOverview.OfflinePartitions = numOfOfflineRepl

		clustClient.Consumer = saramaConsumer
		clustClient.Client = client
		clustClient.Available = true
		tempClustList = append(tempClustList, clustClient)
	}

	ClusterList = tempClustList

	log.Logger.Trace("cluster initialization completed", fmt.Sprintf("No. of clusters : %v", len(ClusterList)))
}
