package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/pborman/getopt/v2"
	"os"
	"strings"
)

type OutputType int

const (
	Full       OutputType = 0
	Prompt     OutputType = 1
	StatusLine OutputType = 2
)

type RepoType int

const (
	Detect    RepoType = 0
	Git       RepoType = 1
	Mercurial RepoType = 2
)

// Vcs Status Request
type Request struct {
	ForceColor bool
	Directory string
	Output OutputType
	Vcs RepoType
}

// Daemon startup options
type DaemonOptions struct {
	Enabled bool
	SockPath string
}

func parseOptions() (Request, DaemonOptions, error) {
	// Set up options
	forcecolor := getopt.BoolLong("color", 'c', "Force colored output.")

	workingdir := getopt.StringLong("dir", 'd', "",
		"The working directory to pretend we're in.\nNOTE: Tilde (~) exp    ansion is best-effort and should not be relied on.")

	outputtype := getopt.StringLong("output", 'o', "full", "Output format, from [full, prompt, statusline]")

	vcstype := getopt.StringLong("vcs", 'r', "detect", "Version Control System, from [detect, git, hg]")

	daemon := getopt.BoolLong("daemon", 'D', "Run as a daemon, listening on a socket for requests (does not fork).")

	sockpath := getopt.StringLong("socketpath",  'S', "", "What path to create a listening socket on?  Defaults to $HOME/.vcsstatus-sock")

	// Parse

	getopt.Parse()

	if *forcecolor {
		color.NoColor = false
	}

	dir := *workingdir

	if dir == "" {
		fullPath, err := os.Getwd()
		if err != nil {
			return Request{}, DaemonOptions{Enabled: false}, err
		}
		dir = fullPath
	}

	var output OutputType
	if *outputtype != "" {
		switch *outputtype {
		case "full":
			output = Full
			break
		case "prompt":
			output = Prompt
			break
		case "statusline":
			output = StatusLine
			break
		default:
			return Request{}, DaemonOptions{Enabled: false}, fmt.Errorf("invalid format passed to --output: '%s'", *outputtype)
		}
	}

	var vcs RepoType
	if *vcstype != "" {
		switch *vcstype {
		case "detect":
			vcs = Detect
			break
		case "git":
			vcs = Git
			break
		case "hg":
		case "mercurial":
			vcs = Mercurial
			break
		default:
			return Request{}, DaemonOptions{Enabled: false}, fmt.Errorf("invalid vcs system passed to --vcs: '%s'", *vcstype)
		}
	}

	socket := *sockpath
	if socket == "" {
		socket = os.ExpandEnv("$HOME") + "/.vcsstatus-sock"
	}

	return Request{
		ForceColor: *forcecolor,
		Directory: dir,
		Output: output,
		Vcs: vcs,
	}, DaemonOptions{Enabled: *daemon, SockPath: socket}, nil
}

func loadRepo(req Request) *RepoInfo {

	switch req.Vcs {
	case Git:
		return NewGitRepoInfo(&req.Directory)
	case Mercurial:
		return NewMercurialRepoInfo(&req.Directory)
	}

	// cases Detect, default, and other invalid options
	var info *RepoInfo

	// Git first
	info = NewGitRepoInfo(&req.Directory)
	if info != nil && info.IsRepo {
		// It was a git repo
		return info
	}

	// Mercurial next
	info = NewMercurialRepoInfo(&req.Directory)
	if info != nil && info.IsRepo {
		// It was a hg repo
		return info
	}

	// All done
	return nil
}

func buildResponse(req Request, info *RepoInfo) string {
	switch req.Output {
	case Prompt:
		var response strings.Builder
		response.WriteString(fmt.Sprintf("%s%s%s%s", info.VCS.Colored, info.VCSColor.Sprint(":<"), info.BranchName.Colored, info.VCSColor.Sprint(">")))
		if len(info.OtherBranches) > 0 {
			// Get just the colored names
			branches := []string{}
			for _, b := range info.OtherBranches {
				branches = append(branches, b.Colored)
			}
			response.WriteString(fmt.Sprintf(" {%s}", strings.Join(branches, ", ")))
		}
		response.WriteString("\n")
		response.WriteString(info.Status.Colored + "\n")
		return response.String()
	case StatusLine:
		var response strings.Builder
		response.WriteString(info.VCS.Colored + "\n")
		response.WriteString(info.RepoName + "\n")
		response.WriteString(info.BranchTrackingInfo.Colored + "\n")
		response.WriteString(info.Status.Colored + "\n")
		response.WriteString(info.RepoPath + "\n")
		return response.String()
	}

	// Full and default output types
	output, _ := json.MarshalIndent(info, "", " ")
	return string(output) + "\n"
}

func singleMain(req Request) {
	if req.Directory == "" {
		os.Exit(2)
	}

	info := loadRepo(req)

	if info == nil {
		os.Exit(1)
	}

	fmt.Print(buildResponse(req, info))
}

func daemonMain(req Request) {
	panic("Not implemented.")
}

func main() {
	req, daemon, err := parseOptions()
	if err != nil {
		panic(err)
	}

	if daemon.Enabled {
		daemonMain(req)
	} else {
		singleMain(req)
	}
}
