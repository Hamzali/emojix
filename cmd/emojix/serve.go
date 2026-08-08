package main

import (
	"emojix"
	"emojix/repository"
	"emojix/service"
	"emojix/usecase"
	"flag"
	"fmt"
	"log"
	"net"
)

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	dbName := fs.String("db", "emojix.db", "sqlite file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	localIP := getLocalIP()
	fmt.Println("server running on http://localhost:9000...")
	if localIP != "" {
		fmt.Printf("server running on http://%s:9000...\n", localIP)
	}

	db, err := repository.InitSqliteDB(*dbName)
	if err != nil {
		return err
	}

	emojix.NewWebServer(
		usecase.NewEmojixUsecase(
			repository.NewUserRepository(db),
			repository.NewGameRepository(db),
			repository.NewWordRepository(db),
			repository.NewUnitOfWorkFactory(db),
			service.NewGameNotifier(),
			service.NewGameLoop(service.NewRealClock()),
			service.NewRealClock(),
		),
		emojix.NewHTMLView(),
	).Start()
	return nil
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Println(err)
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
