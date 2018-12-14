package main

/**
 * Mercurial Repo Information
 *
 * Look at [this](https://github.com/robbyrussell/oh-my-zsh/blob/master/plugins/branch/branch.plugin.zsh) for
 * possibly a faster way to get some data.
 */

import (
	"github.com/fatih/color"
	"path"
	"strings"
)

func NewMercurialRepoInfo(workingDirectory *string) *RepoInfo {
	codes := RepoChangeStatusFieldDefinitions["hg"]

	// TODO: Make this not run a command to get this data
	// Go do a hg summary in that folder
	output, exitCode, err := execAndGetOutput("hg", workingDirectory, "summary", "--remote")

	if err != nil {
		// Some kind of command execution error!
		return nil
	} else if exitCode == 255 {
		// Not a hg repo
		return &RepoInfo{IsRepo: false, VCS: AnsiString{Plain: codes.VCS, Colored: codes.VCS}}
	} else if exitCode != 0 {
		// Some kind of hg error!
		return nil
	}

	vcscolor := color.New(color.FgHiCyan)

	info := &RepoInfo{IsRepo: true, VCS: AnsiString{Plain: codes.VCS, Colored: vcscolor.Sprint(codes.VCS)}, VCSColor: vcscolor}

	// Figure out branch status TODO: This could be optimized I bet
	branchColor := color.New(color.FgGreen)

	if strings.Contains(output, "merge") {
		branchColor = color.New(color.FgHiMagenta)
	} else if strings.Contains(output, "unknown") {
		branchColor = color.New(color.FgHiRed)
	} else if strings.Contains(output, "(clean)") {
		branchColor = color.New(color.FgGreen)
	} else {
		branchColor = color.New(color.FgHiYellow)
	}

	// Grab tracking information
	for _, line := range strings.Split(output, "\n") {
		line := strings.TrimSpace(line)
		if strings.HasPrefix(line, "remote") {
			track := strings.TrimPrefix(line, "remote: ")
			info.BranchTrackingInfo = AnsiString{Plain: track, Colored: track}
		}
	}

	// Get repo name
	output, exitCode, err = execAndGetOutput("hg", workingDirectory, "root")
	if err == nil {
		info.RepoPath = strings.TrimSpace(output)
		info.RepoName = path.Base(info.RepoPath)
	} else {
		info.RepoName = "unknown"
		info.RepoPath = *workingDirectory
	}

	// Figure out branches
	output, _, err = execAndGetOutput("hg", workingDirectory, "bookmark")

	if err == nil && !strings.Contains(output, "no bookmarks set") {
		// Return bookmarks instead of branches
		lines := strings.Split(output, "\n")

		info.OtherBranches = []AnsiString{}

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if len(line) > 0 {
				if strings.HasPrefix(line, "* ") {
					branch := strings.TrimPrefix(line, "* ")
					if strings.Contains(branch, ":") {
						branch = branch[:len(branch)-15]
					}
					branch = strings.TrimSpace(branch)

					info.BranchName = AnsiString{Plain: branch, Colored: branchColor.Sprint(branch)}

					// Update the hg color if this is the master branch
					if branch == "master" || branch == "mainline" || branch == "default" {
						info.VCSColor = color.New(color.FgHiGreen)
						info.VCS.Colored = info.VCSColor.Sprint(info.VCS.Plain)
					}
				} else {
					branch := strings.TrimSpace(line)
					if strings.Contains(branch, ":") {
						branch = branch[:len(branch)-15]
					}
					branch = strings.TrimSpace(branch)
					info.OtherBranches = append(info.OtherBranches, AnsiString{Plain: branch, Colored: branch})
				}
			}
		}
	} else {
		output, _, err = execAndGetOutput("hg", workingDirectory, "branch")

		if err == nil {
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)

				branch := strings.TrimSpace(line)
				if len(branch) > 0 {
					info.BranchName = AnsiString{Plain: branch, Colored: branchColor.Sprint(branch)}

					// Update the hg color if this is the master branch
					if branch == "master" || branch == "mainline" || branch == "default" {
						info.VCSColor = color.New(color.FgHiGreen)
						info.VCS.Colored = info.VCSColor.Sprint(info.VCS.Plain)
					}
				}
			}
		} else {
			info.OtherBranches = []AnsiString{}
			branch := "!branch!"
			info.BranchName = AnsiString{Plain: branch, Colored: branchColor.Sprint(branch)}
		}
	}

	if err == nil {
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

	} else {
		status := "!status!"
		info.Status = AnsiString{Plain: status, Colored: color.HiRedString(status)}
	}

	return info
}
