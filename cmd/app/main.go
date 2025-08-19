package main

import (
	"flag"
	"go-cf-zone-switch/pkg/at"
	"go-cf-zone-switch/pkg/config"
	"log"
	"os"
)

// "flag"
// "fmt"
// "go-cf-zone-switch/pkg/servers"
// "strings"
// "time"


func main() {
	cfgPath := flag.String("config-path", "config.toml", "Set path of toml file with config ")

	// help := flag.Bool("help", false, "--help")

	flag.Parse()
	cfg, err := config.Load(*cfgPath)

	log.Printf("%+v %v", cfg, err)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	remote := at.NewRemoteRepository(cfg.At)

	domains, err := remote.GetAllDomains()
	
	

	if err != nil {
		log.Printf("%+v", err)
	}
	
	log.Printf("%+v", domains)
	
	// mainHost := flag.String("main-host", "", "")

	// flag.Parse()
	

	// spl := strings.Split((*mainHost), ":")
	// fmt.Println(spl)
	// ip, port := spl[0], spl[1]

	// for {

	// 	time.Sleep(time.Second)

	// 	// ip := "176.9.70.13"
	// 	// port := "80"

	// 	ok, err := servers.IsServerReachable(ip, port, time.Second * 5)


	// 	fmt.Printf("Requesting host %s:%s ok:%t %v \n", ip, port, ok, err)

	// }

}