package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"adminSetup/download"
	"adminSetup/elevate"
	"adminSetup/envpath"
	"adminSetup/exe"
	"adminSetup/gitclone"
	"adminSetup/msi"
	"adminSetup/singleinst"
)

const (
	targetDir = `C:\APP`
	reposRoot = `C:\APP\admin`

	DownloadTimeout = 20 * time.Minute

	mutexName = `Global\Bootstrap_Node_Mongo_DownloadAndInstall`

	NodeName = "Node.js"
	NodeURL  = "https://nodejs.org/dist/v20.20.0/node-v20.20.0-x64.msi"

	MongoName = "MongoDB"
	MongoURL  = "https://fastdl.mongodb.org/windows/mongodb-windows-x86_64-8.2.5-signed.msi"

	VSCodeName = "VS Code"
	VSCodeURL  = "https://code.visualstudio.com/sha/download?build=stable&os=win32-x64-user"

	REPO1_URL = "https://github.com/tiredsosha/adminFront.git"
	REPO1_DIR = `C:\APP\admin\adminFront`

	REPO2_URL = "https://github.com/tiredsosha/adminBack.git"
	REPO2_DIR = `C:\APP\admin\adminBack`

	// Порты
	ClientServerPort = "5005"
	ServerRemotePort = "8080"

	// ENV шаблоны
	ClientEnvTemplate = "REACT_APP_SERVER_PORT=http://%s:%s"
	ServerEnvTemplate = "REMOTE_SERVER_URL=http://%s:%s"
)

var (
	REPO1_BRANCH string
	REPO2_BRANCH string
	LocalIP      string
)

func userInput() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter adminFront branch (default: main): ")
	r1, _ := reader.ReadString('\n')
	r1 = strings.TrimSpace(r1)
	if r1 == "" {
		r1 = "main"
	}

	fmt.Print("Enter adminBack branch (default: main): ")
	r2, _ := reader.ReadString('\n')
	r2 = strings.TrimSpace(r2)
	if r2 == "" {
		r2 = "main"
	}

	fmt.Print("Enter local IP (default: 127.0.0.1): ")
	ip, _ := reader.ReadString('\n')
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "127.0.0.1"
	}

	REPO1_BRANCH = r1
	REPO2_BRANCH = r2
	LocalIP = ip
}

func askYesNo(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	def := "y/N"
	if defaultYes {
		def = "Y/n"
	}

	for {
		fmt.Printf("%s [%s]: ", prompt, def)
		s, _ := reader.ReadString('\n')
		s = strings.TrimSpace(strings.ToLower(s))

		if s == "" {
			return defaultYes
		}
		if s == "y" || s == "yes" {
			return true
		}
		if s == "n" || s == "no" {
			return false
		}
		fmt.Println("Please type y or n.")
	}
}

func main() {
	release, already, err := singleinst.Acquire(mutexName)
	if err != nil {
		fatal(err)
	}
	if already {
		return
	}
	defer release()

	if !elevate.IsElevated() {
		if err := elevate.RelaunchAsAdmin(); err != nil {
			fatal(err)
		}
		return
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(reposRoot, 0o755); err != nil {
		fatal(err)
	}

	// 0) User input
	userInput()

	// 1) Node
	if askYesNo("Step 1: Download & install Node.js?", true) {
		nodeMsi := filepath.Join(targetDir, filepath.Base(NodeURL))
		fmt.Println("\n[1/8] Download:", NodeName)
		if err := download.ToFile(NodeURL, nodeMsi, DownloadTimeout); err != nil {
			fatal(err)
		}
		fmt.Println("\nInstall:", NodeName)
		if err := msi.RunAndWait(nodeMsi); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped Node.js.")
	}

	// 2) Mongo
	if askYesNo("Step 2: Download & install MongoDB?", true) {
		mongoMsi := filepath.Join(targetDir, filepath.Base(MongoURL))
		fmt.Println("\n[2/8] Download:", MongoName)
		if err := download.ToFile(MongoURL, mongoMsi, DownloadTimeout); err != nil {
			fatal(err)
		}
		fmt.Println("\nInstall:", MongoName)
		if err := msi.RunAndWait(mongoMsi); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped MongoDB install.")
	}

	// 3) MongoDB -> PATH
	if askYesNo("Step 3: Add MongoDB to system PATH?", true) {
		fmt.Println("\n[3/8] Add MongoDB to system PATH")
		if err := envpath.AddMongoBinToSystemPath(); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped PATH update.")
	}

	// 4) VS Code
	if askYesNo("Step 4: Download & install VS Code?", true) {
		vscodeFile := "VSCodeUserSetup-x64.exe"
		vscodePath := filepath.Join(targetDir, vscodeFile)
		fmt.Println("\n[4/8] Download:", VSCodeName)
		if err := download.ToFile(VSCodeURL, vscodePath, DownloadTimeout); err != nil {
			fatal(err)
		}
		fmt.Println("\nInstall:", VSCodeName)
		if err := exe.RunAndWait(vscodePath); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped VS Code.")
	}

	// 5) Repos
	if askYesNo("Step 5: Clone/update repositories?", true) {
		fmt.Println("\n[5/8] Clone/update repos")
		if err := gitclone.Ensure(REPO1_URL, REPO1_BRANCH, REPO1_DIR); err != nil {
			fatal(err)
		}
		if err := gitclone.Ensure(REPO2_URL, REPO2_BRANCH, REPO2_DIR); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped repos.")
	}

	// 6) .ENV files
	if askYesNo("Step 6: Create .env files for adminFront?", true) {
		fmt.Println("\n[6/8] Create .ENV files")

		clientEnvContent := fmt.Sprintf(ClientEnvTemplate, LocalIP, ClientServerPort)
		serverEnvContent := fmt.Sprintf(ServerEnvTemplate, LocalIP, ServerRemotePort)

		clientEnvPath := filepath.Join(REPO1_DIR, "client", ".env")
		serverEnvPath := filepath.Join(REPO1_DIR, "server", ".env")

		if err := envpath.EnsureFile(clientEnvPath, clientEnvContent); err != nil {
			fatal(err)
		}
		if err := envpath.EnsureFile(serverEnvPath, serverEnvContent); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped .env creation.")
	}

	// 7) NPM INSTALL
	if askYesNo("Step 7: Run npm install in server and client?", true) {
		fmt.Println("\n[7/8] Installing NPM dependencies")

		serverDir := filepath.Join(REPO1_DIR, "server")
		clientDir := filepath.Join(REPO1_DIR, "client")

		fmt.Println("\nInstalling server dependencies...")
		if err := exe.RunCommand(serverDir, "npm install"); err != nil {
			fatal(err)
		}

		fmt.Println("\nInstalling client dependencies...")
		if err := exe.RunCommand(clientDir, "npm install"); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped npm install.")
	}

	fmt.Println("\n[8/8] Done.")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}
