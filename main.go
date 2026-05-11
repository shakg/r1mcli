package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	appName = "r1mcli"

	defaultHost    = "192.168.168.48"
	defaultPort    = 37256
	defaultTimeout = 5 * time.Second

	// Known packet:
	// HD -> FHD, assumed 720 -> 1080.
	packetSet1080Hex = "5566aabb0109000000d70084a31fec99010280073804001e1ec6195e42" 
	
	// FHD -> HD, 1080p -> 720p.
	packetSet720Hex = "5566aabb0109000000d900848990fa9301020005d002001e1e9df45edd"
)

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func main() {
	var (
		setRes     int
		setHost    string
		host       string
		port       int
		timeoutSec int
		verbose    bool
		dryRun     bool
	)

	flag.IntVar(&setRes, "set-res", 0, "set resolution: 720 or 1080")
	flag.StringVar(&setHost, "set-host", "", "save default camera host")
	flag.StringVar(&host, "host", "", "camera host override")
	flag.IntVar(&port, "port", 0, "camera TCP port")
	flag.IntVar(&timeoutSec, "timeout", int(defaultTimeout.Seconds()), "timeout in seconds")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&dryRun, "dry-run", false, "print packet without sending")
	flag.Usage = usage

	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		exitf("failed to load config: %v", err)
	}

	if setHost != "" {
		cfg.Host = setHost

		if cfg.Port == 0 {
			cfg.Port = defaultPort
		}

		if err := saveConfig(cfg); err != nil {
			exitf("failed to save config: %v", err)
		}

		fmt.Printf("Saved host: %s\n", cfg.Host)

		if setRes == 0 {
			return
		}
	}

	targetHost := resolveHost(host, cfg)
	targetPort := resolvePort(port, cfg)
	timeout := time.Duration(timeoutSec) * time.Second

	if setRes == 0 {
		flag.Usage()
		return
	}

	packetHex, err := packetForResolution(setRes)
	if err != nil {
		exitf("%v", err)
	}

	packet, err := hex.DecodeString(packetHex)
	if err != nil {
		exitf("invalid hardcoded packet hex: %v", err)
	}

	if verbose || dryRun {
		fmt.Printf("Host: %s\n", targetHost)
		fmt.Printf("Port: %d\n", targetPort)
		fmt.Printf("Timeout: %s\n", timeout)
		fmt.Printf("Resolution: %dp\n", setRes)
		fmt.Printf("Packet (%d bytes): %s\n", len(packet), packetHex)
	}

	if dryRun {
		return
	}

	response, err := sendPacket(targetHost, targetPort, timeout, packet)
	if err != nil {
		exitf("%v", err)
	}

	fmt.Printf("Resolution command sent: %dp\n", setRes)

	if len(response) > 0 {
		fmt.Printf("Response (%d bytes): %s\n", len(response), hex.EncodeToString(response))
	} else {
		fmt.Println("Response: empty")
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `%s

Usage:
  r1mcli --set-res 1080
  r1mcli --set-res 720
  r1mcli --set-host 192.168.168.48
  r1mcli --host 192.168.168.48 --port 37256 --set-res 1080

Options:
`, appName)

	flag.PrintDefaults()

	fmt.Fprintf(os.Stderr, `
Examples:
  Save camera host:
    r1mcli --set-host 192.168.168.48

  Set resolution to 1080p using saved host:
    r1mcli --set-res 1080

  Set resolution to 720p using explicit host:
    r1mcli --host 192.168.168.48 --set-res 720

  Dry run:
    r1mcli --set-res 1080 --dry-run -v

`)
}

func packetForResolution(res int) (string, error) {
	switch res {
	case 1080:
		return packetSet1080Hex, nil
	case 720:
		if packetSet720Hex == "" {
			return "", errors.New(
				"720p packet is not configured yet; add packetSet720Hex",
			)
		}

		return packetSet720Hex, nil
	default:
		return "", fmt.Errorf("unsupported resolution: %d; use 720 or 1080", res)
	}
}

func sendPacket(host string, port int, timeout time.Duration, packet []byte) ([]byte, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := net.Dialer{
		Timeout: timeout,
	}

	fmt.Println("Connecting to camera...")

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	fmt.Println("Connected.")
	fmt.Printf("Sending (%d bytes): %s\n", len(packet), hex.EncodeToString(packet))

	n, err := conn.Write(packet)
	if err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}

	if n != len(packet) {
		return nil, fmt.Errorf("partial write: sent %d of %d bytes", n, len(packet))
	}

	buf := make([]byte, 1024)

	n, err = conn.Read(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, errors.New("timeout: camera did not respond")
		}

		return nil, fmt.Errorf("read failed: %w", err)
	}

	return buf[:n], nil
}

func resolveHost(flagHost string, cfg Config) string {
	if flagHost != "" {
		return flagHost
	}

	if cfg.Host != "" {
		return cfg.Host
	}

	return defaultHost
}

func resolvePort(flagPort int, cfg Config) int {
	if flagPort > 0 {
		return flagPort
	}

	if cfg.Port > 0 {
		return cfg.Port
	}

	return defaultPort
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, appName, "config.json"), nil
}

func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{
				Host: defaultHost,
				Port: defaultPort,
			}, nil
		}

		return Config{}, err
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func saveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	return os.WriteFile(path, data, 0o644)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}