package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	rl "github.com/pin-yu/system-design-implementation/rate_limiter"
)

func main() {
	rm := rl.NewRateLimiterMap()
	rm.Add("my-ip", rl.NewTokenBucketRateLimiter(5, 100))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	csvC := make(chan []string)

	cancels := []context.CancelFunc{}
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		go func(rId int) {
			log.Printf("client starts, client=%d", rId)
			for {
				select {
				case <-time.After(time.Duration(rand.ExpFloat64()*1000) * time.Millisecond):
					d := time.Now().UnixMilli()
					record := []string{fmt.Sprint(d), fmt.Sprint(rId)}
					if rm.Get("my-ip").TryRequest() {
						record = append(record, "ok")
					} else {
						record = append(record, "fail")
					}
					csvC <- record
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	csvDone := make(chan struct{}, 1)
	go func() {
		// csv
		outputFile, err := os.Create("rate_limiter.csv")
		if err != nil {
			log.Fatal(err)
		}

		writer := csv.NewWriter(outputFile)

		header := []string{"timestamp", "client_id", "success"}
		writer.Write(header)

		for record := range csvC {
			writer.Write(record)
		}

		writer.Flush()
		outputFile.Close()

		csvDone <- struct{}{}
	}()

	<-quit

	for _, cancel := range cancels {
		cancel()
	}

	close(csvC)

	<-csvDone
}
