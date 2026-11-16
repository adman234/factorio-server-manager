package factorio

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/rcon"
)

var (
	// ErrAlreadyRunning is returned by Start when a Factorio process is running or starting.
	ErrAlreadyRunning = errors.New("factorio server is already running")
	// ErrNotRunning is returned by Stop and Kill when no Factorio process is running.
	ErrNotRunning = errors.New("factorio server is not running")
)

// stateMu guards the process state of the server (Running, Savefile, BindIP, Port,
// Cmd, starting, exited). It is package level so that Server stays copyable.
var stateMu sync.Mutex

// rconMu guards Server.Rcon. The rcon client is not safe for concurrent use.
var rconMu sync.Mutex

// logMu serializes console output handling (log file and websocket), which is
// fed by the stdout and stderr readers.
var logMu sync.Mutex
var consoleLog *os.File

type Server struct {
	Cmd            *exec.Cmd              `json:"-"`
	Savefile       string                 `json:"savefile"`
	Latency        int                    `json:"latency"`
	BindIP         string                 `json:"bindip"`
	Port           int                    `json:"port"`
	Running        bool                   `json:"running"`
	Version        Version                `json:"fac_version"`
	BaseModVersion string                 `json:"base_mod_version"`
	Settings       map[string]interface{} `json:"-"`
	Rcon           *rcon.RemoteConsole    `json:"-"`

	starting bool
	exited   chan struct{}
}

// StartOptions are the per-start settings sent by the web UI.
type StartOptions struct {
	Savefile string `json:"savefile"`
	BindIP   string `json:"bindip"`
	Port     int    `json:"port"`
}

var instantiated Server

func broadcastStatus(status []byte) {
	websocket.WebsocketHub.GetRoom("server_status").Send(string(status))
}

// SetRunning updates the running state and notifies websocket clients when it changed.
func (server *Server) SetRunning(newState bool) {
	stateMu.Lock()
	changed := server.Running != newState
	server.Running = newState
	var status []byte
	if changed {
		status, _ = json.Marshal(server)
	}
	stateMu.Unlock()

	if changed {
		broadcastStatus(status)
	}
}

func (server *Server) GetRunning() bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return server.Running
}

// Status returns a JSON-safe snapshot of the server state.
func (server *Server) Status() Server {
	stateMu.Lock()
	defer stateMu.Unlock()
	return Server{
		Savefile:       server.Savefile,
		Latency:        server.Latency,
		BindIP:         server.BindIP,
		Port:           server.Port,
		Running:        server.Running,
		Version:        server.Version,
		BaseModVersion: server.BaseModVersion,
	}
}

// Exited returns a channel that is closed when the current Factorio process exits.
// It returns nil when no process is running.
func (server *Server) Exited() <-chan struct{} {
	stateMu.Lock()
	defer stateMu.Unlock()
	if !server.Running {
		return nil
	}
	return server.exited
}

func (server *Server) autostart() {
	stateMu.Lock()
	opts := StartOptions{Savefile: "Load Latest", BindIP: server.BindIP, Port: server.Port}
	stateMu.Unlock()

	if opts.BindIP == "" {
		opts.BindIP = "0.0.0.0"
	}
	if opts.Port == 0 {
		opts.Port = 34197
	}

	if err := server.Start(opts); err != nil {
		log.Printf("Error starting Factorio server: %+v", err)
	}
}

func SetFactorioServer(server Server) {
	stateMu.Lock()
	defer stateMu.Unlock()
	instantiated = server
}

func NewFactorioServer() (err error) {
	server := Server{}
	server.Settings = make(map[string]interface{})
	config := bootstrap.GetConfig()
	if err = os.MkdirAll(config.FactorioConfigDir, 0755); err != nil {
		log.Printf("failed to create config directory: %v", err)
		return
	}

	settingsPath := config.SettingsFile
	var settings *os.File

	if _, err = os.Stat(settingsPath); os.IsNotExist(err) {
		// copy example settings to supplied settings file, if not exists
		log.Printf("Server settings at %s not found, copying example server settings.\n", settingsPath)

		examplePath := filepath.Join(config.FactorioDir, "data", "server-settings.example.json")

		var example *os.File
		example, err = os.Open(examplePath)
		if err != nil {
			log.Printf("failed to open example server settings: %v", err)
			return
		}
		defer example.Close()

		settings, err = os.Create(settingsPath)
		if err != nil {
			log.Printf("failed to create server settings file: %v", err)
			return
		}
		defer settings.Close()

		_, err = io.Copy(settings, example)
		if err != nil {
			log.Printf("failed to copy example server settings: %v", err)
			return
		}

		err = example.Close()
		if err != nil {
			log.Printf("failed to close example server settings: %s", err)
			return
		}
	} else {
		// otherwise, open file normally
		settings, err = os.Open(settingsPath)
		if err != nil {
			log.Printf("failed to open server settings file: %v", err)
			return
		}
		defer settings.Close()
	}

	// before reading reset offset
	if _, err = settings.Seek(0, 0); err != nil {
		log.Printf("error while seeking in settings file: %v", err)
		return
	}

	if err = json.NewDecoder(settings).Decode(&server.Settings); err != nil {
		log.Printf("error reading %s: %v", settingsPath, err)
		return
	}

	log.Printf("Loaded Factorio settings from %s\n", settingsPath)

	var out []byte
	//Load factorio version
	if config.GlibcCustom == "true" {
		out, err = exec.Command(config.GlibcLocation, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--version").Output()
	} else {
		out, err = exec.Command(config.FactorioBinary, "--version").Output()
	}

	if err != nil {
		log.Printf("error on loading factorio version: %s", err)
		return
	}

	reg := regexp.MustCompile(`Version.*?((\d+\.)?(\d+\.)?(\*|\d+)+)`)
	found := reg.FindStringSubmatch(string(out))
	if len(found) < 2 {
		err = fmt.Errorf("could not find a version in the output of %s --version: %q", config.FactorioBinary, out)
		log.Println(err)
		return
	}
	err = server.Version.UnmarshalText([]byte(found[1]))
	if err != nil {
		log.Printf("could not parse version: %v", err)
		return
	}

	//Load baseMod version
	baseModInfoFile := filepath.Join(config.FactorioBaseModDir, "info.json")
	bmifBa, err := os.ReadFile(baseModInfoFile)
	if err != nil {
		log.Printf("couldn't open baseMods info.json: %s", err)
		return
	}
	var modInfo ModInfo
	err = json.Unmarshal(bmifBa, &modInfo)
	if err != nil {
		log.Printf("error unmarshalling baseMods info.json to a modInfo: %s", err)
		return
	}

	server.BaseModVersion = modInfo.Version

	// load admins from additional file
	if (server.Version.Greater(Version{0, 17, 0})) {
		if _, err = os.Stat(config.FactorioAdminFile); os.IsNotExist(err) {
			//save empty admins-file
			err = os.WriteFile(config.FactorioAdminFile, []byte("[]"), 0664)
			server.Settings["admins"] = make([]string, 0)
		} else {
			var data []byte
			data, err = os.ReadFile(config.FactorioAdminFile)
			if err != nil {
				log.Printf("Error loading FactorioAdminFile: %s", err)
				return
			}

			var jsonData interface{}
			err = json.Unmarshal(data, &jsonData)
			if err != nil {
				log.Printf("Error unmarshalling FactorioAdminFile: %s", err)
				return
			}

			server.Settings["admins"] = jsonData
		}
	}

	SetFactorioServer(server)

	// autostart factorio is configured to do so
	if config.Autostart == "true" {
		go GetFactorioServer().autostart()
	}

	return
}

func GetFactorioServer() (f *Server) {
	return &instantiated
}

// Start launches the Factorio process and returns once it has been started.
// It returns ErrAlreadyRunning if a process is already running or starting.
func (server *Server) Start(opts StartOptions) (err error) {
	if opts.Savefile == "" {
		return errors.New("no save file provided")
	}

	stateMu.Lock()
	if server.Running || server.starting {
		stateMu.Unlock()
		return ErrAlreadyRunning
	}
	server.starting = true
	server.Savefile = opts.Savefile
	server.BindIP = opts.BindIP
	server.Port = opts.Port
	version := server.Version
	stateMu.Unlock()

	defer func() {
		if err != nil {
			stateMu.Lock()
			server.starting = false
			stateMu.Unlock()
		}
	}()

	config := bootstrap.GetConfig()
	data, err := json.MarshalIndent(server.Settings, "", "  ")
	if err != nil {
		log.Println("Failed to marshal FactorioServerSettings: ", err)
	} else if err = os.WriteFile(config.SettingsFile, data, 0644); err != nil {
		log.Println("Failed to write FactorioServerSettings: ", err)
	}

	saves, err := ListSaves()
	if err != nil {
		log.Println("Failed to get saves list: ", err)
	}

	if len(saves) == 0 {
		return errors.New("no savefile exists on the server")
	}

	args := []string{}

	//The factorio server refenences its executable-path, since we execute the ld.so file and pass the factorio binary as a parameter
	//the game would use the path to the ld.so file as it's executable path and crash, to prevent this the parameter "--executable-path" is added
	if config.GlibcCustom == "true" {
		log.Println("Custom glibc selected, glibc.so location:", config.GlibcLocation, " lib location:", config.GlibcLibLoc)
		args = append(args, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--executable-path", config.FactorioBinary)
	}

	args = append(args,
		"--bind", opts.BindIP,
		"--port", strconv.Itoa(opts.Port),
		"--server-settings", config.SettingsFile,
		"--rcon-port", strconv.Itoa(config.FactorioRconPort))

	if (version.Greater(Version{0, 17, 0})) {
		args = append(args, "--server-adminlist", config.FactorioAdminFile)
	}

	if strings.HasPrefix(opts.Savefile, "Load Latest") {
		args = append(args, "--start-server-load-latest")
	} else {
		args = append(args, "--start-server", filepath.Join(config.FactorioSavesDir, opts.Savefile))
	}

	// Write chat log to a different file if requested (if not it will be mixed-in with the default logfile)
	if config.ChatLogFile != "" {
		args = append(args, "--console-log", config.ChatLogFile)
	}

	// The rcon password is appended after logging, so it never ends up in the log.
	executable := config.FactorioBinary
	if config.GlibcCustom == "true" {
		executable = config.GlibcLocation
	}
	log.Println("Starting server with command: ", executable, args)
	cmd := exec.Command(executable, append(args, "--rcon-password", config.FactorioRconPass)...)

	// Keep stdin open (but unused) like before, so Factorio doesn't see EOF on its console.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Error opening stdin pipe: %s", err)
		return err
	}

	// Use io.Pipe instead of StdoutPipe: Wait() then waits until all output has
	// been consumed, so the last lines (e.g. the final save) are never lost.
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	cmd.Stdout = outW
	cmd.Stderr = errW

	openConsoleLog(config.ConsoleLogFile)

	err = cmd.Start()
	if err != nil {
		log.Printf("Factorio process failed to start: %s", err)
		outW.Close()
		errW.Close()
		closeConsoleLog()
		return err
	}

	var readers sync.WaitGroup
	readers.Add(2)
	go func() { defer readers.Done(); server.parseRunningCommand(outR) }()
	go func() { defer readers.Done(); server.parseRunningCommand(errR) }()

	exited := make(chan struct{})
	stateMu.Lock()
	server.Cmd = cmd
	server.exited = exited
	server.starting = false
	server.Running = true
	status, _ := json.Marshal(server)
	stateMu.Unlock()
	broadcastStatus(status)

	go server.wait(cmd, stdin, outW, errW, &readers, exited)

	return nil
}

// wait reaps the Factorio process and resets the server state once it has exited.
func (server *Server) wait(cmd *exec.Cmd, stdin io.Closer, outW, errW *io.PipeWriter, readers *sync.WaitGroup, exited chan struct{}) {
	err := cmd.Wait()
	stdin.Close()
	outW.Close()
	errW.Close()
	readers.Wait()
	closeConsoleLog()

	if err != nil {
		log.Printf("Factorio process exited with error: %s", err)
	} else {
		log.Printf("Factorio process is closed")
	}

	rconMu.Lock()
	rc := server.Rcon
	server.Rcon = nil
	rconMu.Unlock()
	if rc != nil {
		if err := rc.Close(); err != nil {
			log.Printf("Error closing rcon connection: %s", err)
		}
	}

	stateMu.Lock()
	server.Running = false
	server.Cmd = nil
	close(exited)
	status, _ := json.Marshal(server)
	stateMu.Unlock()
	broadcastStatus(status)
}

func (server *Server) runningCmd() (*exec.Cmd, <-chan struct{}) {
	stateMu.Lock()
	defer stateMu.Unlock()
	if !server.Running || server.Cmd == nil || server.Cmd.Process == nil {
		return nil, nil
	}
	return server.Cmd, server.exited
}

// Stop asks Factorio to save and shut down. It does not wait for the process to exit.
func (server *Server) Stop() error {
	cmd, _ := server.runningCmd()
	if cmd == nil {
		return ErrNotRunning
	}
	if err := interruptProcess(cmd); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		log.Printf("Error sending interrupt to Factorio process: %s", err)
		return err
	}
	log.Printf("Sent interrupt to Factorio process. Factorio shutting down...")
	return nil
}

// Kill forces Factorio to exit without saving and waits briefly for it to go away.
func (server *Server) Kill() error {
	cmd, exited := server.runningCmd()
	if cmd == nil {
		return ErrNotRunning
	}
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		log.Printf("Error sending SIGKILL to Factorio process: %s", err)
		return err
	}
	log.Printf("Sent SIGKILL to Factorio process. Factorio forced to exit.")

	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		return errors.New("factorio process did not exit after SIGKILL")
	}
	return nil
}

// StopAndWait stops Factorio (letting it save) and waits up to timeout for it to
// exit, killing it if it doesn't. It is used when FSM itself shuts down.
func (server *Server) StopAndWait(timeout time.Duration) error {
	cmd, exited := server.runningCmd()
	if cmd == nil {
		return nil
	}
	log.Printf("Stopping Factorio server (waiting up to %s for it to save)", timeout)
	if err := server.Stop(); err != nil && !errors.Is(err, ErrNotRunning) {
		log.Printf("Could not stop Factorio gracefully: %s", err)
	}
	select {
	case <-exited:
		return nil
	case <-time.After(timeout):
		log.Printf("Factorio did not exit within %s, killing it", timeout)
		return server.Kill()
	}
}

func (server *Server) parseRunningCommand(std io.Reader) {
	stdScanner := bufio.NewScanner(std)
	stdScanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for stdScanner.Scan() {
		text := stdScanner.Text()

		log.Printf("Factorio Server: %s", text)

		logMu.Lock()
		if err := writeLog(text); err != nil {
			log.Printf("Error: %s", err)
		}
		// send the reported line per websocket, in order
		websocket.WebsocketHub.GetRoom("gamelog").Send(text)
		logMu.Unlock()

		line := strings.Fields(text)
		// Ensure logline slice is in bounds
		if len(line) > 1 {
			// Check if Factorio Server reports any errors if so handle it
			if line[1] == "Error" {
				err := server.checkLogError(line)
				if err != nil {
					log.Printf("Error checking Factorio Server Error: %s", err)
				}
			}
			// check if slice index is greater than 2 to prevent panic
			if len(line) > 2 {
				// If rcon port opens indicated in log connect to rcon
				if strings.Contains(text, "Starting RCON interface at IP") {
					log.Printf("Rcon running on Factorio Server")
					if err := connectRC(text); err != nil {
						log.Printf("Error: %s", err)
					}
				}

				server.checkProcessHealth(text)
			}
		}
	}
	if err := stdScanner.Err(); err != nil {
		log.Printf("Error reading std buffer: %s", err)
		// keep draining so the process never blocks on a full pipe
		_, _ = io.Copy(io.Discard, std)
	}
}

func openConsoleLog(name string) {
	logMu.Lock()
	defer logMu.Unlock()
	if consoleLog != nil {
		consoleLog.Close()
	}
	file, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Cannot open logfile %s for appending Factorio Server output: %s", name, err)
		consoleLog = nil
		return
	}
	consoleLog = file
}

func closeConsoleLog() {
	logMu.Lock()
	defer logMu.Unlock()
	if consoleLog != nil {
		consoleLog.Close()
		consoleLog = nil
	}
}

// writeLog appends a line to the console log file. Callers must hold logMu.
func writeLog(logline string) error {
	if consoleLog == nil {
		return nil
	}
	if _, err := consoleLog.WriteString(logline + "\n"); err != nil {
		return fmt.Errorf("appending to %s: %w", consoleLog.Name(), err)
	}
	return nil
}

func (server *Server) checkLogError(logline []string) error {
	// TODO Handle errors generated by running Factorio Server
	log.Println(logline)

	return nil
}

func init() {
	websocket.WebsocketHub.RegisterControlHandler <- serverWebsocketControl
}

// react to websocket control messages and run the command if it is requested
func serverWebsocketControl(controls websocket.WsControls) {
	if controls.Type != "command" {
		return
	}
	command := controls.Value
	server := GetFactorioServer()
	if !server.GetRunning() {
		return
	}
	log.Printf("Received command: %v", command)

	rconMu.Lock()
	defer rconMu.Unlock()
	if server.Rcon == nil {
		log.Printf("Cannot send command, RCON is not connected yet")
		return
	}
	reqId, err := server.Rcon.Write(command)
	if err != nil {
		log.Printf("Error sending rcon command: %s", err)
		return
	}

	log.Printf("Command send to Factorio: %s, with rcon request id: %v", command, reqId)
}
