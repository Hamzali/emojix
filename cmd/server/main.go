package main

import (
	"emojix"
	"emojix/repository"
	"emojix/service"
	"emojix/usecase"
	"fmt"
	"log"
)

func main() {
	fmt.Println("server is runnning on 9000...")
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

	e, err := emojix.NewWebServer(emojixUsecase)

	if err != nil {
		log.Printf("failed to init err: %v", err)
		return
	}

	e.Start()
}
