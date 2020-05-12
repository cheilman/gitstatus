package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/pborman/getopt/v2"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

type ExecutionType int

const (
	SingleUse   ExecutionType = 0
	Daemon      ExecutionType = 1
	Client      ExecutionType = 2
	DaemonCheck ExecutionType = 3
)

// Vcs Status Request
type Request struct {
	ForceColor  bool
	Directory   string
	Output      OutputType
	Vcs         RepoType
	StatusCheck bool
}

// Vcs Status Response
type Response struct {
	ExitCode int
	Content  string
}

// Execution options
type ExecutionOptions struct {
	Execution            ExecutionType
	SocketPath           string
	ForceSocketOverwrite bool
}

func parseOptions() (Request, ExecutionOptions, error) {
	// Set up options
	forcecolor := getopt.BoolLong("color", 'c', "Force colored output.")

	workingdir := getopt.StringLong("dir", 'd', "",
		"The working directory to pretend we're in.\nNOTE: Tilde (~) exp    ansion is best-effort and should not be relied on.")

	outputtype := getopt.EnumLong("output", 'o', []string{"full", "prompt", "statusline"}, "full", "Output format")

	vcstype := getopt.EnumLong("vcs", 'r', []string{"detect", "git", "hg"}, "detect", "Version Control System")

	exectype := getopt.EnumLong("exec", 'X', []string{"singleuse", "daemon", "client", "daemoncheck"}, "singleuse", "How to invoke vcsstatus.  Listen for requests as a daemon, connect to a daemon as a client, or run single-use.")

	socketpath := getopt.StringLong("socketpath", 'S', "", "What path to listen/connect on (for daemon/client) Defaults to $HOME/.vcsstatus-sock")

	overwritesocket := getopt.BoolLong("overwritesocket", 'O', "If the socketpath exists, overwrite it.")

	// Parse

	getopt.Parse()

	if *forcecolor {
		color.NoColor = false
	}

	dir := *workingdir

	if dir == "" {
		fullPath, err := os.Getwd()
		if err != nil {
			return Request{}, ExecutionOptions{}, err
		}
		dir = fullPath
	}

	var output OutputType
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
		return Request{}, ExecutionOptions{}, fmt.Errorf("invalid format passed to --output: '%s'", *outputtype)
	}

	var vcs RepoType
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
		return Request{}, ExecutionOptions{}, fmt.Errorf("invalid vcs system passed to --vcs: '%s'", *vcstype)
	}

	var exec ExecutionType
	switch *exectype {
	case "singleuse":
		exec = SingleUse
		break
	case "daemon":
		exec = Daemon
		break
	case "client":
		exec = Client
		break
	case "daemoncheck":
		exec = DaemonCheck
		break
	default:
		return Request{}, ExecutionOptions{}, fmt.Errorf("invalid execution type passed to --exec: '%s'", *exectype)
	}

	socket := *socketpath
	if socket == "" {
		socket = os.ExpandEnv("$HOME") + "/.vcsstatus-sock"
	}

	return Request{
			ForceColor: *forcecolor,
			Directory:  dir,
			Output:     output,
			Vcs:        vcs,
		}, ExecutionOptions{
			Execution:            exec,
			SocketPath:           socket,
			ForceSocketOverwrite: *overwritesocket},
		nil
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

func buildResponse(req Request, info *RepoInfo) Response {
	// TODO: Color handling isn't thread-safe / handles multiple requests well
	// TODO: Color support doesn't work the first time w/ daemons
	color.NoColor = !req.ForceColor

	if req.Directory == "" {
		return Response{ExitCode: 2, Content: "Directory must be non-empty."}
	}

	if info == nil {
		return Response{ExitCode: 1, Content: "Error loading repository information."}
	}

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
		return Response{ExitCode: 0, Content: response.String()}
	case StatusLine:
		var response strings.Builder
		response.WriteString(info.VCS.Colored + "\n")
		response.WriteString(info.RepoName + "\n")
		response.WriteString(info.BranchTrackingInfo.Colored + "\n")
		response.WriteString(info.Status.Colored + "\n")
		response.WriteString(info.RepoPath + "\n")
		return Response{ExitCode: 0, Content: response.String()}
	}

	// Full and default output types
	output, _ := json.MarshalIndent(info, "", " ")
	return Response{ExitCode: 0, Content: string(output) + "\n"}
}

func singleMain(req Request) {
	info := loadRepo(req)

	response := buildResponse(req, info)

	fmt.Print(response.Content)
	os.Exit(response.ExitCode)
}

func cleanUpExistingSocket(options ExecutionOptions) {
	_, err := os.Stat(options.SocketPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found, good!
			return
		}

		// Any error other than file not found
		log.Fatalf("Error reading socket path '%s': %s", options.SocketPath, err)
	}

	if options.ForceSocketOverwrite {
		// Allow us to overwrite existing files
		if err := os.RemoveAll(options.SocketPath); err == nil {
			// Successfully deleted
			return
		}
		log.Fatalf("Could not remove existing file at '%s': %s", options.SocketPath, err)
	}
}

func writeResponse(connection net.Conn, response Response) {

	output, _ := json.MarshalIndent(response, "", " ")

	writer := bufio.NewWriter(connection)
	_, err := writer.WriteString(string(output) + "\n")
	if err == nil {
		_ = writer.Flush()
	} else {
		log.Printf("Error writing response: %s", err)
	}
}

func handleConnection(connection net.Conn) {
	//noinspection GoUnhandledErrorResult
	defer connection.Close()

	decoder := json.NewDecoder(connection)

	var req Request
	err := decoder.Decode(&req)
	if err != nil {
		writeResponse(connection, Response{ExitCode: 100, Content: fmt.Sprintf("Error decoding request: %s\n", err)})
		return
	}

	if req.StatusCheck {
		// All we need to do is say we're up
		writeResponse(connection, Response{ExitCode: 0, Content: "OK\n"})
		return
	}

	// Load repo
	repo := loadRepo(req)

	// Build response
	response := buildResponse(req, repo)
	writeResponse(connection, response)
}

func daemonMain(options ExecutionOptions) {
	cleanUpExistingSocket(options)

	// Handle shutdown better
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// At this point we should be clear to create a socket
	listener, err := net.Listen("unix", options.SocketPath)
	if err != nil {
		log.Fatal(err)
	}

	// Try to have clean shutdowns
	done := false
	shutdown := func() {
		log.Printf("Shutting down...")
		done = true

		// Cleanup
		log.Printf("Closing listener")
		err := listener.Close()
		if err != nil {
			log.Printf("Failed to close listener: %s", err)
		}
	}

	// Cleanup on signal
	go func() {
		sig := <-sigs
		log.Printf("Received signal: %s", sig)
		shutdown()
	}()

	log.Printf("Listening on: %s", options.SocketPath)

	for !done {
		connection, err := listener.Accept()
		if err != nil {
			if done {
				break
			}

			log.Printf("Error accepting: %s", err)
			continue
		}

		go handleConnection(connection)
	}
}

func clientMain(req Request, options ExecutionOptions) {
	connection, err := net.Dial("unix", options.SocketPath)
	if err != nil {
		log.Fatalf("Error connecting to '%s': %s", options.SocketPath, err)
	}
	defer connection.Close()

	encoder := json.NewEncoder(connection)
	err = encoder.Encode(req)
	if err != nil {
		log.Fatalf("Error encoding request: %s", err)
	}

	decoder := json.NewDecoder(connection)
	var response Response
	err = decoder.Decode(&response)
	if err != nil {
		log.Fatalf("Error decoding response: %s\n", err)
	}

	_, err = os.Stdout.WriteString(response.Content)
	if err != nil {
		log.Fatalf("Error outputting response: %s", err)
	}

	os.Exit(response.ExitCode)
}

func daemonCheckMain(options ExecutionOptions) {
	connection, err := net.Dial("unix", options.SocketPath)
	if err != nil {
		log.Printf("Failed to connect to socket: '%s': %s", options.SocketPath, err)
		os.Exit(1)
	}
	defer connection.Close()

	req := Request{
		StatusCheck: true,
	}
	encoder := json.NewEncoder(connection)
	err = encoder.Encode(req)
	if err != nil {
		log.Fatalf("Error encoding request: %s", err)
		os.Exit(127)
	}

	decoder := json.NewDecoder(connection)
	var response Response
	err = decoder.Decode(&response)
	if err != nil {
		log.Fatalf("Error decoding response: %s\n", err)
	}

	os.Exit(response.ExitCode)
}

func main() {
	req, options, err := parseOptions()
	if err != nil {
		panic(err)
	}

	switch options.Execution {
	case Daemon:
		daemonMain(options)
		break
	case Client:
		clientMain(req, options)
		break
	case DaemonCheck:
		daemonCheckMain(options)
		break
	case SingleUse:
		fallthrough
	default:
		singleMain(req)
		break
	}
}
