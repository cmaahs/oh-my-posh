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
	FromCache   string

	ProjectID string
}

type mergeRequestList []mergeRequest

type mergeRequest struct {
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
	// EnvToken environment variable name that holds the gitlab personal access token
	EnvToken properties.Property = "token_variable"
	// AccessToken gitlab personal access token
	AccessToken properties.Property = "access_token"
	// GitlabHost hostname of the gitlab instance
	GitlabHost properties.Property = "gitlab_hostname"
	// GitlabAPIVersion the api version of the gitlab instance
	GitlabAPIVersion properties.Property = "gitlab_api_version"
	// AuthorUsername gitlab username
	AuthorUsername properties.Property = "author_username"
	// AuthorOnly displays MRs for the AuthorUsername
	AuthorOnly properties.Property = "author_only"
	// RootOnly displays the segment only in the root of the git worktree
	RootOnly properties.Property = "root_only"

	// GitlabMRCacheKeyResponse key used when caching the response
	GitlabMRCacheKeyResponse string = "gitlabmr_response"
	// GitlabMRCacheKeyURL key used when caching the url responsible for the response
	GitlabMRCacheKeyProjectID string = "gitlabmr_project_id"
	// ErrorMsg display when an MR count cannot be obtained
	ErrorMsg string = "ERR"
)

func (mr *GitlabMR) Enabled() bool {
	// TODO: create an inProject bool function
	inProject := false
	gitRoot, gerr := find.Repo()
	if gerr != nil {
		return false
	}
	rootOnly := mr.props.GetBool(RootOnly, true)
	if rootOnly {
		cwd, _ := os.Getwd()
		if cwd != gitRoot.Path {
			return false
		}
	}
	gitDir := fmt.Sprintf("%s/.git", gitRoot.Path)
	if stat, err := os.Stat(gitDir); err == nil {
		inProject = true
		if !stat.IsDir() {
			realDir, rerr := os.ReadFile(gitDir)
			if rerr != nil {
				mr.Count = "cannot read .git file"
				return true
				// return false
			}
			workDir := strings.TrimSuffix(strings.Split(strings.TrimSpace(strings.TrimPrefix(string(realDir), "gitdir: ")), ".git")[0], "/")
			gitRoot.Path = workDir
		}
		inProject = mr.OriginMatch(gitRoot)
	} else {
		mr.Count = "no .git dir"
		inProject = true
		// inProject = false
	}

	if inProject {

		cacheTimeout := mr.props.GetInt(properties.CacheTimeout, properties.DefaultCacheTimeout)

		response := new(mergeRequestList)
		if cacheTimeout > 0 {
			path, err := os.Getwd()
			if err != nil {
				mr.Count = ErrorMsg
				mr.AuthorCount = ErrorMsg
				return true
			}
			id, projectFound := mr.env.Cache().Get(fmt.Sprintf("%s_%s", GitlabMRCacheKeyProjectID, path))
			if projectFound {
				// check if data stored in cache
				val, found := mr.env.Cache().Get(fmt.Sprintf("%s_%s", GitlabMRCacheKeyResponse, id))
				// we got something from te cache
				if found {
					err := json.Unmarshal([]byte(val), response)
					if err != nil {
						mr.Count = ErrorMsg
						mr.AuthorCount = ErrorMsg
						return true
					}
					// mr.URL, _ = mr.env.Cache().Get(GitlabMRCacheKeyURL)
					authorUsername := mr.props.GetString(AuthorUsername, "")
					authorOnly := mr.props.GetBool(AuthorOnly, false)
					mr.buildCount(response, authorOnly, authorUsername)
					mr.FromCache = "*"
					return true
				}
			}
		}
		// get project id
		projectID := mr.doFetchGitlabProjectID(gitRoot)
		if projectID == 0 {
			mr.Count = "project 0"
			return true
			// return false
		}
		mr.ProjectID = fmt.Sprintf("%d", projectID)
		authorOnly := mr.props.GetBool(AuthorOnly, false)
		mr.doFetchGitlabMR(authorOnly, projectID, cacheTimeout)
		mr.FromCache = ""
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

// TODO: Fix all the duplicate opening of gitlab
func (mr *GitlabMR) OriginMatch(gitRoot find.Stat) bool {
	glHost := mr.props.GetString(GitlabHost, "gitlab.com")

	repo, rerr := git.PlainOpen(gitRoot.Path)
	if rerr != nil {
		return false
	}
	repoConfig, rcerr := repo.Config()
	if rcerr != nil {
		return false
	}
	// fmt.Printf("%#v\n", repoConfig.Remotes)
	pURLs, _ := giturls.Parse(repoConfig.Remotes["origin"].URLs[0])

	return strings.Contains(pURLs.String(), glHost)
}

func (mr *GitlabMR) doFetchGitlabProjectID(gitRoot find.Stat) int {
	var gitClient gl.GitlabClient

	glHost := mr.props.GetString(GitlabHost, "gitlab.com")
	tokenEnv := mr.props.GetString(EnvToken, "")
	glToken := mr.props.GetString(AccessToken, "")
	if tokenEnv != "" {
		glToken = os.Getenv(tokenEnv)
	}

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

func (mr *GitlabMR) doFetchGitlabMR(authorOnly bool, id int, cacheTimeout int) {
	var gitClient gl.GitlabClient

	glHost := mr.props.GetString(GitlabHost, "gitlab.com")
	tokenEnv := mr.props.GetString(EnvToken, "")
	glToken := mr.props.GetString(AccessToken, "")
	authorUsername := mr.props.GetString(AuthorUsername, "")
	if tokenEnv != "" {
		glToken = os.Getenv(tokenEnv)
	}

	gitClient = gl.New(glHost, "", glToken)

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
		mr.Count = ErrorMsg
		mr.AuthorCount = ErrorMsg
		return
	}

	var mrList mergeRequestList
	marshErr := json.Unmarshal([]byte(gitdata), &mrList)
	if marshErr != nil {
		mr.Count = ErrorMsg
		mr.AuthorCount = ErrorMsg
		return
	}
	if cacheTimeout > 0 {
		path, err := os.Getwd()
		if err != nil {
			mr.Count = ErrorMsg
			mr.AuthorCount = ErrorMsg
			return
		}
		// persist new forecasts in cache
		mr.env.Cache().Set(fmt.Sprintf("%s_%d", GitlabMRCacheKeyResponse, id), gitdata, cacheTimeout)
		mr.env.Cache().Set(fmt.Sprintf("%s_%s", GitlabMRCacheKeyProjectID, path), fmt.Sprintf("%d", id), cacheTimeout)
	}

	mr.buildCount(&mrList, authorOnly, authorUsername)
}

func (mr *GitlabMR) buildCount(mrList *mergeRequestList, authorOnly bool, authorUsername string) {
	if authorOnly {
		mr.Count = fmt.Sprintf("%d", len(*mrList))
		mr.AuthorCount = fmt.Sprintf("%d", len(*mrList))
	} else {
		mr.Count = fmt.Sprintf("%d", len(*mrList))
		count := 0
		for _, m := range *mrList {
			if m.Author.Username == authorUsername {
				count++
			}
		}
		mr.AuthorCount = fmt.Sprintf("%d", count)
	}
}
