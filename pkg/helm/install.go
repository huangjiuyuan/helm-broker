package helm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
)

// InstallRelease loads a chart, installs it, and returns the release response.
func (c *Client) InstallRelease(chart string, namespace string, name string, values map[string]interface{}) (*services.InstallReleaseResponse, error) {
	rawValues, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}

	chartPath, err := locateChartPath("", "", "", chart, "", false, "", "", "", "", c.settings)
	if err != nil {
		return nil, err
	}

	chartRequested, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, prettyError(err)
	}

	resp, err := c.client.InstallReleaseFromChart(
		chartRequested,
		"default",
		helm.ReleaseName(name),
		helm.ValueOverrides(rawValues),
		helm.InstallTimeout(300),
	)
	if err != nil {
		return nil, prettyError(err)
	}

	return resp, nil
}

// locateChartPath looks for a chart directory in known places, and returns either the full path or an error.
func locateChartPath(repoURL, username, password, name, version string, verify bool, keyring,
	certFile, keyFile, caFile string, settings environment.EnvSettings) (string, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if fi, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		if verify {
			if fi.IsDir() {
				return "", errors.New("cannot verify a directory")
			}
			if _, err := downloader.VerifyChart(abs, keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, fmt.Errorf("path %q not found", name)
	}

	crepo := filepath.Join(settings.Home.Repository(), name)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Keyring:  keyring,
		Getters:  getter.All(settings),
		Username: username,
		Password: password,
	}
	if verify {
		dl.Verify = downloader.VerifyAlways
	}
	if repoURL != "" {
		chartURL, err := repo.FindChartInAuthRepoURL(repoURL, username, password, name, version,
			certFile, keyFile, caFile, getter.All(settings))
		if err != nil {
			return "", err
		}
		name = chartURL
	}

	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		os.MkdirAll(settings.Home.Archive(), 0744)
	}

	filename, _, err := dl.DownloadTo(name, version, settings.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		return lname, nil
	} else if settings.Debug {
		return filename, err
	}

	return filename, fmt.Errorf("failed to download %q", name)
}
