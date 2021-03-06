package broker

import (
	"flag"
)

// Options holds the options specified by the broker's code on the command
// line. Users should add their own options here and add flags for them in
// AddFlags.
type Options struct {
	CatalogPath string
	Async       bool
	TillerHost  string
	HelmHome    string
}

// AddFlags is a hook called to initialize the CLI flags for broker options.
// It is called after the flags are added for the skeleton and before flag
// parse is called.
func AddFlags(o *Options) {
	flag.StringVar(&o.CatalogPath, "catalogPath", "", "The path to the catalog")
	flag.BoolVar(&o.Async, "async", false, "Indicates whether the broker is handling the requests asynchronously.")
	flag.StringVar(&o.TillerHost, "tillerHost", "", "The host and port of Tiller")
	flag.StringVar(&o.HelmHome, "helmHome", "", "The local path to the Helm home directory")
}
