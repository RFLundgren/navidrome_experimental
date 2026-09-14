package artwork

import (
	"context"
	"io"
)

// resolvePodcastChannel serves an admin-uploaded channel image if present, falling back to the
// feed's own <itunes:image>/<image> URL - so a subscribed channel without a manual override still
// shows real cover art instead of a placeholder. The remote fetch is worker-only (r.ext == nil for
// a synchronous request), matching resolvePlaylist's ExternalImageURL handling: a request never
// blocks on an outbound fetch to a feed host, only the background worker does.
func (r *resolver) resolvePodcastChannel(ctx context.Context, channelID string) (resolution, error) {
	channel, err := r.ds.PodcastChannel(ctx).Get(channelID)
	if err != nil {
		return resolution{}, err
	}
	if res, ok := resolveLocalFile(channel.UploadedImagePath(), "upload"); ok || res.localError {
		return res, nil
	}
	if r.ext == nil {
		return resolution{}, nil
	}
	localImg, remoteImg := classifyPlaylistImage(channel.CoverArtUrl)
	if localImg != "" {
		if res, ok := resolveLocalFile(localImg, "folder"); ok || res.localError {
			return res, nil
		}
	}
	if remoteImg == nil {
		return resolution{}, nil
	}
	sf := func() (io.ReadCloser, string, error) { return fromURL(ctx, remoteImg) }
	if res, ok, err := resolveExternalStep(r.ext.gate, "external", sf); ok {
		return res, nil
	} else if err != nil {
		return resolution{extErr: err}, nil
	}
	return resolution{}, nil
}
