package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keybase/client/go/chat/storage"
	"github.com/keybase/client/go/engine"
	"github.com/keybase/client/go/kbtest"
	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/protocol/chat1"
	"github.com/keybase/client/go/protocol/gregor1"
	"github.com/keybase/client/go/protocol/keybase1"
	"github.com/stretchr/testify/require"
)

func TestChatBackgroundIdentify(t *testing.T) {

	world, _, _, _, listener, _, _ := setupTest(t, 2)
	defer world.Cleanup()

	u := world.GetUsers()[0]
	u1 := world.GetUsers()[1]
	tc := world.Tcs[u.Username]

	inbox := storage.NewInbox(tc.G, u.User.GetUID().ToBytes(), func() libkb.SecretUI {
		return &libkb.TestSecretUI{}
	})

	tlfName := u.Username
	msg := chat1.MessageBoxed{
		ClientHeader: chat1.MessageClientHeader{
			TlfName:     tlfName,
			Sender:      u.User.GetUID().ToBytes(),
			MessageType: chat1.MessageType_TEXT,
		},
	}
	conv := chat1.Conversation{
		Metadata: chat1.ConversationMetadata{
			ActiveList: []gregor1.UID{u.User.GetUID().ToBytes()},
		},
		MaxMsgs: []chat1.MessageBoxed{msg},
	}
	require.NoError(t, inbox.Merge(context.TODO(), 1, []chat1.Conversation{conv}, nil, nil))

	handler := NewIdentifyChangedHandler(tc.G, func() keybase1.TlfInterface {
		return kbtest.NewTlfMock(world)
	})
	require.NotNil(t, handler.G().NotifyRouter, "notify router")

	t.Logf("new error job in inbox")
	job := engine.NewIdentifyJob(u.User.GetUID(), errors.New("AHHHHHHH"), nil)
	go handler.BackgroundIdentifyChanged(context.TODO(), job)
	select {
	case update := <-listener.identifyUpdate:
		require.Equal(t, update.CanonicalName.String(), tlfName, "wrong tlf name")
		require.NotZero(t, len(update.Breaks.Breaks), "no breaks")
	case <-time.After(2 * time.Second):
		require.Fail(t, "no identify update received")
	}

	t.Logf("new error job not in inbox")
	job = engine.NewIdentifyJob(u1.User.GetUID(), errors.New("AHHHHHHH"), nil)
	handler.BackgroundIdentifyChanged(context.TODO(), job)
	select {
	case <-listener.identifyUpdate:
		require.Fail(t, "not supposed to get update")
	default:
	}

	t.Logf("cleared error in inbox")
	job = engine.NewIdentifyJob(u.User.GetUID(), nil, errors.New("AHHHHHHH"))
	go handler.BackgroundIdentifyChanged(context.TODO(), job)
	select {
	case update := <-listener.identifyUpdate:
		require.Equal(t, update.CanonicalName.String(), tlfName, "wrong tlf name")
		require.Zero(t, len(update.Breaks.Breaks), "breaks")
	case <-time.After(20 * time.Second):
		require.Fail(t, "no identify update received")
	}

}
