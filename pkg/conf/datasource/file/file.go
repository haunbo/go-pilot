// Package file provides a filesystem backed configuration source.
package file

import "github.com/haunbo/go-pilot/pkg/conf"

// Scheme is the configuration scheme name.
const Scheme = "file"

// DataSource is a filesystem configuration source.
type DataSource = conf.DataSource

// New creates a file data source. When watch is true the parent directory is
// monitored and changes are reported through IsConfigChanged.
func New(path string, watch bool) (conf.DataSource, error) {
	return conf.NewFileDataSource(path, watch)
}
