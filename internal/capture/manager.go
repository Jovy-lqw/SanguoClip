package capture

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	adbPath    string
	serial     string
	remotePath string
	remotePID  string
	localPath  string
	done       chan error
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*session)}
}

func ListDevices() ([]domain.Device, error) {
	path, err := exec.LookPath("adb")
	if err != nil {
		return nil, errors.New("未找到 ADB，请先安装 Android Platform Tools")
	}
	output, err := exec.Command(path, "devices", "-l").Output()
	if err != nil {
		return nil, fmt.Errorf("读取 ADB 设备失败: %w", err)
	}

	var devices []domain.Device
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		device := domain.Device{Serial: fields[0], State: fields[1]}
		for _, field := range fields[2:] {
			if value, ok := strings.CutPrefix(field, "model:"); ok {
				device.Model = value
			}
		}
		if device.State == "device" {
			device.Clients = listGameClients(path, device.Serial, "yokasgs://?payload=sanguoclip-probe")
		}
		devices = append(devices, device)
	}
	return devices, scanner.Err()
}

func (m *Manager) Start(id, serial, packageName, deepLink, localPath string) error {
	devices, err := ListDevices()
	if err != nil {
		return err
	}
	authorized := false
	for _, device := range devices {
		if device.Serial == serial && device.State == "device" {
			authorized = true
			break
		}
	}
	if !authorized {
		return errors.New("所选设备未连接或未授权")
	}

	adb, _ := exec.LookPath("adb")
	packageName, err = selectGameClient(listGameClients(adb, serial, deepLink), packageName)
	if err != nil {
		return err
	}
	remotePath := "/sdcard/Movies/SanguoClip/" + id + ".mp4"
	if output, err := exec.Command(adb, "-s", serial, "shell", "mkdir", "-p", "/sdcard/Movies/SanguoClip").CombinedOutput(); err != nil {
		return fmt.Errorf("准备设备录制目录失败: %s", commandError(err, output))
	}
	if err := ensureParent(localPath); err != nil {
		return err
	}
	if packageName != "" {
		_ = exec.Command(adb, "-s", serial, "shell", "am", "force-stop", packageName).Run()
	}

	cmd := exec.Command(adb, "-s", serial, "shell", "screenrecord", remotePath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动屏幕录制失败: %w", err)
	}
	current := &session{
		cmd:        cmd,
		adbPath:    adb,
		serial:     serial,
		remotePath: remotePath,
		localPath:  localPath,
		done:       make(chan error, 1),
	}
	go func() {
		current.done <- cmd.Wait()
	}()

	m.mu.Lock()
	m.sessions[id] = current
	m.mu.Unlock()

	time.Sleep(500 * time.Millisecond)
	select {
	case waitErr := <-current.done:
		m.removeSession(id)
		if waitErr == nil {
			return errors.New("设备屏幕录制意外结束")
		}
		return fmt.Errorf("设备屏幕录制启动后立即退出: %w", waitErr)
	default:
	}
	current.setRemotePID(findScreenrecordPID(adb, serial))
	startCommand := "am start -a android.intent.action.VIEW -d " + shellQuote(deepLink)
	if packageName != "" {
		startCommand += " -p " + shellQuote(packageName)
	}
	output, err := exec.Command(adb, "-s", serial, "shell", startCommand).CombinedOutput()
	if err != nil {
		_, _ = m.stopProcess(id, false)
		return fmt.Errorf("唤起三国杀客户端失败: %s", commandError(err, output))
	}
	return nil
}

func (m *Manager) Stop(id string) (string, error) {
	current, err := m.stopProcess(id, true)
	if err != nil {
		return "", err
	}
	adb := current.adbPath
	output, pullErr := exec.Command(adb, "-s", current.serial, "pull", current.remotePath, current.localPath).CombinedOutput()
	if pullErr != nil {
		return "", fmt.Errorf("拉取录制视频失败: %s", commandError(pullErr, output))
	}
	_ = exec.Command(adb, "-s", current.serial, "shell", "rm", "-f", current.remotePath).Run()
	return current.localPath, nil
}

func (m *Manager) stopProcess(id string, remove bool) (*session, error) {
	m.mu.Lock()
	current, ok := m.sessions[id]
	if ok && remove {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if !ok {
		return nil, errors.New("录制任务不存在或已停止")
	}

	stopRemoteRecording(current)
	select {
	case <-current.done:
	case <-time.After(8 * time.Second):
		if current.cmd.Process != nil {
			_ = current.cmd.Process.Kill()
		}
		<-current.done
	}
	if !remove {
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
	}
	return current, nil
}

func (m *Manager) removeSession(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

func findScreenrecordPID(adb, serial string) string {
	output, err := exec.Command(adb, "-s", serial, "shell", "pidof", "screenrecord").Output()
	if err != nil {
		return ""
	}
	return firstPID(string(output))
}

func firstPID(output string) string {
	for _, field := range strings.Fields(output) {
		pid, err := strconv.Atoi(field)
		if err == nil && pid > 0 {
			return field
		}
	}
	return ""
}

func listGameClients(adb, serial, deepLink string) []domain.GameClient {
	query := "cmd package query-activities --brief -a android.intent.action.VIEW -d " + shellQuote(deepLink)
	output, err := exec.Command(adb, "-s", serial, "shell", query).Output()
	if err != nil {
		return nil
	}
	clients := parseGameClients(string(output))
	for index := range clients {
		clients[index].Channel = detectChannel(adb, serial, clients[index].PackageName)
	}
	sort.Slice(clients, func(left, right int) bool {
		if clients[left].Channel == clients[right].Channel {
			return clients[left].PackageName < clients[right].PackageName
		}
		return clients[left].Channel < clients[right].Channel
	})
	return clients
}

func parseGameClients(output string) []domain.GameClient {
	seen := make(map[string]bool)
	var clients []domain.GameClient
	for _, line := range strings.Split(output, "\n") {
		component := strings.TrimSpace(line)
		packageName, activity, ok := strings.Cut(component, "/")
		if !ok || packageName == "" || activity == "" || strings.ContainsAny(packageName, " \t:") || seen[packageName] {
			continue
		}
		seen[packageName] = true
		clients = append(clients, domain.GameClient{
			PackageName: packageName,
			Activity:    component,
		})
	}
	return clients
}

func selectGameClient(clients []domain.GameClient, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		if len(clients) == 1 {
			return clients[0].PackageName, nil
		}
		if len(clients) > 1 {
			return "", errors.New("检测到多个三国杀客户端，请选择要录制的渠道")
		}
		return "", nil
	}
	for _, client := range clients {
		if client.PackageName == requested {
			return requested, nil
		}
	}
	return "", errors.New("所选三国杀渠道客户端未安装或不支持分享链接")
}

func detectChannel(adb, serial, packageName string) string {
	output, _ := exec.Command(adb, "-s", serial, "shell", "pm", "list", "packages", "-i", packageName).Output()
	installer := strings.ToLower(string(output))
	lowerPackage := strings.ToLower(packageName)
	switch {
	case strings.Contains(installer, "com.huawei.appmarket"),
		strings.Contains(installer, "com.huawei.gamebox"),
		strings.Contains(lowerPackage, "huawei"):
		return "华为渠道服"
	case strings.HasSuffix(lowerPackage, ".uc"):
		return "UC 渠道服"
	case strings.Contains(lowerPackage, "xiaomi"):
		return "小米渠道服"
	case strings.Contains(lowerPackage, "oppo"):
		return "OPPO 渠道服"
	case strings.Contains(lowerPackage, "vivo"):
		return "vivo 渠道服"
	case packageName == "com.bf.sgs.hdexp":
		return "官方服"
	default:
		return "渠道客户端"
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func stopRemoteRecording(current *session) {
	pid := current.screenrecordPID()
	if pid == "" {
		pid = findScreenrecordPID(current.adbPath, current.serial)
	}
	if pid != "" {
		if exec.Command(current.adbPath, "-s", current.serial, "shell", "kill", "-2", pid).Run() == nil {
			return
		}
	}
	_ = exec.Command(current.adbPath, "-s", current.serial, "shell", "pkill", "-2", "screenrecord").Run()
}

func (s *session) setRemotePID(pid string) {
	s.mu.Lock()
	s.remotePID = pid
	s.mu.Unlock()
}

func (s *session) screenrecordPID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.remotePID
}

func ensureParent(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建本地媒体目录失败: %w", err)
	}
	return nil
}

func commandError(err error, output []byte) string {
	message := strings.TrimSpace(string(output))
	if message != "" {
		return message
	}
	return err.Error()
}
