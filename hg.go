package main

/**
 * Mercurial Repo Information
 *
 * Look at [this](https://github.com/robbyrussell/oh-my-zsh/blob/master/plugins/branch/branch.plugin.zsh) for
 * possibly a faster way to get some data.
 */

import (
	"github.com/fatih/color"
	"io/ioutil"
	"path"
	"strings"
)

func NewMercurialRepoInfo(workingDirectory *string) *RepoInfo {
	codes := RepoChangeStatusFieldDefinitions["hg"]

	// Is this a hg repo
	output, exitCode, err := execAndGetOutput("findup", workingDirectory, ".hg")

	if err != nil {
		// Error
		return nil
	} else if exitCode == 1 {
		// .hg folder not found
		return &RepoInfo{IsRepo: false, VCS: AnsiString{Plain: codes.VCS, Colored: codes.VCS}}
	} else if exitCode != 0 {
		// Some other kind of error
		return nil
	}

	vcscolor := color.New(color.FgHiCyan)
	root := strings.TrimSpace(output)

	info := &RepoInfo{
		IsRepo:   true,
		VCS:      AnsiString{Plain: codes.VCS, Colored: vcscolor.Sprint(codes.VCS)},
		VCSColor: vcscolor,
		RepoPath: root,
		RepoName: path.Base(root),
	}

	// Go do a hg summary in that folder (TODO: This was super slow, for not don't implement)
	//output, exitCode, err = execAndGetOutput("hg", workingDirectory, "summary", "--remote")

	// Figure out branch status
	branchColor := color.New(color.FgGreen)

	// Figure out branches (TODO: Not fully implemented)
	info.OtherBranches = []AnsiString{} // not implemented
	branchBytes, branchErr := ioutil.ReadFile(info.RepoPath + "/.hg/branch")
	var branch string
	if branchErr != nil {
		branch = "!branch!"
	} else {
		branch = strings.TrimSpace(string(branchBytes))
	}
	info.BranchName = AnsiString{Plain: branch, Colored: branchColor.Sprint(branch)}

	// Get per-file status, as well as tracking info

	status := make(map[rune]int, len(codes.StatusCodes))
	for field := range codes.StatusCodes {
		status[field] = 0
	}

	output, _, err = execAndGetOutput("hg", workingDirectory, "status")

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if len(line) < 2 {
			continue
		}

		// Grab first character
		statchars := stripANSI(line)[:1]

		// File status
		for key := range status {
			if strings.ContainsRune(statchars, key) {
				status[key]++
			}
		}
	}

	info.ChangeStatusCounts = status
	colorStatus := buildColoredStatusStringFromMap(status, &codes)

	info.Status = AnsiString{Plain: stripANSI(colorStatus), Colored: colorStatus}

	return info
}
