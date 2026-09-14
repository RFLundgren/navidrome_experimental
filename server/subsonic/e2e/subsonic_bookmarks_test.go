package e2e

import (
	"fmt"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bookmark and PlayQueue Endpoints", Ordered, func() {
	BeforeAll(func() {
		setupTestDB()
	})

	Describe("Bookmark Endpoints", Ordered, func() {
		var trackID string

		BeforeAll(func() {
			// Get a media file ID from the database to use for bookmarks
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).ToNot(BeEmpty())
			trackID = mfs[0].ID
		})

		It("getBookmarks returns empty initially", func() {
			resp := doReq("getBookmarks")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Bookmarks).ToNot(BeNil())
			Expect(resp.Bookmarks.Bookmark).To(BeEmpty())
		})

		It("createBookmark creates a bookmark with position", func() {
			resp := doReq("createBookmark", "id", trackID, "position", "12345", "comment", "test bookmark")

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("getBookmarks shows the created bookmark", func() {
			resp := doReq("getBookmarks")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Bookmarks).ToNot(BeNil())
			Expect(resp.Bookmarks.Bookmark).To(HaveLen(1))

			bmk := resp.Bookmarks.Bookmark[0]
			Expect(bmk.Entry.Id).To(Equal(trackID))
			Expect(bmk.Position).To(Equal(int64(12345)))
			Expect(bmk.Comment).To(Equal("test bookmark"))
			Expect(bmk.Username).To(Equal(adminUser.UserName))
		})

		It("deleteBookmark removes the bookmark", func() {
			resp := doReq("deleteBookmark", "id", trackID)

			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify it's gone
			resp = doReq("getBookmarks")
			Expect(resp.Bookmarks.Bookmark).To(BeEmpty())
		})
	})

	// Bookmarks on a podcast episode exercise the same generic bookmark mechanism as a song, but
	// through a different repository - the regression coverage for decoupling bookmarks from
	// MediaFile. Episodes are seeded directly against the datastore (bypassing an actual RSS
	// fetch) the same way createUser seeds users directly, rather than going through
	// createPodcastChannel.view.
	Describe("Podcast Episode Bookmark Endpoints", Ordered, func() {
		var trackID, episodeID string

		BeforeAll(func() {
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).ToNot(BeEmpty())
			trackID = mfs[0].ID

			channel := &model.PodcastChannel{Url: "https://example.com/e2e-feed.xml", Title: "E2E Test Feed"}
			Expect(ds.PodcastChannel(ctx).Put(channel)).To(Succeed())

			episode := &model.PodcastEpisode{ChannelID: channel.ID, Title: "E2E Episode", Guid: "e2e-guid-1", Duration: 600}
			Expect(ds.PodcastEpisode(ctx).Put(episode)).To(Succeed())
			episodeID = episode.ID
		})

		It("getBookmarks returns both a song and a podcast episode bookmark, correctly typed", func() {
			Expect(doReq("createBookmark", "id", trackID, "position", "1000", "comment", "song bookmark").Status).To(Equal(responses.StatusOK))
			Expect(doReq("createBookmark", "id", episodeID, "position", "30000", "comment", "episode bookmark").Status).To(Equal(responses.StatusOK))

			resp := doReq("getBookmarks")
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Bookmarks).ToNot(BeNil())
			Expect(resp.Bookmarks.Bookmark).To(HaveLen(2))

			byID := map[string]responses.Bookmark{}
			for _, bmk := range resp.Bookmarks.Bookmark {
				byID[bmk.Entry.Id] = bmk
			}
			Expect(byID[trackID].Comment).To(Equal("song bookmark"))
			Expect(byID[episodeID].Comment).To(Equal("episode bookmark"))
			Expect(byID[episodeID].Entry.Title).To(Equal("E2E Episode"))
			Expect(byID[episodeID].Position).To(Equal(int64(30000)))

			Expect(doReq("deleteBookmark", "id", trackID).Status).To(Equal(responses.StatusOK))
			Expect(doReq("deleteBookmark", "id", episodeID).Status).To(Equal(responses.StatusOK))
			Expect(doReq("getBookmarks").Bookmarks.Bookmark).To(BeEmpty())
		})

		It("reports the episode played only once position crosses the completion threshold", func() {
			ep, err := ds.PodcastEpisode(ctx).Get(episodeID)
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.PlayCount).To(Equal(int64(0)), "episode should not start out already played")

			Expect(doReq("createBookmark", "id", episodeID, "position", "5000").Status).To(Equal(responses.StatusOK))
			ep, err = ds.PodcastEpisode(ctx).Get(episodeID)
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.PlayCount).To(Equal(int64(0)), "a position far short of the episode's duration must not count as a play")

			Expect(doReq("createBookmark", "id", episodeID, "position", "550000").Status).To(Equal(responses.StatusOK)) // ~92% of 600s
			ep, err = ds.PodcastEpisode(ctx).Get(episodeID)
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.PlayCount).To(Equal(int64(1)), "crossing the completion threshold should mark the episode played")

			Expect(doReq("deleteBookmark", "id", episodeID).Status).To(Equal(responses.StatusOK))
		})
	})

	Describe("PlayQueue Endpoints", Ordered, func() {
		var trackIDs []string

		BeforeAll(func() {
			// Get multiple media file IDs from the database
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 3, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(mfs)).To(BeNumerically(">=", 2))
			for _, mf := range mfs {
				trackIDs = append(trackIDs, mf.ID)
			}
		})

		It("getPlayQueue returns minimum required fields when nothing specified", func() {
			resp := doReq("getPlayQueue")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.PlayQueue).ToNot(BeNil())
			Expect(resp.PlayQueue.Entry).To(HaveLen(0))
			Expect(resp.PlayQueue.Current).To(BeEmpty())
			Expect(resp.PlayQueue.Position).To(Equal(int64(0)))
			Expect(resp.PlayQueue.Username).To(Equal(adminUser.UserName))
			Expect(resp.PlayQueue.ChangedBy).To(BeEmpty())
		})

		It("getPlayQueueByIndex returns minimum required fields when nothing specified", func() {
			resp := doReq("getPlayQueueByIndex")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.PlayQueueByIndex).ToNot(BeNil())
			Expect(resp.PlayQueueByIndex.Entry).To(HaveLen(0))
			Expect(resp.PlayQueueByIndex.CurrentIndex).To(BeNil())
			Expect(resp.PlayQueueByIndex.Position).To(Equal(int64(0)))
			Expect(resp.PlayQueueByIndex.Username).To(Equal(adminUser.UserName))
			Expect(resp.PlayQueueByIndex.ChangedBy).To(BeEmpty())
		})

		It("savePlayQueue stores current play queue", func() {
			resp := doReq("savePlayQueue",
				"id", trackIDs[0],
				"id", trackIDs[1],
				"current", trackIDs[1],
				"position", "5000",
			)

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("getPlayQueue returns saved queue with tracks", func() {
			resp := doReq("getPlayQueue")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.PlayQueue).ToNot(BeNil())
			Expect(resp.PlayQueue.Entry).To(HaveLen(2))
			Expect(resp.PlayQueue.Current).To(Equal(trackIDs[1]))
			Expect(resp.PlayQueue.Position).To(Equal(int64(5000)))
			Expect(resp.PlayQueue.Username).To(Equal(adminUser.UserName))
			Expect(resp.PlayQueue.ChangedBy).To(Equal("test-client"))
		})

		It("getPlayQueueByIndex returns data with current index", func() {
			resp := doReq("getPlayQueueByIndex")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.PlayQueueByIndex).ToNot(BeNil())
			Expect(resp.PlayQueueByIndex.Entry).To(HaveLen(2))
			Expect(resp.PlayQueueByIndex.CurrentIndex).ToNot(BeNil())
			Expect(*resp.PlayQueueByIndex.CurrentIndex).To(Equal(1))
			Expect(resp.PlayQueueByIndex.Position).To(Equal(int64(5000)))
		})

		It("savePlayQueueByIndex stores queue by index", func() {
			resp := doReq("savePlayQueueByIndex",
				"id", trackIDs[0],
				"id", trackIDs[1],
				"id", trackIDs[2],
				"currentIndex", fmt.Sprintf("%d", 0),
				"position", "9999",
			)

			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify with getPlayQueueByIndex
			resp = doReq("getPlayQueueByIndex")
			Expect(resp.PlayQueueByIndex).ToNot(BeNil())
			Expect(resp.PlayQueueByIndex.Entry).To(HaveLen(3))
			Expect(resp.PlayQueueByIndex.CurrentIndex).ToNot(BeNil())
			Expect(*resp.PlayQueueByIndex.CurrentIndex).To(Equal(0))
			Expect(resp.PlayQueueByIndex.Position).To(Equal(int64(9999)))
		})
	})
})
