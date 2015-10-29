package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func cacheFetcher() {
	fmt.Println("cacheFetch!!")
	time.Sleep(2 * time.Second)
	fmt.Println("cacheFetched!!")
}

func main() {
	fd := flag.Uint("fd", 0, "fd to listen and serve")
	port := flag.Uint("port", 3001, "port to listen and serve")
	flag.Parse()

	// まずキャッシュを取る
	// ここを goroutine にしないことで準備ができるまでファイルディスクリプタORポートをListenしないのでリクエストが来ない
	cacheFetcher()

	interval := time.Tick(10 * time.Second)
	go func() {
		for {
			// 10sごとにキャッシュを取ってくる
			<-interval
			cacheFetcher()
		}
	}()

	cs := make(chan os.Signal, 1)
	ec := make(chan int)
	signal.Notify(cs, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for {
			switch <-cs {
			// circusctl reload は SIGTERM を送る
			// 5s待つ
			case syscall.SIGTERM:
				fmt.Println(">> SIGTERM <<")
				time.Sleep(5 * time.Second)
				fmt.Println("ec <- 0")
				ec <- 0
			case syscall.SIGINT:
				ec <- 0
			}
		}
	}()

	go func() {
		for i := 0; ; i++ {
			// 200msごとに数字を出す
			time.Sleep(200 * time.Millisecond)
			fmt.Printf("%d\n", i)
		}
	}()

	http.HandleFunc("/app_check", func(w http.ResponseWriter, r *http.Request) {
		// サーバーとして機能しているか確認できる
		w.WriteHeader(http.StatusTeapot)
	})

	var l net.Listener
	var err error

	if *fd == 0 {
		log.Println(fmt.Sprintf("listening on port %d", *port))
		l, err = net.ListenTCP("tcp", &net.TCPAddr{Port: int(*port)})
	} else {
		log.Println("listening on socket")
		l, err = net.FileListener(os.NewFile(uintptr(*fd), ""))
	}

	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	go func() {
		log.Println(http.Serve(l, nil))
	}()

	<-ec

}
