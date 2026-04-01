package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	appName        = "VRChat Join Manager Agent"
	svcName        = "vjmagent"
	svcDisplayName = appName
)

// windowsService implements svc.Handler for Windows Service Control Manager.
type windowsService struct{}

func (ws *windowsService) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cacheDir, _ := os.UserCacheDir()
	appDir := filepath.Join(cacheDir, appName)

	errCh := make(chan error, 1)
	go func() { errCh <- run(ctx, appDir) }()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				cancel()
				if err := <-errCh; err != nil {
					log.Printf("run error on stop: %v", err)
				}
				return false, 0
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("run exited with error: %v", err)
			}
			return false, 0
		}
	}
}

func installService(exePath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %q already exists", svcName)
	}

	s, err = m.CreateService(svcName, exePath, mgr.Config{
		DisplayName: svcDisplayName,
		StartType:   mgr.StartAutomatic,
		ServiceType: 0x00000050, // SERVICE_USER_OWN_PROCESS
	})
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()
	log.Printf("Service %q installed as user service (starts on login).", svcName)
	return nil
}

func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", svcName, err)
	}
	defer s.Close()

	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err != nil {
			log.Printf("warning: stop service: %v", err)
		} else {
			for i := 0; i < 10; i++ {
				time.Sleep(500 * time.Millisecond)
				status, err = s.Query()
				if err != nil || status.State == svc.Stopped {
					break
				}
			}
		}
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	log.Printf("Service %q removed.", svcName)
	return nil
}
