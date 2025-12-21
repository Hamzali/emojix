package service

import (
	"time"
)

type GameNotifier interface {
	Pub(gameID string, userID string, notif GameNotification)
	Sub(gameID string, userID string) chan GameNotification
	Subs(gameID string) []string
	Unsub(userID string)
}

type GameNotification interface {
	GetType() string
	GetData() string
}

type gameSub struct {
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
		subs = append(subs, sub.UserID)
	}

	return subs
}

func NewGameNotifier() GameNotifier {
	// we need to listen for channel in order to sub/unsub people as well
	// also think about one channel to funnel all the messages and filter at the sub side
	return &gameNotifier{subs: []gameSub{}}
}

func (gn *gameNotifier) Sub(gameID string, userID string) chan GameNotification {
	for _, sub := range gn.subs {
		if sub.GameID == gameID && sub.UserID == userID {
			return sub.NotifChan
		}
	}

	ch := make(chan GameNotification)
	gameSub := gameSub{userID, gameID, ch, time.Now()}
	gn.subs = append(gn.subs, gameSub)
	return ch
}

// TODO: test this shit so you don't make this kind of errors
func (gn *gameNotifier) Unsub(userID string) {
	newSubs := []gameSub{}
	for _, s := range gn.subs {
		if s.UserID == userID {
			continue
		}
		newSubs = append(newSubs, s)
	}
	gn.subs = newSubs
}

func (gn *gameNotifier) Pub(gameID string, userID string, notif GameNotification) {
	for _, s := range gn.subs {
		if s.GameID != gameID || s.UserID == userID {
			continue
		}

		s.NotifChan <- notif
	}
}
