package vulnerabilities

import (
	"github.com/A1bemuth/deputy"
	deptypes "github.com/A1bemuth/deputy/types"
	"github.com/AlexAkulov/hungryfox"
	"github.com/package-url/packageurl-go"
	"github.com/rs/zerolog"
)

type DepsSearcher struct {
	DepsChannel chan<- *hungryfox.Dependency
	Log         zerolog.Logger

	parser deptypes.SelectiveParser
}

func (a DepsSearcher) Process(diff *hungryfox.Diff) {
	if a.parser == nil {
		a.parser = deputy.NewParser()
	}
	if deps, err := a.parser.Parse(diff.FilePath, diff.Content); err == nil {
		for _, dep := range deps {
			dependency := hungryfox.Dependency{
				Purl: packageurl.PackageURL{
					Type:    dep.Ecosystem,
					Name:    dep.Name,
					Version: dep.Version,
				},
				Diff: *diff,
			}
			a.DepsChannel <- &dependency
		}
		return
	} else {
		if err == deptypes.ErrExtensionNotSupported {
			return
		}
		a.Log.Error().Str("error", err.Error()).Msg("could not parse deps")
	}
}
