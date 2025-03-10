package devnet

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/github"
)

// Get assets of latest release or prerelease from GitHub
func GetAssetsFromLastReleaseGitHub(ctx context.Context, client *github.Client, namespace, repository string, tag string) ([]ReleaseAsset, error) {
	// List the tags of the GitHub repository
	slog.Debug("github: listing tags for", "namespace", namespace, "repository", repository)

	releases := make([]*github.RepositoryRelease, 0)
	releaseAssets := make([]ReleaseAsset, 0)

	if tag != "" {
		release, _, err := client.Repositories.GetReleaseByTag(ctx, namespace, repository, tag)

		if err != nil {
			return nil, fmt.Errorf("github: %s(%s): failed to get release %s", namespace, repository, err.Error())
		}

		releases = append(releases, release)
	} else {
		listReleases, _, err := client.Repositories.ListReleases(ctx, namespace, repository, &github.ListOptions{
			PerPage: 1,
		})

		// For stable releases
		// release, _, err := client.Repositories.GetLatestRelease(ctx, namespace, repository)

		if err != nil {
			return nil, fmt.Errorf("github: %s(%s): failed to list releases %s", namespace, repository, err.Error())
		}

		releases = append(releases, listReleases...)
	}

	for _, r := range releases {
		for _, a := range r.Assets {
			slog.Debug("github: asset", "tag", r.GetTagName(), "name", a.GetName(), "url", a.GetBrowserDownloadURL())
			releaseAssets = append(releaseAssets, ReleaseAsset{
				Tag:      r.GetTagName(),
				AssetId:  a.GetID(),
				Filename: a.GetName(),
				Url:      a.GetBrowserDownloadURL(),
			})
		}
	}

	return releaseAssets, nil
}
