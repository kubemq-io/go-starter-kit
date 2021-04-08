package main

import (
	"context"
	"fmt"
	"github.com/kubemq-io/kubemq-go"
	"github.com/nats-io/nuid"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deadLetterChannel := "queues.dead-letter.dead"
	sendCount := 10

	queuesClient, err := kubemq.NewQueuesClient(ctx,
		kubemq.WithAddress("localhost", 50000),
		kubemq.WithClientId("go-sdk-cookbook-queues-dead-letter"),
		kubemq.WithTransportType(kubemq.TransportTypeGRPC))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := queuesClient.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	doneA, err := queuesClient.TransactionStream(ctx, &kubemq.QueueTransactionMessageRequest{

		ClientID:          "go-sdk-cookbook-queues-dead-letter",
		Channel:           "queues.a",
		VisibilitySeconds: 1,
		WaitTimeSeconds:   1,
	}, func(response *kubemq.QueueTransactionMessageResponse, err error) {
		if err != nil {

		} else {
			log.Printf("Queue: Receiver ,MessageID: %s, Body: %s - rejecting message", response.Message.MessageID, string(response.Message.Body))
			time.Sleep(10 * time.Millisecond)
			// do some work that fails
			err = response.Reject()
			if err != nil {
				log.Fatal(err)
			}
		}

	})
	if err != nil {
		log.Fatal(err)
	}
	doneDeadLetterA, err := queuesClient.TransactionStream(ctx, &kubemq.QueueTransactionMessageRequest{

		ClientID:          "go-sdk-cookbook-queues-dead-letter",
		Channel:           deadLetterChannel,
		VisibilitySeconds: 1,
		WaitTimeSeconds:   1,
	}, func(response *kubemq.QueueTransactionMessageResponse, err error) {
		if err != nil {

		} else {
			log.Printf("Queue: Receiver Dead-letter ,MessageID: %s, Body: %s - accepting message", response.Message.MessageID, string(response.Message.Body))
			err = response.Ack()
			if err != nil {
				log.Fatal(err)
			}
		}

	})
	if err != nil {
		log.Fatal(err)
	}
	for i := 1; i <= sendCount; i++ {
		messageID := nuid.New().Next()
		sendResult, err := queuesClient.Send(ctx, kubemq.NewQueueMessage().
			SetId(messageID).
			SetChannel("queues.a").
			SetBody([]byte(fmt.Sprintf("sending message %d", i))).
			SetPolicyMaxReceiveCount(1).
			SetPolicyMaxReceiveQueue(deadLetterChannel))
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Send to Queue Result: MessageID:%s,Sent At: %s\n", sendResult.MessageID, time.Unix(0, sendResult.SentAt).String())
	}
	var gracefulShutdown = make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, syscall.SIGTERM)
	signal.Notify(gracefulShutdown, syscall.SIGINT)
	signal.Notify(gracefulShutdown, syscall.SIGQUIT)
	select {

	case <-gracefulShutdown:

	}
	doneA <- struct{}{}
	doneDeadLetterA <- struct{}{}
}
