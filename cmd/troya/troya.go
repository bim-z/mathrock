/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package troya

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/charmbracelet/log"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/spf13/cobra"
)

var (
	count    int
	interval time.Duration
	size     int
	limit    time.Duration
	slow     time.Duration
	raw      bool
)

var logger = log.NewWithOptions(os.Stdout, log.Options{
	ReportTimestamp: true,
	TimeFormat:      "15:04:05",
	Prefix:          "pong",
})

var root = &cobra.Command{
	Use:   "pong <host>",
	Short: "Ping with beautiful output",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	flags := root.Flags()
	flags.IntVarP(&count, "count", "c", 0, "Number of packets to send, 0 for unlimited")
	flags.DurationVarP(&interval, "interval", "i", time.Second, "Time between packets")
	flags.IntVarP(&size, "size", "s", 56, "Payload size in bytes")
	flags.DurationVarP(&limit, "timeout", "t", 0, "Total time before stopping, 0 for unlimited")
	flags.DurationVar(&slow, "slow", 150*time.Millisecond, "Latency threshold for slow replies")
	flags.BoolVar(&raw, "privileged", false, "Use raw ICMP sockets, requires root")
}

func execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	pinger, err := probing.NewPinger(args[0])
	if err != nil {
		return err
	}

	if count > 0 {
		pinger.Count = count
	}
	if limit > 0 {
		pinger.Timeout = limit
	}
	pinger.Interval = interval
	pinger.Size = size
	pinger.SetPrivileged(raw)

	pinger.OnRecv = func(pkt *probing.Packet) {
		fields := []any{
			"seq", pkt.Seq,
			"bytes", pkt.Nbytes,
			"ttl", pkt.TTL,
			"time", pkt.Rtt.Round(10 * time.Microsecond),
		}
		if pkt.Rtt > slow {
			logger.Warn("slow reply", fields...)
			return
		}
		logger.Info("reply", fields...)
	}

	pinger.OnDuplicateRecv = func(pkt *probing.Packet) {
		logger.Warn("duplicate", "seq", pkt.Seq, "ttl", pkt.TTL)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		pinger.Stop()
	}()

	fmt.Println(header(args[0], pinger.IPAddr().String()))

	if err := pinger.Run(); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(summary(pinger.Statistics()))
	return nil
}
