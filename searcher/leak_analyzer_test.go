package searcher

import (
	"regexp"
	"testing"

	"github.com/AlexAkulov/hungryfox"
	"github.com/rs/zerolog"

	. "github.com/smartystreets/goconvey/convey"
)

var testData = hungryfox.Diff{
	CommitHash: "hash123",
	RepoURL:    "http://github.com",
	RepoPath:   "my/repo",
	FilePath:   "no_secret_here.txt",
	Content: `
	line 1
	line 2
	secret1
	line 3
	secret2
	password="qwerty123"
	`,
	AuthorEmail: "alexakulov86@gmail.com",
	Author:      "AA",
}

var expectedData = []hungryfox.Leak{
	hungryfox.Leak{
		PatternName:  "pattern1",
		Regexp:       "secret",
		FilePath:     "no_secret_here.txt",
		RepoPath:     "my/repo",
		RepoURL:      "http://github.com",
		CommitHash:   "hash123",
		CommitAuthor: "AA",
		CommitEmail:  "alexakulov86@gmail.com",
		LeakString:   "\tsecret1",
	},
	hungryfox.Leak{
		PatternName:  "pattern1",
		Regexp:       "secret",
		FilePath:     "no_secret_here.txt",
		RepoPath:     "my/repo",
		RepoURL:      "http://github.com",
		CommitHash:   "hash123",
		CommitAuthor: "AA",
		CommitEmail:  "alexakulov86@gmail.com",
		LeakString:   "\tsecret2",
	},
}

func TestAnalyzeDiff(t *testing.T) {
	Convey("Test GetLeaks", t, func() {
		leakChannel := make(chan *hungryfox.Leak)
		patterns := []patternType{
			patternType{
				Name:      "pattern1",
				ContentRe: regexp.MustCompile("secret"),
				FileRe:    regexp.MustCompile("secret"),
			},
			patternType{
				Name:      "pattern2",
				ContentRe: regexp.MustCompile("Password="),
				FileRe:    matchAllRegex,
			},
		}
		leakAnalyzer := LeakAnalyzer{
			Log:         zerolog.Nop(),
			LeakChannel: leakChannel,
			Matchers: &Matchers{
				patterns: patterns,
			},
		}

		go leakAnalyzer.Analyze(&testData)
		var results []hungryfox.Leak
		for i := 0; i < 2; i++ {
			results = append(results, *<-leakChannel)
		}

		So(results, ShouldResemble, expectedData)
	})
}
