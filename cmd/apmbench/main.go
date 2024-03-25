// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"testing"
	"time"

	"go.uber.org/zap"
)

func main() {
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}

	extraMetrics := func(*testing.B) {}
	resetStoreFunc := func() {}
	cleanupFunc := func(context.Context) error { return nil }
	if cfg.BenchmarkTelemetryEndpoint != "" {
		telemetry := telemetry{endpoint: cfg.BenchmarkTelemetryEndpoint}
		extraMetrics = func(b *testing.B) {
			m, err := telemetry.GetAll()
			if err != nil {
				logger.Warn("failed to retrive benchmark metrics", zap.Error(err))
				return
			}
			for unit, val := range m {
				b.ReportMetric(val, unit)
			}
		}
		resetStoreFunc = func() {
			if err := telemetry.Reset(); err != nil {
				logger.Warn("failed to reset store, benchmark report may be corrupted", zap.Error(err))
			}
		}
		cleanupFunc = func(ctx context.Context) error {
			t := time.NewTicker(time.Second)
			defer t.Stop()

			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("condition not satisfied: %w", ctx.Err())
				case <-t.C:
					// TODO: take index lag as input/config
					v, err := telemetry.Get("indexlag")
					if err != nil {
						logger.Warn("failed to get cleanup metric, retrying", zap.Error(err))
						continue
					}
					logger.Info("found indexlag cleanup metric", zap.Float64("value", v))
					if v == 0 {
						logger.Info("cleanup metric value met, success")
						return nil
					}
				}
			}
		}
	}
	// Run benchmarks
	if err := Run(
		extraMetrics,
		resetStoreFunc,
		cleanupFunc,
		Benchmark1000Transactions,
		BenchmarkOTLPTraces,
		BenchmarkAgentAll,
		BenchmarkAgentGo,
		BenchmarkAgentNodeJS,
		BenchmarkAgentPython,
		BenchmarkAgentRuby,
		Benchmark10000AggregationGroups,
	); err != nil {
		logger.Fatal("failed to run benchmarks", zap.Error(err))
	}
	logger.Info("finished running benchmarks")
}

func init() {
	testing.Init()
}
