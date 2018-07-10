package helm

import (
	"github.com/golang/glog"
	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

// SearchReleases searches releases from all repositories.
func (c *Client) SearchReleases() ([]*search.Result, error) {
	index, err := buildIndex(c.HelmHome)
	if err != nil {
		return nil, err
	}

	var res []*search.Result
	res = index.All()

	search.SortScore(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func buildIndex(home helmpath.Home) (*search.Index, error) {
	rf, err := repo.LoadRepositoriesFile(home.RepositoryFile())
	if err != nil {
		return nil, err
	}

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		f := home.CacheIndex(n)
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			glog.Warningf("repository %q is corrupt or missing.", n)
			continue
		}
		i.AddRepo(n, ind, false)
	}

	return i, nil
}
