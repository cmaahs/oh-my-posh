package segments

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/integralist/go-findroot/find"
	"github.com/jandedobbeleer/oh-my-posh/src/platform"
	"github.com/jandedobbeleer/oh-my-posh/src/properties"
	gl "github.com/maahsome/gitlab-go"
	giturls "github.com/whilp/git-urls"
)

type GitlabMR struct {
	props properties.Properties
	env   platform.Environment

	Count       string
	AuthorCount string

	ProjectID string
}

type MergeRequestList []MergeRequest

type MergeRequest struct {
	ID        int `json:"id"`
	Iid       int `json:"iid"`
	ProjectID int `json:"project_id"`
	Author    struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"author"`
}

const (
	//NewProp enables something
	Token            properties.Property = "token_variable"
	GitlabHost       properties.Property = "gitlab_hostname"
	GitlabAPIVersion properties.Property = "gitlab_api_version"
	AuthorUsername   properties.Property = "author_username"
	AuthorOnly       properties.Property = "author_only"
)

func (mr *GitlabMR) Enabled() bool {
	inProject := false
	gitRoot, gerr := find.Repo()
	if gerr != nil {
		mr.Count = "no root"
		return true
		// return false
	}
	gitDir := fmt.Sprintf("%s/.git", gitRoot.Path)
	if stat, err := os.Stat(gitDir); err == nil {
		inProject = true
		if !stat.IsDir() {
			realDir, rerr := os.ReadFile(gitDir)
			if rerr != nil {
				mr.Count = "cannot read file"
				return true
				// return false
			}
			workDir := strings.TrimSuffix(strings.Split(strings.TrimSpace(strings.TrimPrefix(string(realDir), "gitdir: ")), ".git")[0], "/")
			gitRoot.Path = workDir
		}
	} else {
		mr.Count = "just false"
		inProject = true
		// inProject = false
	}
	if inProject {
		// get project id
		projectID := mr.doFetchGitlabProjectID(gitRoot)
		if projectID == 0 {
			mr.Count = "project 0"
			return true
			// return false
		}
		mr.ProjectID = fmt.Sprintf("%d", projectID)
		authorOnly := mr.props.GetBool(AuthorOnly, false)
		mr.doFetchGitlabMR(authorOnly, projectID)
	}
	return inProject
}

func (mr *GitlabMR) Template() string {
	return " {{.Count}} "
}

func (mr *GitlabMR) Init(props properties.Properties, env platform.Environment) {
	mr.props = props
	mr.env = env

	// mr.Count = "10"
	// mr.Text = props.GetString(NewProp, "Hello")
}

func (mr *GitlabMR) doFetchGitlabProjectID(gitRoot find.Stat) int {
	var gitClient gl.GitlabClient

	glHost := mr.props.GetString(GitlabHost, "gitlab.com")
	tokenEnv := mr.props.GetString(Token, "")
	glToken := os.Getenv(tokenEnv)
	gitClient = gl.New(glHost, "", glToken)

	repo, rerr := git.PlainOpen(gitRoot.Path)
	if rerr != nil {
		return 0
	}
	repoConfig, rcerr := repo.Config()
	if rcerr != nil {
		return 0
	}
	// fmt.Printf("%#v\n", repoConfig.Remotes)
	pURLs, _ := giturls.Parse(repoConfig.Remotes["origin"].URLs[0])
	glSlug := strings.TrimPrefix(strings.TrimSuffix(pURLs.EscapedPath(), ".git"), "/")
	glSlug = url.PathEscape(glSlug)

	projectID, pierr := gitClient.GetProjectID(glSlug)
	if pierr != nil {
		return 0
	}

	return projectID

}

func (mr *GitlabMR) doFetchGitlabMR(authorOnly bool, id int) {
	var gitClient gl.GitlabClient

	glHost := mr.props.GetString(GitlabHost, "gitlab.com")
	tokenEnv := mr.props.GetString(Token, "")
	glToken := os.Getenv(tokenEnv)
	gitClient = gl.New(glHost, "", glToken)
	authorUsername := mr.props.GetString(AuthorUsername, "")

	var uri string
	if authorOnly {
		uri = fmt.Sprintf("/projects/%d/merge_requests?state=opened&per_page=50", id)
		if authorUsername != "" {
			uri = fmt.Sprintf("/projects/%d/merge_requests?state=opened&author_username=%s&per_page=50", id, authorUsername)
		}
	} else {
		uri = fmt.Sprintf("/projects/%d/merge_requests?state=opened&per_page=50", id)
	}

	gitdata, err := gitClient.Get(uri)
	if err != nil {
		mr.Count = "ERR"
		mr.AuthorCount = "ERR"
		return
	}

	var mrList MergeRequestList
	marshErr := json.Unmarshal([]byte(gitdata), &mrList)
	if marshErr != nil {
		mr.Count = "ERR"
		mr.AuthorCount = "ERR"
		return
	}
	if authorOnly {
		mr.Count = fmt.Sprintf("%d", len(mrList))
		mr.AuthorCount = fmt.Sprintf("%d", len(mrList))
	} else {
		mr.Count = fmt.Sprintf("%d", len(mrList))
		count := 0
		for _, m := range mrList {
			if m.Author.Username == authorUsername {
				count++
			}
		}
		mr.AuthorCount = fmt.Sprintf("%d", count)
	}

}
