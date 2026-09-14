package podcasts

import (
	"context"
	"testing"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
)

func newTestPodcasts() (*podcasts, *tests.MockedPodcastChannelRepo, *tests.MockedPodcastSubscriptionRepo) {
	ds := &tests.MockDataStore{}
	channelRepo := tests.CreateMockedPodcastChannelRepo()
	ds.MockedPodcastChannel = channelRepo
	subRepo := tests.CreateMockedPodcastSubscriptionRepo()
	ds.MockedPodcastSubscription = subRepo
	ds.MockedPodcastEpisode = tests.CreateMockedPodcastEpisodeRepo()
	return New(ds, nil).(*podcasts), channelRepo, subRepo
}

func newTestPodcastsWithEpisode(episode *model.PodcastEpisode) (*podcasts, *tests.MockedPodcastEpisodeRepo) {
	ds := &tests.MockDataStore{}
	ds.MockedPodcastChannel = tests.CreateMockedPodcastChannelRepo()
	ds.MockedPodcastSubscription = tests.CreateMockedPodcastSubscriptionRepo()
	episodeRepo := tests.CreateMockedPodcastEpisodeRepo()
	episodeRepo.Data[episode.ID] = episode
	ds.MockedPodcastEpisode = episodeRepo
	return New(ds, nil).(*podcasts), episodeRepo
}

// A position short of the completion threshold must not count as a play - the whole point of
// switching away from the old "mark listened the instant streaming starts" behavior.
func TestRecordEpisodePositionBelowThresholdDoesNotMarkListened(t *testing.T) {
	episode := &model.PodcastEpisode{ID: "ep-1", Duration: 600} // 10 minutes
	svc, episodeRepo := newTestPodcastsWithEpisode(episode)
	ctx := request.WithUser(context.Background(), model.User{ID: "user-1"})

	justCompleted, err := svc.RecordEpisodePosition(ctx, episode.ID, 5000) // 5s in
	if err != nil {
		t.Fatalf("RecordEpisodePosition returned error: %v", err)
	}
	if justCompleted {
		t.Fatalf("expected justCompleted=false for a position far short of the episode's duration")
	}
	if episodeRepo.Data[episode.ID].PlayCount != 0 {
		t.Fatalf("expected PlayCount to remain 0, got %d", episodeRepo.Data[episode.ID].PlayCount)
	}
}

// A position crossing the completion threshold marks the episode listened, and reports
// justCompleted=true so the caller knows to fire any completion-triggered side effects.
func TestRecordEpisodePositionAtThresholdMarksListened(t *testing.T) {
	episode := &model.PodcastEpisode{ID: "ep-1", Duration: 600} // 10 minutes = 600,000ms
	svc, episodeRepo := newTestPodcastsWithEpisode(episode)
	ctx := request.WithUser(context.Background(), model.User{ID: "user-1"})

	justCompleted, err := svc.RecordEpisodePosition(ctx, episode.ID, 550000) // ~92%
	if err != nil {
		t.Fatalf("RecordEpisodePosition returned error: %v", err)
	}
	if !justCompleted {
		t.Fatalf("expected justCompleted=true once position crosses the completion threshold")
	}
	if episodeRepo.Data[episode.ID].PlayCount != 1 {
		t.Fatalf("expected PlayCount=1, got %d", episodeRepo.Data[episode.ID].PlayCount)
	}
}

// Repeated position reports past the threshold (e.g. the client keeps sending updates as the
// episode finishes) must not keep incrementing PlayCount or keep reporting justCompleted.
func TestRecordEpisodePositionIsIdempotentOnceListened(t *testing.T) {
	episode := &model.PodcastEpisode{ID: "ep-1", Duration: 600}
	svc, episodeRepo := newTestPodcastsWithEpisode(episode)
	ctx := request.WithUser(context.Background(), model.User{ID: "user-1"})

	if _, err := svc.RecordEpisodePosition(ctx, episode.ID, 550000); err != nil {
		t.Fatalf("first RecordEpisodePosition returned error: %v", err)
	}
	justCompleted, err := svc.RecordEpisodePosition(ctx, episode.ID, 590000)
	if err != nil {
		t.Fatalf("second RecordEpisodePosition returned error: %v", err)
	}
	if justCompleted {
		t.Fatalf("expected justCompleted=false on a repeat report after the episode is already listened")
	}
	if episodeRepo.Data[episode.ID].PlayCount != 1 {
		t.Fatalf("expected PlayCount to stay at 1, got %d", episodeRepo.Data[episode.ID].PlayCount)
	}
}

// An episode with no known duration (some feeds omit <itunes:duration>) can't have a completion
// ratio computed - position reports must never mark it listened.
func TestRecordEpisodePositionSkipsUnknownDuration(t *testing.T) {
	episode := &model.PodcastEpisode{ID: "ep-1", Duration: 0}
	svc, episodeRepo := newTestPodcastsWithEpisode(episode)
	ctx := request.WithUser(context.Background(), model.User{ID: "user-1"})

	justCompleted, err := svc.RecordEpisodePosition(ctx, episode.ID, 999999999)
	if err != nil {
		t.Fatalf("RecordEpisodePosition returned error: %v", err)
	}
	if justCompleted {
		t.Fatalf("expected justCompleted=false for an episode with unknown duration")
	}
	if episodeRepo.Data[episode.ID].PlayCount != 0 {
		t.Fatalf("expected PlayCount to remain 0, got %d", episodeRepo.Data[episode.ID].PlayCount)
	}
}

// A non-admin subscriber subscribing to a feed that some other user already subscribes to must
// reuse the existing shared channel row, not create a second one - the whole point of the shared-
// channel model is that a feed is only ever fetched/stored once regardless of subscriber count.
func TestSubscribeReusesExistingSharedChannel(t *testing.T) {
	svc, channelRepo, subRepo := newTestPodcasts()
	existing := &model.PodcastChannel{ID: "existing-channel", Url: "https://example.com/feed.xml"}
	channelRepo.Data[existing.ID] = existing

	ctx := request.WithUser(context.Background(), model.User{ID: "user-2", IsAdmin: false})

	channel, err := svc.Subscribe(ctx, existing.Url)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if channel.ID != existing.ID {
		t.Fatalf("expected Subscribe to reuse existing channel %q, got %q", existing.ID, channel.ID)
	}
	if len(channelRepo.Data) != 1 {
		t.Fatalf("expected exactly 1 channel to exist, got %d", len(channelRepo.Data))
	}

	sub, err := subRepo.FindByChannelAndUser(existing.ID, "user-2")
	if err != nil {
		t.Fatalf("expected a subscription to be created for user-2: %v", err)
	}
	if sub.ChannelID != existing.ID {
		t.Fatalf("subscription points at wrong channel: %q", sub.ChannelID)
	}
}

// Subscribing twice for the same user/channel is idempotent - it must not create a second
// subscription row.
func TestSubscribeIsIdempotent(t *testing.T) {
	svc, channelRepo, subRepo := newTestPodcasts()
	existing := &model.PodcastChannel{ID: "existing-channel", Url: "https://example.com/feed.xml"}
	channelRepo.Data[existing.ID] = existing

	ctx := request.WithUser(context.Background(), model.User{ID: "user-2", IsAdmin: false})

	if _, err := svc.Subscribe(ctx, existing.Url); err != nil {
		t.Fatalf("first Subscribe returned error: %v", err)
	}
	if _, err := svc.Subscribe(ctx, existing.Url); err != nil {
		t.Fatalf("second Subscribe returned error: %v", err)
	}

	count := 0
	for _, s := range subRepo.Data {
		if s.ChannelID == existing.ID && s.UserID == "user-2" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 subscription for user-2, got %d", count)
	}
}

// Unsubscribing the last remaining subscriber must tear down the now-orphaned shared channel.
func TestUnsubscribeLastSubscriberDeletesChannel(t *testing.T) {
	svc, channelRepo, subRepo := newTestPodcasts()
	existing := &model.PodcastChannel{ID: "existing-channel", Url: "https://example.com/feed.xml"}
	channelRepo.Data[existing.ID] = existing
	sub := &model.PodcastSubscription{ID: "sub-1", ChannelID: existing.ID, UserID: "user-2"}
	if err := subRepo.Put(sub); err != nil {
		t.Fatalf("seeding subscription failed: %v", err)
	}

	ctx := request.WithUser(context.Background(), model.User{ID: "user-2", IsAdmin: false})

	if err := svc.Unsubscribe(ctx, existing.ID); err != nil {
		t.Fatalf("Unsubscribe returned error: %v", err)
	}
	if _, ok := channelRepo.Data[existing.ID]; ok {
		t.Fatalf("expected orphaned channel to be deleted")
	}
}
