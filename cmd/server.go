/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2021 WireGuard LLC. All Rights Reserved.
 */

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/netip"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/zensey/wg-tun2gvisor/netstack"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
)

func main() {
	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr("10.0.0.1")},
		[]netip.Addr{netip.MustParseAddr("8.8.8.8")},
		1420,
	)
	if err != nil {
		log.Panic(err)
	}

	bind := conn.NewDefaultBind()
	dev := device.NewDevice(tun, bind, device.NewLogger(device.LogLevelVerbose, ""))
	err = dev.IpcSet(`listen_port=42642
private_key=f885df17fc8ecb85d4928dfa981b900e0b88453bcb87c5260edb270f31ea026f
public_key=27a1b07f444e86d0f904a937bb409ab0d101da9e2a50070c24a9bb9e9b65e40c
`)
	log.Println("err>", err)

	/*
	   endpoint=192.168.56.101:42000
	   allowed_ip=0.0.0.0/0
	   persistent_keepalive_interval=25
	*/

	// +IXfF/yOy4XUko36mBuQDguIRTvLh8UmDtsnDzHqAm8=
	// J6Gwf0ROhtD5BKk3u0CasNEB2p4qUAcMJKm7nptl5Aw=

	err = dev.IpcSet(`public_key=f982e8c5e40d1ddc302c638a6e089e39e0052462b4962129ebb7970d62658533
allowed_ip=0.0.0.0/0
`)
	log.Println("err>", err)
	dev.Up()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("os.Interrupt >>>")
		// dev.Down()

		// tun.(*netstack.NetTun).Close()
		// tun.Close()
		dev.Down()
		dev.Close()
		<-dev.Wait()
		log.Println("os.Interrupt >>> wait")
		os.Exit(0)
	}()

	_ = tnet

	go func() {
		time.Sleep(time.Minute)
		runtime.GC()
		debug.FreeOSMemory()
	}()

	http.ListenAndServe("localhost:8080", nil)
	select {}

}
