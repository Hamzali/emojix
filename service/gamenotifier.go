package service

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"slices"
	"sync"
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
	mu   sync.RWMutex
	subs []gameSub
}

func (gn *gameNotifier) Subs(gameID string) []string {
	gn.mu.RLock()
	defer gn.mu.RUnlock()

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

	gn.mu.Lock()
	gn.subs = append(gn.subs, gs)
	gn.mu.Unlock()

	return ch, func() {
		gn.mu.Lock()
		defer gn.mu.Unlock()
		gn.subs = slices.DeleteFunc(gn.subs, func(s gameSub) bool {
			return s.SubID == subID
		})
	}
}

func (gn *gameNotifier) Pub(gameID string, userID string, notif GameNotification) {
	gn.mu.RLock()
	targets := []gameSub{}
	for _, s := range gn.subs {
		if s.GameID != gameID || s.UserID == userID {
			continue
		}
		targets = append(targets, s)
	}
	gn.mu.RUnlock()

	for _, s := range targets {
		s.NotifChan <- notif
	}
}

func (gn *gameNotifier) PubAll(gameID string, notif GameNotification) {
	gn.mu.RLock()
	targets := []gameSub{}
	for _, s := range gn.subs {
		if s.GameID != gameID {
			continue
		}
		targets = append(targets, s)
	}
	gn.mu.RUnlock()

	for _, s := range targets {
		s.NotifChan <- notif
	}
}
