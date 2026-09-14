package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlBookmarks", func() {
	var mr model.MediaFileRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		mr = NewMediaFileRepository(ctx, GetDBXBuilder())
	})

	Describe("Bookmarks", func() {
		It("returns an empty collection if there are no bookmarks", func() {
			Expect(mr.GetBookmarks()).To(BeEmpty())
		})

		It("saves and overrides bookmarks", func() {
			By("Saving the bookmark")
			Expect(mr.AddBookmark(songAntenna.ID, "this is a comment", 123)).To(BeNil())

			bms, err := mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())

			Expect(bms).To(HaveLen(1))
			Expect(bms[0].Item.(model.MediaFile).ID).To(Equal(songAntenna.ID))
			Expect(bms[0].Item.(model.MediaFile).Title).To(Equal(songAntenna.Title))
			Expect(bms[0].Comment).To(Equal("this is a comment"))
			Expect(bms[0].Position).To(Equal(int64(123)))
			created := bms[0].CreatedAt
			updated := bms[0].UpdatedAt
			Expect(created.IsZero()).To(BeFalse())
			Expect(updated).To(BeTemporally(">=", created))

			By("Overriding the bookmark")
			Expect(mr.AddBookmark(songAntenna.ID, "another comment", 333)).To(BeNil())

			bms, err = mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())

			Expect(bms[0].Item.(model.MediaFile).ID).To(Equal(songAntenna.ID))
			Expect(bms[0].Comment).To(Equal("another comment"))
			Expect(bms[0].Position).To(Equal(int64(333)))
			Expect(bms[0].CreatedAt).To(Equal(created))
			Expect(bms[0].UpdatedAt).To(BeTemporally(">=", updated))

			By("Saving another bookmark")
			Expect(mr.AddBookmark(songComeTogether.ID, "one more comment", 444)).To(BeNil())
			bms, err = mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())
			Expect(bms).To(HaveLen(2))

			By("Delete bookmark")
			Expect(mr.DeleteBookmark(songAntenna.ID)).To(Succeed())
			bms, err = mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())
			Expect(bms).To(HaveLen(1))
			Expect(bms[0].Item.(model.MediaFile).ID).To(Equal(songComeTogether.ID))
			Expect(bms[0].Item.(model.MediaFile).Title).To(Equal(songComeTogether.Title))

			Expect(mr.DeleteBookmark(songComeTogether.ID)).To(Succeed())
			Expect(mr.GetBookmarks()).To(BeEmpty())
		})
	})
})

// Bookmarks on a podcast episode exercise the same generic bookmark table as a song, but through
// a different repository - this is the regression coverage for decoupling GetBookmarks from
// MediaFile (see model.Bookmark.Item's switch from a fixed MediaFile to `any`).
var _ = Describe("sqlBookmarks - podcast episodes", func() {
	var er model.PodcastEpisodeRepository
	var channel *model.PodcastChannel
	var episode *model.PodcastEpisode

	BeforeEach(func() {
		adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "admin-id", IsAdmin: true})
		userCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid"})

		channel = &model.PodcastChannel{Url: "https://example.com/feed.xml", Title: "Test Feed"}
		Expect(NewPodcastChannelRepository(adminCtx, GetDBXBuilder()).Put(channel)).To(Succeed())

		episode = &model.PodcastEpisode{ChannelID: channel.ID, Title: "Episode 1", Guid: "guid-1"}
		Expect(NewPodcastEpisodeRepository(adminCtx, GetDBXBuilder()).Put(episode)).To(Succeed())

		sub := &model.PodcastSubscription{ChannelID: channel.ID, UserID: "userid"}
		Expect(NewPodcastSubscriptionRepository(userCtx, GetDBXBuilder()).Put(sub)).To(Succeed())

		er = NewPodcastEpisodeRepository(userCtx, GetDBXBuilder())
	})

	AfterEach(func() {
		adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "admin-id", IsAdmin: true})
		_ = NewPodcastChannelRepository(adminCtx, GetDBXBuilder()).Delete(channel.ID)
	})

	It("returns an empty collection if there are no bookmarks", func() {
		Expect(er.GetBookmarks()).To(BeEmpty())
	})

	It("saves, lists, and deletes a podcast episode bookmark, typed as a PodcastEpisode", func() {
		Expect(er.AddBookmark(episode.ID, "resume here", 45000)).To(BeNil())

		bms, err := er.GetBookmarks()
		Expect(err).ToNot(HaveOccurred())
		Expect(bms).To(HaveLen(1))
		ep := bms[0].Item.(model.PodcastEpisode)
		Expect(ep.ID).To(Equal(episode.ID))
		Expect(ep.Title).To(Equal(episode.Title))
		Expect(bms[0].Comment).To(Equal("resume here"))
		Expect(bms[0].Position).To(Equal(int64(45000)))

		Expect(er.DeleteBookmark(episode.ID)).To(Succeed())
		Expect(er.GetBookmarks()).To(BeEmpty())
	})
})
