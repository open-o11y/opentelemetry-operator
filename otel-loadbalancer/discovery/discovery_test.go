package discovery

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	gokitlog "github.com/go-kit/log"
	"github.com/otel-loadbalancer/config"
	"github.com/stretchr/testify/assert"
)

var (
	appHome               = os.Getenv("APP_HOME")
	tmpFile               = appHome + "/config/testdata/tmp.json"
	testFile              = appHome + "/config/testdata/config_test.yaml"
	initialFileSdTestFile = appHome + "/config/testdata/file_sd_test.json"
	modFileSdTestFile     = appHome + "/config/testdata/file_sd_test_2.json"
)

func copyFileHelper(src string, dst string) error {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}
	return nil
}

func copyFile(t testing.TB, src string, dst string) {
	t.Helper()
	err := ioutil.WriteFile(tmpFile, []byte(""), 0644)
	assert.NoError(t, err)

	err = copyFileHelper(src, tmpFile)
	assert.NoError(t, err)
	err = copyFileHelper(dst, src)
	assert.NoError(t, err)
	err = copyFileHelper(tmpFile, dst)
	assert.NoError(t, err)

	err = os.Remove(tmpFile)
	assert.NoError(t, err)
}

func TestTargetDiscovery(t *testing.T) {
	cfg, err := config.Load(testFile)
	assert.NoError(t, err)
	manager := NewManager(context.Background(), gokitlog.NewNopLogger())

	t.Run("should discover targets", func(t *testing.T) {
		targets, err := manager.ApplyConfig(cfg)
		assert.NoError(t, err)

		actualTargets := []string{}
		expectedTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000"}

		assert.Len(t, targets, 5)
		for _, targets := range targets {
			actualTargets = append(actualTargets, targets.Target)
		}

		sort.Strings(expectedTargets)
		sort.Strings(actualTargets)

		assert.Equal(t, expectedTargets, actualTargets)

	})

	t.Run("should update targets", func(t *testing.T) {
		targets, err := manager.ApplyConfig(cfg)
		assert.NoError(t, err)

		actualTargets := []string{}
		expectedTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000", "promfile.domain:4000"}

		copyFile(t, initialFileSdTestFile, modFileSdTestFile)

		targets, err = manager.Targets()
		assert.NoError(t, err)

		assert.Len(t, targets, 6)
		for _, targets := range targets {
			actualTargets = append(actualTargets, targets.Target)
		}

		sort.Strings(expectedTargets)
		sort.Strings(actualTargets)

		assert.Equal(t, expectedTargets, actualTargets)

		copyFile(t, modFileSdTestFile, initialFileSdTestFile)

	})
}
