package service_test

import (
	"emojix/service"
	"fmt"
	"testing"
)

type testNotif struct {
	notiftype string
	content   string
}

func (t testNotif) GetData() string {
	return t.content
}

func (t testNotif) GetType() string {
	return t.notiftype
}

func TestGameNotifierPubSub(t *testing.T) {
	notifier := service.NewGameNotifier()

	subCh := notifier.Sub("some-game-id", "some-user-id")

	// important note if you don't want to block publishing process use go routines!
	go func() {
		notifier.Pub("some-game-id", "other-user-id", testNotif{notiftype: "test-1", content: "content-1"})
		notifier.Pub("some-game-id", "other-user-id", testNotif{notiftype: "test-2", content: "content-2"})
		notifier.Pub("some-game-id", "other-user-id", testNotif{notiftype: "test-3", content: "content-3"})
		close(subCh)
	}()

	msgCounter := 0
	for msg := range subCh {
		msgCounter += 1

		expectedNotifType := fmt.Sprintf("test-%d", msgCounter)
		notifType := msg.GetType()
		if expectedNotifType != notifType {
			t.Errorf("expected %s notif type but got %s", expectedNotifType, notifType)
		}
	}

	if 3 != msgCounter {
		t.Errorf("expected to receive 3 messages but got %d", msgCounter)

	}
}

func TestGameNotifierSubs(t *testing.T) {

	notifier := service.NewGameNotifier()
	_ = notifier.Sub("some-game-id", "user-1")
	_ = notifier.Sub("some-game-id", "user-2")
	_ = notifier.Sub("some-game-id", "user-3")
	_ = notifier.Sub("other-game-id", "user-4")

	subs := notifier.Subs("some-game-id")

	if len(subs) != 3 {
		t.Errorf("expected to have 3 subscribers but got %d", len(subs))
	}

	expectedUser := "user-1"
	if subs[0] != expectedUser {
		t.Errorf("expected user '%s' but got %s", expectedUser, subs[0])
	}

	expectedUser = "user-2"
	if subs[1] != expectedUser {
		t.Errorf("expected user '%s' but got %s", expectedUser, subs[1])
	}

	expectedUser = "user-3"
	if subs[2] != expectedUser {
		t.Errorf("expected user '%s' but got %s", expectedUser, subs[2])
	}
}
