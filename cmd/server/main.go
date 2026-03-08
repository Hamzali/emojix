package main

import (
	"emojix"
	"emojix/repository"
	"emojix/service"
	"emojix/usecase"
	"fmt"
	"log"
	"net"
)

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Println(err)
		return ""
	}

	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func main() {
	localIP := getLocalIP()
	fmt.Println("server is runnning on http://localhost:9000...")
	fmt.Printf("server is runnning on http://%s:9000...\n", localIP)
	db, err := repository.InitSqliteDB("emojix.db")
	if err != nil {
		log.Fatalln(err)
	}

	userRepo := repository.NewUserRepository(db)
	gameRepo := repository.NewGameRepository(db)
	wordRepo := repository.NewWordRepository(db)
	unitOfWorkFactory := repository.NewUnitOfWorkFactory(db)

	gameNotifier := service.NewGameNotifier()

	emojixUsecase := usecase.NewEmojixUsecase(
		userRepo,
		gameRepo,
		wordRepo,
		unitOfWorkFactory,
		gameNotifier,
	)

	view := emojix.NewHTMLView()

	e, err := emojix.NewWebServer(emojixUsecase, view)

	if err != nil {
		log.Printf("failed to init err: %v", err)
		return
	}

	e.Start()
}
