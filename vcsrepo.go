package main

/**
* Information stored about VCS repos.
 */

import (
	"github.com/fatih/color"
)

//
// Output formats for our change status (M/A/D/R/C/U/?/!)
//

type RepoChangeStatus struct {
	OutputCharacter rune
	OutputColor     *color.Color
	Meaning         string
}

// What order to print status out in
var RepoChangeStatusFieldDefinitionsOrderedKeys = []rune{'M', 'A', 'D', 'R', 'C', 'U', '?', '!'}

var RepoChangeStatusFieldDefinitions = map[rune]RepoChangeStatus{
	'M': {OutputCharacter: 'M', OutputColor: color.New(color.FgGreen), Meaning: "modified"},
	'A': {OutputCharacter: '+', OutputColor: color.New(color.FgHiGreen), Meaning: "added"},
	'D': {OutputCharacter: '-', OutputColor: color.New(color.FgHiRed), Meaning: "deleted"},
	'R': {OutputCharacter: 'R', OutputColor: color.New(color.FgHiYellow), Meaning: "renamed"},
	'C': {OutputCharacter: 'C', OutputColor: color.New(color.FgHiBlue), Meaning: "copied"},
	'U': {OutputCharacter: 'U', OutputColor: color.New(color.FgHiMagenta), Meaning: "updated"},
	'?': {OutputCharacter: '?', OutputColor: color.New(color.FgRed), Meaning: "untracked"},
	'!': {OutputCharacter: '!', OutputColor: color.New(color.FgCyan), Meaning: "ignored"},
}

type AnsiString struct {
	Plain   string `json:"plain"`
	Colored string `json:"colored"`
}

type RepoInfo struct {
	IsRepo             bool         `json:"is_repo"`
	VCS                AnsiString   `json:"vcs"`
	VCSColor           *color.Color `json:"vcs_color"`
	RepoName           string       `json:"repo_name"`
	BranchName         AnsiString   `json:"current_branch"`
	BranchTrackingInfo AnsiString   `json:"tracking"`
	OtherBranches      []AnsiString `json:"branches"`
	ChangeStatusCounts map[rune]int `json:"status_counts"`
	Status             AnsiString   `json:"status"`
}
