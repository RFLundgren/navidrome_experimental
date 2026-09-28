package subsonic

import (
	"context"
	"fmt"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func contextWithUser(ctx context.Context, userID string, libraryIDs ...int) context.Context {
	libraries := make([]model.Library, len(libraryIDs))
	for i, id := range libraryIDs {
		libraries[i] = model.Library{ID: id, Name: fmt.Sprintf("Test Library %d", id), Path: fmt.Sprintf("/music/library%d", id)}
	}
	user := model.User{
		ID:        userID,
		Libraries: libraries,
	}
	return request.WithUser(ctx, user)
}

var _ = Describe("Browsing", func() {
	var api *Router
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		api = &Router{ds: ds}
		ctx = context.Background()
	})

	Describe("GetMusicFolders", func() {
		It("should return all libraries the user has access", func() {
			// Create mock user with libraries
			ctx := contextWithUser(ctx, "user-id", 1, 2, 3)

			// Create request
			r := httptest.NewRequest("GET", "/rest/getMusicFolders", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetMusicFolders(r)

			// Verify results
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.MusicFolders).ToNot(BeNil())
			Expect(response.MusicFolders.Folders).To(HaveLen(3))
			Expect(response.MusicFolders.Folders[0].Name).To(Equal("Test Library 1"))
			Expect(response.MusicFolders.Folders[1].Name).To(Equal("Test Library 2"))
			Expect(response.MusicFolders.Folders[2].Name).To(Equal("Test Library 3"))
		})
	})

	Describe("GetIndexes", func() {
		It("should expose physical top-level folders when Subsonic folder browsing is enabled", func() {
			oldFolderBrowsing := conf.Server.Subsonic.FolderBrowsing
			conf.Server.Subsonic.FolderBrowsing = true
			DeferCleanup(func() { conf.Server.Subsonic.FolderBrowsing = oldFolderBrowsing })

			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})

			rock := model.NewFolder(lib, "Rock")
			game := model.NewFolder(lib, "Game")
			nested := model.NewFolder(lib, "Game/Final Fantasy")
			mockFolders := ds.Folder(ctx).(*tests.MockFolderRepo)
			mockFolders.Data = map[string]model.Folder{
				rock.ID:   *rock,
				game.ID:   *game,
				nested.ID: *nested,
			}

			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=1", nil).WithContext(ctx)
			response, err := api.GetIndexes(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Indexes).ToNot(BeNil())

			var entries []string
			var ids []string
			for _, index := range response.Indexes.Index {
				for _, entry := range index.Artists {
					entries = append(entries, entry.Name)
					ids = append(ids, entry.Id)
				}
			}
			Expect(entries).To(ConsistOf("Game", "Rock"))
			Expect(ids).To(ConsistOf(game.ID, rock.ID))
			Expect(entries).ToNot(ContainElement("Final Fantasy"))
		})

		It("should keep the legacy artist index when Subsonic folder browsing is disabled", func() {
			oldFolderBrowsing := conf.Server.Subsonic.FolderBrowsing
			conf.Server.Subsonic.FolderBrowsing = false
			DeferCleanup(func() { conf.Server.Subsonic.FolderBrowsing = oldFolderBrowsing })

			ctx = contextWithUser(ctx, "user-id", 1)
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{{ID: "artist-1", Name: "Test Artist"}})

			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=1", nil).WithContext(ctx)
			response, err := api.GetIndexes(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Indexes).ToNot(BeNil())
			var names []string
			for _, index := range response.Indexes.Index {
				for _, entry := range index.Artists {
					names = append(names, entry.Name)
				}
			}
			Expect(names).To(ContainElement("Test Artist"))
		})

		It("should validate user access to the specified musicFolderId", func() {
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=2 (not accessible)
			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=2", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			// Create mock user with access to libraries 2 and 3
			ctx = contextWithUser(ctx, "user-id", 2, 3)

			// Setup minimal mock library data for working tests
			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{
				{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
				{ID: 3, Name: "Test Library 3", Path: "/music/library3"},
			})

			// Setup mock artist data
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "1", Name: "Test Artist 1"},
				{ID: "2", Name: "Test Artist 2"},
			})

			// Create request without musicFolderId
			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should succeed and use first accessible library (2)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Indexes).ToNot(BeNil())
		})
	})

	Describe("GetArtistInfo", func() {
		It("should return an empty response for physical folder IDs", func() {
			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})

			game := model.NewFolder(lib, "Game")
			mockFolders := ds.Folder(ctx).(*tests.MockFolderRepo)
			mockFolders.Data = map[string]model.Folder{game.ID: *game}

			r := httptest.NewRequest("GET", "/rest/getArtistInfo?id="+game.ID, nil).WithContext(ctx)
			response, err := api.GetArtistInfo(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.ArtistInfo).ToNot(BeNil())
			Expect(response.ArtistInfo.Biography).To(BeEmpty())
			Expect(response.ArtistInfo.SimilarArtist).To(BeEmpty())
		})
	})

	Describe("Physical folder artist mapping", func() {
		It("should map an artist-named folder to the real artist", func() {
			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})

			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				{ID: "artist-tyketto", Name: "Tyketto"},
			})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).CountAllValue = 12

			folder := model.NewFolder(lib, "Rock/Tyketto")
			artistID, err := api.artistIDForPhysicalFolder(ctx, *folder)

			Expect(err).ToNot(HaveOccurred())
			Expect(artistID).To(Equal("artist-tyketto"))
		})

		It("should not map a category folder just because artists exist", func() {
			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})

			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				{ID: "artist-moby", Name: "Moby"},
			})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).CountAllValue = 10

			folder := model.NewFolder(lib, "Game")
			artistID, err := api.artistIDForPhysicalFolder(ctx, *folder)

			Expect(err).ToNot(HaveOccurred())
			Expect(artistID).To(BeEmpty())
		})

		It("should require the matching artist to have content in the folder subtree", func() {
			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})

			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				{ID: "artist-game", Name: "Game"},
			})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).CountAllValue = 0

			folder := model.NewFolder(lib, "Game")
			artistID, err := api.artistIDForPhysicalFolder(ctx, *folder)

			Expect(err).ToNot(HaveOccurred())
			Expect(artistID).To(BeEmpty())
		})

		It("should expose a resolved artist folder as an artist to DSub", func() {
			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})
			ctx = request.WithPlayer(ctx, model.Player{Client: "DSub"})

			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				{ID: "artist-aha", Name: "a-ha", SongCount: 42},
			})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).CountAllValue = 42

			folder := model.NewFolder(lib, "Pop/A-Ha")
			child := childFromFolder(ctx, *folder)
			api.applyPhysicalArtistMetadata(ctx, *folder, &child)

			Expect(child.ArtistId).To(Equal("artist-aha"))
			Expect(child.SongCount).To(Equal(int32(42)))
			Expect(child.Parent).To(BeEmpty())
			Expect(child.Artist).To(BeEmpty())
		})

		It("should reject ambiguous same-name artist matches", func() {
			lib := model.Library{ID: 1, Name: "Music", Path: "/music"}
			ctx = request.WithUser(ctx, model.User{ID: "user-id", Libraries: model.Libraries{lib}})

			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				{ID: "artist-one", Name: "Same Name"},
				{ID: "artist-two", Name: "Same Name"},
			})
			ds.MediaFile(ctx).(*tests.MockMediaFileRepo).CountAllValue = 1

			folder := model.NewFolder(lib, "Rock/Same Name")
			artistID, err := api.artistIDForPhysicalFolder(ctx, *folder)

			Expect(err).ToNot(HaveOccurred())
			Expect(artistID).To(BeEmpty())
		})
	})

	Describe("GetArtists", func() {
		It("should validate user access to the specified musicFolderId", func() {
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=3 (not accessible)
			r := httptest.NewRequest("GET", "/rest/getArtists?musicFolderId=3", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			// Create mock user with access to libraries 1 and 2
			ctx = contextWithUser(ctx, "user-id", 1, 2)

			// Setup minimal mock library data for working tests
			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{
				{ID: 1, Name: "Test Library 1", Path: "/music/library1"},
				{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
			})

			// Setup mock artist data
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "1", Name: "Test Artist 1"},
				{ID: "2", Name: "Test Artist 2"},
			})

			// Create request without musicFolderId
			r := httptest.NewRequest("GET", "/rest/getArtists", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should succeed and use first accessible library (1)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Artist).ToNot(BeNil())
		})
	})
})
