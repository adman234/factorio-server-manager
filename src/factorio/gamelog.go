package factorio

import (
	"bufio"
	"log"
	"os"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

// TailLog returns every line of the Factorio log file.
func TailLog() ([]string, error) {
	result := []string{}

	config := bootstrap.GetConfig()

	f, err := os.Open(config.FactorioLog)
	if err != nil {
		log.Printf("Error reading log %s", err)
		return result, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading log %s", err)
		return result, err
	}

	return result, nil
}
