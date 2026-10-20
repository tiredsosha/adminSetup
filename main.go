package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
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
	MongoURL  = "http://tiredsosha.ru/updates/mongodb.msi"

	VSCodeName = "VS Code"
	VSCodeURL  = "https://code.visualstudio.com/sha/download?build=stable&os=win32-x64-user"

	GitName = "Git"
	GitURL  = "https://github.com/git-for-windows/git/releases/download/v2.56.0.windows.2/Git-2.56.0.2-64-bit.exe"

	REPO1_URL = "git@github.com:tiredsosha/adminFront.git"
	REPO1_DIR = `C:\APP\admin\adminFront`

	REPO2_URL = "git@github.com:tiredsosha/adminBack.git"
	REPO2_DIR = `C:\APP\admin\adminBack`

	// Порты
	ClientServerPort = "5005"
	ServerRemotePort = "8080"

	// ENV шаблоны
	ClientEnvTemplate = "REACT_APP_SERVER_PORT=http://%s:%s\nREACT_APP_ADMIN_LOGIN=%s\nREACT_APP_ADMIN_PASSWORD=%s"
	ServerEnvTemplate = "REMOTE_SERVER_URL=http://%s:%s"
)

var (
	REPO1_BRANCH string
	REPO2_BRANCH string
	LocalIP      string
)

func userInput() error {
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

	defaultIP, err := discoverLocalIP()
	if err != nil {
		fmt.Printf("Could not detect local IP automatically: %v\n", err)
		fmt.Print("Enter local IP: ")
	} else {
		fmt.Printf("Detected local IP: %s\n", defaultIP)
		fmt.Printf("Enter local IP (press Enter to use %s): ", defaultIP)
	}
	ip, _ := reader.ReadString('\n')
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = defaultIP
	}
	if ip == "" {
		return fmt.Errorf("local IP address is required")
	}

	REPO1_BRANCH = r1
	REPO2_BRANCH = r2
	LocalIP = ip
	return nil
}

func discoverLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("detect network interface: %w", err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected local address type %T", conn.LocalAddr())
	}
	if localAddr.IP == nil || localAddr.IP.IsUnspecified() {
		return "", fmt.Errorf("no usable local IP address found")
	}
	return localAddr.IP.String(), nil
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

func ensureSSHKey() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find user home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return fmt.Errorf("create SSH directory: %w", err)
	}

	privateKeyPath := filepath.Join(sshDir, "id_ed25519")
	publicKeyPath := privateKeyPath + ".pub"
	privateKeyExists, err := fileExists(privateKeyPath)
	if err != nil {
		return fmt.Errorf("check SSH private key: %w", err)
	}
	publicKeyExists, err := fileExists(publicKeyPath)
	if err != nil {
		return fmt.Errorf("check SSH public key: %w", err)
	}
	if privateKeyExists != publicKeyExists {
		return fmt.Errorf("SSH key pair is incomplete; check %s and %s", privateKeyPath, publicKeyPath)
	}

	if !privateKeyExists {
		sshKeygen, err := findSSHKeygen()
		if err != nil {
			return err
		}

		cmd := exec.Command(sshKeygen, "-t", "ed25519", "-C", "adminSetup", "-f", privateKeyPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("generate SSH key: %w", err)
		}
	}

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("read SSH public key: %w", err)
	}

	fmt.Println("\nAdd this public SSH key to your Git repository:")
	fmt.Println(strings.TrimSpace(string(publicKey)))
	fmt.Printf("\nPublic key file: %s\n", publicKeyPath)
	return nil
}

func findSSHKeygen() (string, error) {
	if path, err := exec.LookPath("ssh-keygen"); err == nil {
		return path, nil
	}

	var candidates []string
	for _, baseDir := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		if baseDir != "" {
			candidates = append(candidates, filepath.Join(baseDir, "Git", "usr", "bin", "ssh-keygen.exe"))
		}
	}
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		candidates = append(candidates, filepath.Join(systemRoot, "System32", "OpenSSH", "ssh-keygen.exe"))
	}
	for _, path := range candidates {
		if exists, err := fileExists(path); err != nil {
			return "", fmt.Errorf("check ssh-keygen at %s: %w", path, err)
		} else if exists {
			return path, nil
		}
	}

	return "", fmt.Errorf("ssh-keygen was not found; install Git for Windows or the Windows OpenSSH Client and try again")
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
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

	// 1) SSH key
	if askYesNo("Step 1: Generate SSH key?", true) {
		fmt.Println("\n[1/10] Generate SSH key")
		if err := ensureSSHKey(); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped SSH key generation.")
	}

	// User input
	if err := userInput(); err != nil {
		fatal(err)
	}

	// 2) Node
	if askYesNo("Step 2: Download & install Node.js?", true) {
		nodeMsi := filepath.Join(targetDir, filepath.Base(NodeURL))
		fmt.Println("\n[2/10] Download:", NodeName)
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

	// 3) Mongo
	if askYesNo("Step 3: Download & install MongoDB?", true) {
		mongoMsi := filepath.Join(targetDir, filepath.Base(MongoURL))
		fmt.Println("\n[3/10] Download:", MongoName)
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

	// 4) MongoDB -> PATH
	if askYesNo("Step 4: Add MongoDB to system PATH?", true) {
		fmt.Println("\n[4/10] Add MongoDB to system PATH")
		if err := envpath.AddMongoBinToSystemPath(); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped PATH update.")
	}

	// 5) VS Code
	if askYesNo("Step 5: Download & install VS Code?", true) {
		vscodeFile := "VSCodeUserSetup-x64.exe"
		vscodePath := filepath.Join(targetDir, vscodeFile)
		fmt.Println("\n[5/10] Download:", VSCodeName)
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

	// 6) Git
	if askYesNo("Step 6: Download & install Git?", true) {
		gitPath := filepath.Join(targetDir, filepath.Base(GitURL))
		fmt.Println("\n[6/10] Download:", GitName)
		if err := download.ToFile(GitURL, gitPath, DownloadTimeout); err != nil {
			fatal(err)
		}
		fmt.Println("\nInstall:", GitName)
		if err := exe.RunAndWait(gitPath); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped Git.")
	}

	// 7) Repos
	if askYesNo("Step 7: Clone/update repositories?", true) {
		fmt.Println("\n[7/10] Clone/update repos")
		if err := gitclone.Ensure(REPO1_URL, REPO1_BRANCH, REPO1_DIR); err != nil {
			fatal(err)
		}
		if err := gitclone.Ensure(REPO2_URL, REPO2_BRANCH, REPO2_DIR); err != nil {
			fatal(err)
		}
	} else {
		fmt.Println("Skipped repos.")
	}

	// 8) .ENV files
	if askYesNo("Step 8: Create .env files for adminFront?", true) {
		fmt.Println("\n[8/10] Create .ENV files")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter admin login (default: admin): ")
		adminLogin, _ := reader.ReadString('\n')
		adminLogin = strings.TrimSpace(adminLogin)
		if adminLogin == "" {
			adminLogin = "admin"
		}

		fmt.Print("Enter admin password: ")
		adminPassword, _ := reader.ReadString('\n')
		adminPassword = strings.TrimSpace(adminPassword)
		if adminPassword == "" {
			fatal(fmt.Errorf("admin password cannot be empty"))
		}

		clientEnvContent := fmt.Sprintf(ClientEnvTemplate, LocalIP, ClientServerPort, adminLogin, adminPassword)
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

	// 9) NPM INSTALL
	if askYesNo("Step 9: Run npm install in server and client?", true) {
		fmt.Println("\n[9/10] Installing NPM dependencies")

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

	fmt.Println("\n[10/10] Done.")
	fmt.Print("All steps are complete. Press Enter to close this window...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}
