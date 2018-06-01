package repo

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AlexAkulov/hungryfox"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/diff"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Repo struct {
	DiffChannel      chan<- *hungryfox.Diff
	HistoryPastLimit time.Time
	DataPath         string
	RepoPath         string
	URL              string

	repository *git.Repository
	repoState  hungryfox.RepoState
}

func (r *Repo) Close() {
	r.repository = nil // ???
	runtime.GC()       // ???
}

func (r *Repo) Status() hungryfox.RepoState {
	return r.repoState
}

func (r *Repo) SetOldRefs(refs map[string]string) {
	r.repoState.Refs = refs
}

func (r *Repo) GetNewRefs() (refsMap map[string]string) {
	refsMap = map[string]string{}
	refs, err := r.repository.References()
	if err != nil {
		return
	}
	lastCommit := r.getLastCommit()
	if lastCommit != "" {
		refsMap["last"] = lastCommit
	}

	refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Hash().IsZero() {
			return nil
		}
		if strings.HasPrefix(ref.Name().String(), "refs/keep-around/") {
			return nil
		}
		refsMap[ref.Name().String()] = ref.Hash().String()
		return nil
	})
	return
}

func (r *Repo) isChecked(commitHash string) bool {
	for _, checkedCommit := range r.repoState.Refs {
		if commitHash == checkedCommit {
			return true
		}
	}
	return false
}

func (r *Repo) getLastCommit() string {
	oldWD, err := os.Getwd()
	if err != nil {
		return ""
	}
	if err := os.Chdir(r.fullRepoPath()); err != nil {
		return ""
	}
	// --topo-order???
	out, err := exec.Command("git", "rev-list", "--all", "--remotes", "--date-order", "--max-count=1").Output()
	os.Chdir(oldWD)
	if err != nil {
		return ""
	}
	commits := strings.Split(string(out), "\n")
	if len(commits) > 0 {
		return commits[0]
	}
	return ""
}

func (r *Repo) getRevList() (result []*object.Commit, err error) {
	oldWD, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error on get working dir: %v", err)
	}
	if err := os.Chdir(r.fullRepoPath()); err != nil {
		return nil, fmt.Errorf("error on change dir to %s: %v", r.fullRepoPath(), err)
	}
	// --topo-order???
	out, err := exec.Command("git", "rev-list", "--all", "--remotes", "--date-order").Output()
	os.Chdir(oldWD)
	if err != nil {
		return nil, err
	}

	hashList := strings.Split(string(out), "\n")
	for _, commitHash := range hashList {
		commitHash = strings.TrimSpace(commitHash)
		if r.isChecked(commitHash) {
			break
		}
		commit, err := r.repository.CommitObject(plumbing.NewHash(commitHash))
		if err != nil {
			continue
		}
		if commit.NumParents() != 1 {
			continue
		}
		result = append(result, commit)
	}

	r.repoState.ScanStatus.CommitsTotal = len(result)
	if len(hashList) > 0 {
		r.repoState.Refs["last"] = hashList[0]
	}
	return result, nil
}

func (r *Repo) Open() error {
	var err error
	r.repository, err = git.PlainOpen(r.fullRepoPath())
	return err
}

func (r *Repo) OpenScanClose() error {
	if err := r.Open(); err != nil {
		return err
	}
	defer r.Close()
	return r.Scan()
}

// Scan - start
func (r *Repo) Scan() error {
	r.repoState.ScanStatus = hungryfox.ScanStatus{
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
		Success:   false,
	}
	commits, err := r.getRevList()
	if err != nil {
		return err
	}
	for i, commit := range commits {
		r.repoState.ScanStatus.CommitsScanned = i + 1
		if commit.Committer.When.Before(r.HistoryPastLimit) {
			r.getAllChanges(commit)
			break
		}
		r.getCommitChanges(commit)
	}
	r.repoState.ScanStatus.EndTime = time.Now().UTC()
	r.repoState.ScanStatus.Success = true
	r.repoState.Refs = r.GetNewRefs()
	return nil
}

func (r *Repo) getAllChanges(commit *object.Commit) error {
	tree, err := commit.Tree()
	if err != nil {
		return err
	}
	changes, err := object.DiffTree(nil, tree)
	if err != nil {
		return err
	}
	patch, err := changes.Patch()
	if err != nil {
		return err
	}
	for _, p := range patch.FilePatches() {
		_, f := p.Files()
		if f == nil || p.IsBinary() {
			continue
		}
		for _, chunk := range p.Chunks() {
			if chunk.Type() != diff.Add {
				continue
			}
			r.DiffChannel <- &hungryfox.Diff{
				CommitHash:  commit.Hash.String(),
				RepoURL:     r.URL,
				RepoPath:    r.RepoPath,
				FilePath:    f.Path(),
				LineBegin:   0, // TODO: await https://github.com/src-d/go-git/issues/806
				Content:     chunk.Content(),
				Author:      "unknown", // TODO: Use blame for this
				AuthorEmail: "unknown",
				TimeStamp:   commit.Author.When,
			}
		}
	}
	return nil
}

func (r *Repo) getCommitChanges(commit *object.Commit) error {
	if commit == nil {
		return nil
	}
	parrentCommit, err := commit.Parent(0)
	if err != nil {
		return err
	}
	patch, err := parrentCommit.Patch(commit)
	if err != nil {
		return err
	}
	for _, p := range patch.FilePatches() {
		_, f := p.Files()
		if f == nil || p.IsBinary() {
			continue
		}
		for _, chunk := range p.Chunks() {
			if chunk.Type() != diff.Add {
				continue
			}
			r.DiffChannel <- &hungryfox.Diff{
				CommitHash:  commit.Hash.String(),
				RepoURL:     r.URL,
				RepoPath:    r.RepoPath,
				FilePath:    f.Path(),
				LineBegin:   0, // TODO: await https://github.com/src-d/go-git/issues/806
				Content:     chunk.Content(),
				Author:      commit.Author.Name,
				AuthorEmail: commit.Author.Email,
				TimeStamp:   commit.Author.When,
			}
		}
	}
	return nil
}
