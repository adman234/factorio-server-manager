package factorio

import (
	"errors"
	"log"
	"net"
	"regexp"
	"strconv"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"

	"github.com/OpenFactorioServerManager/rcon"
)

// matches the address in "Starting RCON interface at IP ADDR:({0.0.0.0:27015})"
var rconAddrRegexp = regexp.MustCompile(`\{([^}]+)\}`)

// rconAddress returns the address FSM should dial to reach the RCON interface
// announced in logLine. FSM always runs on the same host as Factorio, so a
// wildcard bind address is dialed via loopback.
func rconAddress(logLine string, port int) string {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	m := rconAddrRegexp.FindStringSubmatch(logLine)
	if len(m) < 2 {
		return addr
	}
	host, p, err := net.SplitHostPort(m[1])
	if err != nil {
		return addr
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsUnspecified() {
		return net.JoinHostPort("127.0.0.1", p)
	}
	return net.JoinHostPort(host, p)
}

func connectRC(logLine string) error {
	config := bootstrap.GetConfig()
	rconAddr := rconAddress(logLine, config.FactorioRconPort)
	rc, err := rcon.Dial(rconAddr, config.FactorioRconPass)
	if err != nil {
		log.Printf("Cannot create rcon session: %s", err)
		return err
	}

	server := GetFactorioServer()
	if !server.GetRunning() {
		rc.Close()
		return errors.New("factorio exited before the rcon session was established")
	}

	rconMu.Lock()
	old := server.Rcon
	server.Rcon = rc
	rconMu.Unlock()
	if old != nil {
		old.Close()
	}

	log.Printf("rcon session established on %s", rconAddr)
	return nil
}
