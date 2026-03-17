package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/teecert/openclaw-configurator/internal/server"
)

var version = "dev"

func main() {
	bind := flag.String("bind", "127.0.0.1", "address to bind to")
	port := flag.Int("port", 19876, "port to listen on")
	noBrowser := flag.Bool("no-browser", false, "do not open browser automatically")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Println("openclaw-configurator", version)
		os.Exit(0)
	}

	token := generateToken()

	hostPort := net.JoinHostPort(*bind, strconv.Itoa(*port))
	url := fmt.Sprintf("http://%s/?token=%s", hostPort, token)

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║      OpenClaw Configurator                  ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Printf("║  Listen: http://%s\n", hostPort)
	fmt.Println("║")
	fmt.Println("║  Open this URL in your browser:")
	fmt.Printf("║  %s\n", url)
	fmt.Println("║")
	fmt.Println("║  Press Ctrl+C to stop")
	fmt.Println("╚══════════════════════════════════════════════╝")

	if !*noBrowser {
		openBrowser(url)
	}

	if err := server.ListenAndServe(*bind, *port, token); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %s\n", err)
		os.Exit(1)
	}
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate token: %s\n", err)
		os.Exit(1)
	}
	return hex.EncodeToString(b)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Println("║  (auto-open not supported on this OS)")
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("║  Could not open browser automatically.")
		fmt.Println("║  Please open the URL above manually.")
	}
}
