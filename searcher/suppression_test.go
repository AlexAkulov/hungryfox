package searcher

import (
	"testing"

	. "github.com/AlexAkulov/hungryfox"
	"github.com/package-url/packageurl-go"
	. "github.com/smartystreets/goconvey/convey"
)

const suppsFile = ".\\test\\suppressions.yml"

func TestLoadSuppressions(t *testing.T) {
	Convey("Loads suppressions", t, func() {
		supps, err := loadSuppressionsFromPath(suppsFile)

		So(err, ShouldBeNil)
		So(supps, ShouldHaveLength, 2)
	})
	Convey("Compiles correctly", t, func() {
		supps, err := loadSuppressionsFromPath(suppsFile)
		So(err, ShouldBeNil)

		Convey("suppression with all fields", func() {
			supp := selectByRepo(supps, "r")

			So(supp.Repository.String(), ShouldEqual, "r")
			So(supp.DependencyName.String(), ShouldEqual, "d")
			So(supp.Version.String(), ShouldEqual, "1\\.0\\.0")
			So(supp.FilePath.String(), ShouldEqual, "f")
			So(supp.Source.String(), ShouldEqual, "s")
			So(supp.Id.String(), ShouldEqual, "i")
			So(supp.Title.String(), ShouldEqual, "t")
			So(supp.Cve.String(), ShouldEqual, "c")
		})

		Convey("suppression with a couple of fields", func() {
			supp := selectByRepo(supps, "foo")

			So(supp.Repository.String(), ShouldEqual, "foo")
			So(supp.Cve.String(), ShouldEqual, "CVE-2019-12345")
		})
	})
	Convey("Filteres vulnerabilities", t, func() {
		supps, err := loadSuppressionsFromPath(suppsFile)
		So(err, ShouldBeNil)

		noMatchDep := &Dependency{
			Purl: packageurl.PackageURL{
				Name:    "Foo",
				Version: "bar",
			},
		}
		partialMatchDep := &Dependency{
			Purl: packageurl.PackageURL{
				Name:    "d",
				Version: "1.0.0",
			},
		}
		fullMatchDep := &Dependency{
			Purl: packageurl.PackageURL{
				Name:    "d",
				Version: "1.0.0",
			},
			Diff: Diff{
				RepoURL:  "r",
				FilePath: "f",
			},
		}
		noMatchVulns := []Vulnerability{
			Vulnerability{
				Cve: "uniq",
			},
		}
		fullMatchVulns := []Vulnerability{
			Vulnerability{
				Source: "s",
				Id:     "i",
				Cve:    "c",
				Title:  "t",
			},
		}
		Convey("matches none - not filtered", func() {
			filtered := filterSuppressed(noMatchDep, noMatchVulns, supps)

			So(filtered, ShouldResemble, noMatchVulns)
		})
		Convey("matches dep, doesn't match vulnerability - not filtered", func() {
			filtered := filterSuppressed(fullMatchDep, noMatchVulns, supps)

			So(filtered, ShouldResemble, noMatchVulns)
		})
		Convey("matches vulnerability, doesn't match dep - not filtered", func() {
			filtered := filterSuppressed(noMatchDep, fullMatchVulns, supps)

			So(filtered, ShouldResemble, fullMatchVulns)
		})
		Convey("matches vulnerability, partially matches dep - not filtered", func() {
			filtered := filterSuppressed(partialMatchDep, fullMatchVulns, supps)

			So(filtered, ShouldResemble, fullMatchVulns)
		})
		Convey("matches all - filtered", func() {
			filtered := filterSuppressed(fullMatchDep, fullMatchVulns, supps)

			So(filtered, ShouldBeEmpty)
		})
		Convey("filteres matches, leaves others (match, nomatch, match)", func() {
			vulnerabilities := append(append(fullMatchVulns, noMatchVulns...), fullMatchVulns...)

			filtered := filterSuppressed(fullMatchDep, vulnerabilities, supps)

			So(filtered, ShouldHaveLength, 1)
			So(filtered, ShouldResemble, noMatchVulns)
		})
		Convey("filteres matches, leaves others (nomatch, match, match)", func() {
			vulnerabilities := append(append(noMatchVulns, fullMatchVulns...), fullMatchVulns...)

			filtered := filterSuppressed(fullMatchDep, vulnerabilities, supps)

			So(filtered, ShouldHaveLength, 1)
			So(filtered, ShouldResemble, noMatchVulns)
		})
		Convey("filteres matches, leaves others (match, match, nomatch)", func() {
			vulnerabilities := append(append(fullMatchVulns, fullMatchVulns...), noMatchVulns...)

			filtered := filterSuppressed(fullMatchDep, vulnerabilities, supps)

			So(filtered, ShouldHaveLength, 1)
			So(filtered, ShouldResemble, noMatchVulns)
		})
	})
}

func TestStuff(t *testing.T) {
	Convey("st", t, func() {
		slc := []int{0, 1, 2, 3}
		for i, item := range slc {
			So(i, ShouldEqual, item)
			slc = append(slc[:0], slc[0+1:]...)
		}
	})
}

func selectByRepo(supps []suppression, repo string) *suppression {
	for _, s := range supps {
		if s.Repository.String() == repo {
			return &s
		}
	}
	return nil
}
