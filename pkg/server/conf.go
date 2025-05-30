package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/file"
	"github.com/heetch/confita/backend/flags"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Logging    LogConfig `yaml:"logging"`
	Listen     string    `config:"port,description=Listen address" yaml:"port"`
	HttpListen string    `config:"http_port,description=Http listen address" yaml:"http_port"`
	// FIXME: Dev* in DevAuth, drop it
	DevLogin           string        `config:"dev-login,description=Default device login" yaml:"dev_login"`
	DevPass            string        `config:"dev-pass,description=Default device password" yaml:"dev_pass"`
	DevUseAgent        bool          `config:"dev-use-agent" yaml:"dev_use_agent"`
	DevAuth            authAppConfig `yaml:"dev_auth"`
	ConfFile           string        `config:"conf-file,description=Path to config file. '-' for stdin"`
	DevConf            string        `config:"dev-conf,Path to yaml with device types" yaml:"dev_conf"`
	Tls                bool          `config:"tls,description=Connection uses TLS if true, else plain TCP" yaml:"tls"`
	CertFile           string        `config:"cert-file,description=The TLS cert file" yaml:"cert_file"`
	KeyFile            string        `config:"key-file,description=The TLS key file" yaml:"key_file"`
	BasicAuth          string        `config:"basic-auth,description=Authenticate client using Basic auth" yaml:"basic_auth"`
	DisableTcp         bool          `config:"disable_tcp,description=Disable TCP listener" yaml:"disable_tcp"`
	UnixSocket         string        `config:"unix-socket,description=Unix socket path" yaml:"unix_socket"`
	Debug              bool          `config:"debug,short=d,description=Set debug log level"`
	DefaultReadTimeout time.Duration `config:"default-read-timeout,description=Default read timeout" yaml:"default_read_timeout"`
	DefaultCmdTimeout  time.Duration `config:"default-cmd-timeout,description=Default command timeout" yaml:"default_cmd_timeout"`
}

type LogConfig struct {
	Level zapcore.Level `yaml:"level"`
	Json  bool          `yaml:"json"`
}

func newDefaultConf() Config {
	return Config{
		Listen: "127.0.0.1:50051",
		Logging: LogConfig{
			Level: zapcore.InfoLevel,
			Json:  false,
		},
	}
}

func newDefaultConfFlag() Config {
	return Config{
		Logging: LogConfig{
			Level: zapcore.InfoLevel,
			Json:  false,
		},
	}
}

func LoadConf() (Config, error) {
	loader := confita.NewLoader(env.NewBackend(), flags.NewBackend())
	flagCfg := newDefaultConfFlag()
	err := loader.Load(context.Background(), &flagCfg)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if len(flagCfg.ConfFile) > 0 {
		backends := []backend.Backend{env.NewBackend()}

		if flagCfg.ConfFile == "-" {
			stdinData, err := io.ReadAll(os.Stdin)
			if err != nil {
				return Config{}, err
			}
			backends = append(backends, newConfVarBackend(stdinData, "yaml"))
		} else {
			backends = append(backends, file.NewBackend(flagCfg.ConfFile))
		}
		loader := confita.NewLoader(backends...)
		pcfg := newDefaultConf()
		err = loader.Load(context.Background(), &pcfg)
		if err != nil {
			return Config{}, err
		}
		// merge with flags
		if len(flagCfg.DevLogin) > 0 {
			pcfg.DevLogin = flagCfg.DevLogin
		}
		if len(flagCfg.Listen) > 0 {
			pcfg.Listen = flagCfg.Listen
		}
		if len(flagCfg.DevPass) > 0 {
			pcfg.DevPass = flagCfg.DevPass
		}
		cfg = pcfg
	} else {
		cfg = flagCfg
	}
	if flagCfg.Debug {
		cfg.Logging.Level = zapcore.DebugLevel
	}
	return cfg, nil
}

func newConfVarBackend(data []byte, format string) *fileVarBackend {
	return &fileVarBackend{
		data:   data,
		format: format,
	}
}

type fileVarBackend struct {
	data   []byte
	format string
}

func (m fileVarBackend) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (m *fileVarBackend) Unmarshal(ctx context.Context, to interface{}) error {
	var err error
	switch m.format {
	case "json":
		err = json.NewDecoder(bytes.NewReader(m.data)).Decode(to)
	case "yml":
		fallthrough
	case "yaml":
		err = yaml.NewDecoder(bytes.NewReader(m.data)).Decode(to)
	default:
		err = errors.Errorf("unsupported extension \"%s\"", m.format)
	}

	return errors.Wrapf(err, "failed to decode file")
}

func (m fileVarBackend) Name() string {
	return "filevar"
}
