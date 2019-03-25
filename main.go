package main

import (
	 "fmt"
	//"os"
	//"os/signal"
    //"syscall"
    //"flag"
	"log"
	"net/http"
	//"time"	 
)
//var addr = flag.String("addr", ":8080", "http service address")

func main() {
    fmt.Println("world start")
    //flag.Parse()
	hub := newHub()
	go hub.run()
	
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {		
		serveWs(hub, w, r)
	})
	//err := http.ListenAndServe(*addr, nil)
	err := http.ListenAndServe("192.168.0.126:8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
    }
    
    // sigs := make(chan os.Signal, 1)//信号通道
    // done := make(chan bool, 1)
    
    //  signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    
    // go func() {
    //     sig := <-sigs
    //     fmt.Println()
    //     fmt.Println(sig)
    //     done <- true
    // }()
  
    // fmt.Println("awaiting signal")
    // <-done
	// fmt.Println("exiting")
} 