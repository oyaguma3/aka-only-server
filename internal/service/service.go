package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const serviceTemplate = `[Unit]
Description={{.Description}}
After=network.target

[Service]
Type=simple
ExecStart={{.ExecStart}}
WorkingDirectory={{.WorkingDirectory}}
Restart=always
User=root
EnvironmentFile={{.EnvFile}}

[Install]
WantedBy=multi-user.target
`

type ServiceConfig struct {
	Description      string
	ExecStart        string
	WorkingDirectory string
	EnvFile          string
}

func Install(serviceName, description string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	absExePath, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute executable path: %w", err)
	}

	workDir := filepath.Dir(absExePath)
	envFile := filepath.Join(workDir, ".env")

	config := ServiceConfig{
		Description:      description,
		ExecStart:        absExePath,
		WorkingDirectory: workDir,
		EnvFile:          envFile,
	}

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	f, err := os.Create(servicePath)
	if err != nil {
		return fmt.Errorf("failed to create service file at %s: %w", servicePath, err)
	}
	defer f.Close()

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}

	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("failed to execute service template: %w", err)
	}

	fmt.Printf("Service file created at %s\n", servicePath)

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	fmt.Println("Systemd daemon reloaded.")

	// Enable service
	if err := exec.Command("systemctl", "enable", serviceName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	fmt.Printf("Service %s enabled.\n", serviceName)

	return nil
}

func Uninstall(serviceName string) error {
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	// Stop service
	_ = exec.Command("systemctl", "stop", serviceName).Run()

	// Disable service
	_ = exec.Command("systemctl", "disable", serviceName).Run()

	// Remove file
	if err := os.Remove(servicePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file: %w", err)
		}
	} else {
		fmt.Printf("Service file %s removed.\n", servicePath)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	fmt.Println("Systemd daemon reloaded.")

	return nil
}
