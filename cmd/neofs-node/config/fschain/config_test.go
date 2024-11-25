package fschainconfig_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nspcc-dev/neofs-node/cmd/neofs-node/config"
	fschainconfig "github.com/nspcc-dev/neofs-node/cmd/neofs-node/config/fschain"
	configtest "github.com/nspcc-dev/neofs-node/cmd/neofs-node/config/test"
	"github.com/stretchr/testify/require"
)

func TestFSChainSection(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		empty := configtest.EmptyConfig()

		require.Panics(t, func() { fschainconfig.Endpoints(empty) })
		require.Equal(t, fschainconfig.DialTimeoutDefault, fschainconfig.DialTimeout(empty))
		require.Equal(t, fschainconfig.CacheTTLDefault, fschainconfig.CacheTTL(empty))
		require.Equal(t, 5, fschainconfig.ReconnectionRetriesNumber(empty))
		require.Equal(t, 5*time.Second, fschainconfig.ReconnectionRetriesDelay(empty))
	})

	const path = "../../../../config/example/node"

	rpcs := []string{"wss://rpc1.morph.fs.neo.org:40341/ws", "wss://rpc2.morph.fs.neo.org:40341/ws"}

	var fileConfigTest = func(c *config.Config) {
		require.Equal(t, rpcs, fschainconfig.Endpoints(c))
		require.Equal(t, 30*time.Second, fschainconfig.DialTimeout(c))
		require.Equal(t, 15*time.Second, fschainconfig.CacheTTL(c))
		require.Equal(t, 6, fschainconfig.ReconnectionRetriesNumber(c))
		require.Equal(t, 6*time.Second, fschainconfig.ReconnectionRetriesDelay(c))
	}

	configtest.ForEachFileType(path, fileConfigTest)

	t.Run("ENV", func(t *testing.T) {
		configtest.ForEnvFileType(path, fileConfigTest)
	})

	t.Run("compatibility with morph section", func(t *testing.T) {
		data := []byte(`
morph:
  dial_timeout: 30s
  cache_ttl: 15s
  reconnections_number: 6
  reconnections_delay: 6s
  endpoints:
    - wss://rpc1.morph.fs.neo.org:40341/ws
    - wss://rpc2.morph.fs.neo.org:40341/ws
`)

		morphPath := filepath.Join(t.TempDir(), "morph.yaml")
		require.NoError(t, os.WriteFile(morphPath, data, 0o640))

		var p config.Prm

		os.Clearenv()

		c := config.New(p,
			config.WithConfigFile(morphPath),
			config.WithValidate(false),
		)

		fileConfigTest(c)
	})
}
