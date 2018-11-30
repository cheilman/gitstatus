package main

/**
 * Git Repo Information
 */

import (
	"github.com/fatih/color"
	"path"
	"strings"
)

func NewGitRepoInfo(workingDirectory *string) *RepoInfo {
	codes := RepoChangeStatusFieldDefinitions["git"]

	// TODO: Make this not run a command to get this data
	// Go do a git status in that folder
	output, exitCode, err := execAndGetOutput("git", workingDirectory,
		"-c", "color.status=never", "-c", "color.ui=never", "status")

	if err != nil {
		// Some kind of command execution error!
		return nil
	} else if exitCode == 128 {
		// Not a git repo
		return &RepoInfo{IsRepo: false, VCS: AnsiString{Plain: codes.VCS, Colored: codes.VCS}}
	} else if exitCode != 0 {
		// Some kind of git error!
		return nil
	}

	vcscolor := color.New(color.FgHiCyan)

	info := &RepoInfo{IsRepo: true, VCS: AnsiString{Plain: codes.VCS, Colored: vcscolor.Sprint(codes.VCS)}, VCSColor: vcscolor}

	// Figure out branch status TODO: This could be optimized I bet
	branchColor := color.New(color.FgGreen)

	if strings.Contains(output, "still merging") || strings.Contains(output, "Unmerged paths") {
		branchColor = color.New(color.FgHiMagenta)
	} else if strings.Contains(output, "Untracked files") {
		branchColor = color.New(color.FgHiRed)
	} else if strings.Contains(output, "Changes not staged for commit") {
		branchColor = color.New(color.FgHiYellow)
	} else if strings.Contains(output, "Changes to be committed") {
		branchColor = color.New(color.FgYellow)
	} else if strings.Contains(output, "Your branch is ahead of") {
		branchColor = color.New(color.FgMagenta)
	}

	// Get repo name
	output, exitCode, err = execAndGetOutput("git", workingDirectory,
		"rev-parse", "--show-toplevel")
	if err == nil {
		info.RepoName = path.Base(strings.TrimSpace(output))
	} else {
		info.RepoName = "unknown"
	}

	// Figure out branches
	output, _, err = execAndGetOutput("git", workingDirectory,
		"-c", "color.status=never", "-c", "color.ui=never", "branch")

	if err == nil {
		lines := strings.Split(output, "\n")

		info.OtherBranches = []AnsiString{}

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if len(line) > 0 {
				if strings.HasPrefix(line, "* ") {
					branch := strings.TrimPrefix(line, "* ")
					info.BranchName = AnsiString{Plain: branch, Colored: branchColor.Sprint(branch)}

					// Update the git color if this is the master branch
					if branch == "master" || branch == "mainline" {
						info.VCSColor = color.New(color.FgHiGreen)
						info.VCS.Colored = info.VCSColor.Sprint(info.VCS.Plain)
					}
				} else {
					branch := strings.TrimSpace(line)
					info.OtherBranches = append(info.OtherBranches, AnsiString{Plain: branch, Colored: color.WhiteString(branch)})
				}
			}
		}
	} else {
		info.OtherBranches = []AnsiString{}
		branch := "!branch!"
		info.BranchName = AnsiString{Plain: branch, Colored: branchColor.Sprint(branch)}
	}

	if err == nil {
		// Get per-file status, as well as tracking info

		status := make(map[rune]int, len(codes.StatusCodes))
		for field := range codes.StatusCodes {
			status[field] = 0
		}

		output, _, err = execAndGetOutput("git", workingDirectory,
			"-c", "color.status=always", "-c", "color.ui=always", "status", "-s", "-b")

		lines := strings.Split(output, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if len(line) < 2 {
				continue
			}

			// Grab first two characters
			statchars := stripANSI(line)[:2]

			if statchars == "##" {
				// Branch status
				tracking := line[3:]
				info.BranchTrackingInfo = AnsiString{Plain: stripANSI(tracking), Colored: tracking}
			} else {
				// File status
				for key := range status {
					if strings.ContainsRune(statchars, key) {
						status[key]++
					}
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
