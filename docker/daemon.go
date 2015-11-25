// +build daemon

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/uuid"
	apiserver "github.com/tiborvass/docker/api/server"
	"github.com/tiborvass/docker/cli"
	"github.com/tiborvass/docker/cliconfig"
	"github.com/tiborvass/docker/daemon"
	"github.com/tiborvass/docker/daemon/logger"
	"github.com/tiborvass/docker/dockerversion"
	"github.com/tiborvass/docker/opts"
	flag "github.com/tiborvass/docker/pkg/mflag"
	"github.com/tiborvass/docker/pkg/pidfile"
	"github.com/tiborvass/docker/pkg/signal"
	"github.com/tiborvass/docker/pkg/system"
	"github.com/tiborvass/docker/pkg/timeutils"
	"github.com/tiborvass/docker/pkg/tlsconfig"
	"github.com/tiborvass/docker/registry"
	"github.com/tiborvass/docker/utils"
)

const daemonUsage = "       docker daemon [ --help | ... ]\n"

var (
	daemonCli cli.Handler = NewDaemonCli()
)

func presentInHelp(usage string) string { return usage }
func absentFromHelp(string) string      { return "" }

// NewDaemonCli returns a pre-configured daemon CLI
func NewDaemonCli() *DaemonCli {
	daemonFlags = cli.Subcmd("daemon", nil, "Enable daemon mode", true)

	// TODO(tiborvass): remove InstallFlags?
	daemonConfig := new(daemon.Config)
	daemonConfig.LogConfig.Config = make(map[string]string)
	daemonConfig.ClusterOpts = make(map[string]string)
	daemonConfig.InstallFlags(daemonFlags, presentInHelp)
	daemonConfig.InstallFlags(flag.CommandLine, absentFromHelp)
	registryOptions := new(registry.Options)
	registryOptions.InstallFlags(daemonFlags, presentInHelp)
	registryOptions.InstallFlags(flag.CommandLine, absentFromHelp)
	daemonFlags.Require(flag.Exact, 0)

	return &DaemonCli{
		Config:          daemonConfig,
		registryOptions: registryOptions,
	}
}

func migrateKey() (err error) {
	// Migrate trust key if exists at ~/.docker/key.json and owned by current user
	oldPath := filepath.Join(cliconfig.ConfigDir(), defaultTrustKeyFile)
	newPath := filepath.Join(getDaemonConfDir(), defaultTrustKeyFile)
	if _, statErr := os.Stat(newPath); os.IsNotExist(statErr) && currentUserIsOwner(oldPath) {
		defer func() {
			// Ensure old path is removed if no error occurred
			if err == nil {
				err = os.Remove(oldPath)
			} else {
				logrus.Warnf("Key migration failed, key file not removed at %s", oldPath)
				os.Remove(newPath)
			}
		}()

		if err := system.MkdirAll(getDaemonConfDir(), os.FileMode(0644)); err != nil {
			return fmt.Errorf("Unable to create daemon configuration directory: %s", err)
		}

		newFile, err := os.OpenFile(newPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("error creating key file %q: %s", newPath, err)
		}
		defer newFile.Close()

		oldFile, err := os.Open(oldPath)
		if err != nil {
			return fmt.Errorf("error opening key file %q: %s", oldPath, err)
		}
		defer oldFile.Close()

		if _, err := io.Copy(newFile, oldFile); err != nil {
			return fmt.Errorf("error copying key: %s", err)
		}

		logrus.Infof("Migrated key from %s to %s", oldPath, newPath)
	}

	return nil
}

// DaemonCli represents the daemon CLI.
type DaemonCli struct {
	*daemon.Config
	registryOptions *registry.Options
}

func getGlobalFlag() (globalFlag *flag.Flag) {
	defer func() {
		if x := recover(); x != nil {
			switch f := x.(type) {
			case *flag.Flag:
				globalFlag = f
			default:
				panic(x)
			}
		}
	}()
	visitor := func(f *flag.Flag) { panic(f) }
	commonFlags.FlagSet.Visit(visitor)
	clientFlags.FlagSet.Visit(visitor)
	return
}

// CmdDaemon is the daemon command, called the raw arguments after `docker daemon`.
func (cli *DaemonCli) CmdDaemon(args ...string) error {
	// warn from uuid package when running the daemon
	uuid.Loggerf = logrus.Warnf

	if !commonFlags.FlagSet.IsEmpty() || !clientFlags.FlagSet.IsEmpty() {
		// deny `docker -D daemon`
		illegalFlag := getGlobalFlag()
		fmt.Fprintf(os.Stderr, "invalid flag '-%s'.\nSee 'docker daemon --help'.\n", illegalFlag.Names[0])
		os.Exit(1)
	} else {
		// allow new form `docker daemon -D`
		flag.Merge(daemonFlags, commonFlags.FlagSet)
	}

	daemonFlags.ParseFlags(args, true)
	commonFlags.PostParse()

	if commonFlags.TrustKey == "" {
		commonFlags.TrustKey = filepath.Join(getDaemonConfDir(), defaultTrustKeyFile)
	}

	if utils.ExperimentalBuild() {
		logrus.Warn("Running experimental build")
	}

	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: timeutils.RFC3339NanoFixed})

	if err := setDefaultUmask(); err != nil {
		logrus.Fatalf("Failed to set umask: %v", err)
	}

	if len(cli.LogConfig.Config) > 0 {
		if err := logger.ValidateLogOpts(cli.LogConfig.Type, cli.LogConfig.Config); err != nil {
			logrus.Fatalf("Failed to set log opts: %v", err)
		}
	}

	var pfile *pidfile.PIDFile
	if cli.Pidfile != "" {
		pf, err := pidfile.New(cli.Pidfile)
		if err != nil {
			logrus.Fatalf("Error starting daemon: %v", err)
		}
		pfile = pf
		defer func() {
			if err := pfile.Remove(); err != nil {
				logrus.Error(err)
			}
		}()
	}

	serverConfig := &apiserver.Config{
		Logging: true,
		Version: dockerversion.Version,
	}
	serverConfig = setPlatformServerConfig(serverConfig, cli.Config)

	defaultHost := opts.DefaultHost
	if commonFlags.TLSOptions != nil {
		if !commonFlags.TLSOptions.InsecureSkipVerify {
			// server requires and verifies client's certificate
			commonFlags.TLSOptions.ClientAuth = tls.RequireAndVerifyClientCert
		}
		tlsConfig, err := tlsconfig.Server(*commonFlags.TLSOptions)
		if err != nil {
			logrus.Fatal(err)
		}
		serverConfig.TLSConfig = tlsConfig
		defaultHost = opts.DefaultTLSHost
	}

	if len(commonFlags.Hosts) == 0 {
		commonFlags.Hosts = make([]string, 1)
	}
	for i := 0; i < len(commonFlags.Hosts); i++ {
		var err error
		if commonFlags.Hosts[i], err = opts.ParseHost(defaultHost, commonFlags.Hosts[i]); err != nil {
			logrus.Fatalf("error parsing -H %s : %v", commonFlags.Hosts[i], err)
		}
	}
	for _, protoAddr := range commonFlags.Hosts {
		protoAddrParts := strings.SplitN(protoAddr, "://", 2)
		if len(protoAddrParts) != 2 {
			logrus.Fatalf("bad format %s, expected PROTO://ADDR", protoAddr)
		}
		serverConfig.Addrs = append(serverConfig.Addrs, apiserver.Addr{Proto: protoAddrParts[0], Addr: protoAddrParts[1]})
	}
	api, err := apiserver.New(serverConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	if err := migrateKey(); err != nil {
		logrus.Fatal(err)
	}
	cli.TrustKeyPath = commonFlags.TrustKey

	registryService := registry.NewService(cli.registryOptions)
	d, err := daemon.NewDaemon(cli.Config, registryService)
	if err != nil {
		if pfile != nil {
			if err := pfile.Remove(); err != nil {
				logrus.Error(err)
			}
		}
		logrus.Fatalf("Error starting daemon: %v", err)
	}

	logrus.Info("Daemon has completed initialization")

	logrus.WithFields(logrus.Fields{
		"version":     dockerversion.Version,
		"commit":      dockerversion.GitCommit,
		"execdriver":  d.ExecutionDriver().Name(),
		"graphdriver": d.GraphDriver().String(),
	}).Info("Docker daemon")

	api.InitRouters(d)

	// The serve API routine never exits unless an error occurs
	// We need to start it as a goroutine and wait on it so
	// daemon doesn't exit
	serveAPIWait := make(chan error)
	go func() {
		if err := api.ServeAPI(); err != nil {
			logrus.Errorf("ServeAPI error: %v", err)
			serveAPIWait <- err
			return
		}
		serveAPIWait <- nil
	}()

	signal.Trap(func() {
		api.Close()
		<-serveAPIWait
		shutdownDaemon(d, 15)
		if pfile != nil {
			if err := pfile.Remove(); err != nil {
				logrus.Error(err)
			}
		}
	})

	// after the daemon is done setting up we can notify systemd api
	notifySystem()

	// Daemon is fully initialized and handling API traffic
	// Wait for serve API to complete
	errAPI := <-serveAPIWait
	shutdownDaemon(d, 15)
	if errAPI != nil {
		if pfile != nil {
			if err := pfile.Remove(); err != nil {
				logrus.Error(err)
			}
		}
		logrus.Fatalf("Shutting down due to ServeAPI error: %v", errAPI)
	}
	return nil
}

// shutdownDaemon just wraps daemon.Shutdown() to handle a timeout in case
// d.Shutdown() is waiting too long to kill container or worst it's
// blocked there
func shutdownDaemon(d *daemon.Daemon, timeout time.Duration) {
	ch := make(chan struct{})
	go func() {
		d.Shutdown()
		close(ch)
	}()
	select {
	case <-ch:
		logrus.Debug("Clean shutdown succeeded")
	case <-time.After(timeout * time.Second):
		logrus.Error("Force shutdown daemon")
	}
}
