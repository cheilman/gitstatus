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

var WORKING_DIRECTORY = ""
var OUTPUT_TYPE OutputType
var REPO_TYPE RepoType

func parseOptions() {
	// Set up options
	forcecolor := getopt.BoolLong("color", 'c', "Force colored output.")

	workingdir := getopt.StringLong("dir", 'd', "",
		"The working directory to pretend we're in.\nNOTE: Tilde (~) exp    ansion is best-effort and should not be relied on.")

	outputtype := getopt.StringLong("output", 'o', "full", "Output format, from [full, prompt, statusline]")

	vcstype := getopt.StringLong("vcs", 'r', "detect", "Version Control System, from [detect, git, hg]")

	// Parse

	getopt.Parse()

	if *forcecolor {
		color.NoColor = false
	}

	WORKING_DIRECTORY = *workingdir

	if WORKING_DIRECTORY == "" {
		fullPath, err := os.Getwd()
		if err != nil {
			// Can't figure out working directory
			panic("Can not figure out working directory!")
		}
		WORKING_DIRECTORY = fullPath
	}

	if *outputtype != "" {
		switch *outputtype {
		case "full":
			OUTPUT_TYPE = Full
			break
		case "prompt":
			OUTPUT_TYPE = Prompt
			break
		case "statusline":
			OUTPUT_TYPE = StatusLine
			break
		default:
			panic("Invalid format passed to --output, " + *outputtype)
		}
	}

	if *vcstype != "" {
		switch *vcstype {
		case "detect":
			REPO_TYPE = Detect
			break
		case "git":
			REPO_TYPE = Git
			break
		case "hg":
		case "mercurial":
			REPO_TYPE = Mercurial
			break
		default:
			panic("Invalid vcs system passed to --vcs, " + *vcstype)
		}
	}

}

func loadRepo() *RepoInfo {

	switch REPO_TYPE {
	case Git:
		return NewGitRepoInfo(&WORKING_DIRECTORY)
	case Mercurial:
		return NewMercurialRepoInfo(&WORKING_DIRECTORY)
	}

	// cases Detect, default, and other invalid options
	var info *RepoInfo

	// Git first
	info = NewGitRepoInfo(&WORKING_DIRECTORY)
	if info != nil && info.IsRepo {
		// It was a git repo
		return info
	}

	// Mercurial next
	info = NewMercurialRepoInfo(&WORKING_DIRECTORY)
	if info != nil && info.IsRepo {
		// It was a hg repo
		return info
	}

	// All done
	return nil
}

func main() {
	parseOptions()

	if WORKING_DIRECTORY == "" {
		os.Exit(2)
	}

	info := loadRepo()

	if info == nil {
		os.Exit(1)
	}

	switch OUTPUT_TYPE {
	case Prompt:
		fmt.Printf("%s%s%s%s", info.VCS.Colored, info.VCSColor.Sprint(":<"), info.BranchName.Colored, info.VCSColor.Sprint(">"))
		if len(info.OtherBranches) > 0 {
			// Get just the colored names
			branches := []string{}
			for _, b := range info.OtherBranches {
				branches = append(branches, b.Colored)
			}
			fmt.Printf(" {%s}", strings.Join(branches, ", "))
		}
		fmt.Println()
		fmt.Println(info.Status.Colored)
		os.Exit(0)
	case StatusLine:
		fmt.Println(info.VCS.Colored)
		fmt.Println(info.RepoName)
		fmt.Println(info.BranchTrackingInfo.Colored)
		fmt.Println(info.Status.Colored)
		os.Exit(0)
	}

	// Full and default output types
	output, _ := json.MarshalIndent(info, "", " ")
	fmt.Println(string(output))
}
