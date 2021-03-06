package resource

import (
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/gobwas/glob"

	"concourse-git-resource/common"
	"concourse-git-resource/git"
)

const DirectoryName = "git-repository-cache"

type CheckPayload struct {
	common.Payload
}

func NewCheckPayload(stdin []byte) *CheckPayload {
	var p CheckPayload
	common.Parse(&p, stdin)

	return &p
}

func Check(payload *CheckPayload, printer *common.Printer) {
	path := os.TempDir() + DirectoryName
	params := git.RepositoryParams{
		RemoteUrl:     payload.Source.Url,
		HttpLogin:     payload.Source.Login,
		HttpPassword:  payload.Source.Password,
		SshPrivateKey: payload.Source.PrivateKey,
	}

	fmt.Fprintln(os.Stderr, fmt.Sprintf("check: workdir=%q, remote=%q", path, params.RemoteUrl))

	var repo *git.Repository
	repo, err := git.Open(path, payload.Source.Branch, params)
	if err == nil {
		repo.Update()
	} else {
		repo = git.Clone(path, payload.Source.Branch, params)
	}
	defer repo.Close()

	var refs []common.Version
	if payload.Source.TagRegex != "" {
		re := regexp.MustCompile(payload.Source.TagRegex)
		for _, t := range repo.ListTags() {
			if !re.Match([]byte(t)) {
				continue
			}

			refs = append(refs, common.Version{Reference: t})

			if t == payload.Version.Reference {
				break
			}
		}
	} else {
		var pgs []glob.Glob
		for _, p := range payload.Source.Paths {
			pgs = append(pgs, glob.MustCompile(p))
		}

		for _, c := range repo.ListCommits() {
			if len(pgs) == 0 {
				refs = append(refs, common.Version{Reference: c.Id})
			} else {
				match := false
				for _, f := range c.Files {
					for _, pg := range pgs {
						if pg.Match(f) {
							match = true
						}
					}
				}

				if match {
					refs = append(refs, common.Version{Reference: c.Id})
				}
			}

			if c.Id == payload.Version.Reference {
				break
			}
		}
	}

	if len(refs) == 0 {
		printer.PrintData([]string{})
		return
	}

	sort.SliceStable(refs, func(k, v int) bool {
		return true
	})
	printer.PrintData(refs)
}
