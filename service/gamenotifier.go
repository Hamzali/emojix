package service

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"slices"
	"time"
)

type GameNotifier interface {
	Pub(gameID string, userID string, notif GameNotification)
	PubAll(gameID string, notif GameNotification)
	Sub(gameID string, userID string) (chan GameNotification, func())
	Subs(gameID string) []string
}

type GameNotification interface {
	GetType() string
	GetData() string
	ParseData(data string) error
}

func generateRandomID() string {
	// Create a byte slice of size 16 (128 bits)
	bytes := make([]byte, 16)

	// Fill the byte slice with random values
	_, err := rand.Read(bytes)
	if err != nil {
		log.Println("failed to gen rand bytes")
		return ""
	}

	// Encode the bytes to a hexadecimal string
	id := hex.EncodeToString(bytes)

	return id
}

type gameSub struct {
	SubID     string
	UserID    string
	GameID    string
	NotifChan chan GameNotification
	LastMsgAt time.Time
}
type gameNotifier struct {
	subs []gameSub
}

func (gn *gameNotifier) Subs(gameID string) []string {
	subs := []string{}

	for _, sub := range gn.subs {
		if sub.GameID != gameID {
			continue
		}

		hasUserID := slices.Contains(subs, sub.UserID)

		if hasUserID {
			continue
		}

		subs = append(subs, sub.UserID)

	}

	return subs
}

func NewGameNotifier() GameNotifier {
	// we need to listen for channel in order to sub/unsub people as well
	// also think about one channel to funnel all the messages and filter at the sub side
	return &gameNotifier{subs: []gameSub{}}
}

func (gn *gameNotifier) Sub(gameID string, userID string) (chan GameNotification, func()) {
	ch := make(chan GameNotification)
	subID := generateRandomID()
	gs := gameSub{subID, userID, gameID, ch, time.Now()}
	gn.subs = append(gn.subs, gs)

	return ch, func() {
		gn.subs = slices.DeleteFunc(gn.subs, func(s gameSub) bool {
			return s.SubID == subID
		})
	}
}

func (gn *gameNotifier) Pub(gameID string, userID string, notif GameNotification) {
	for _, s := range gn.subs {
		if s.GameID != gameID || s.UserID == userID {
			continue
		}

		s.NotifChan <- notif
	}
}

func (gn *gameNotifier) PubAll(gameID string, notif GameNotification) {
	for _, s := range gn.subs {
		if s.GameID != gameID {
			continue
		}

		s.NotifChan <- notif
	}
}
