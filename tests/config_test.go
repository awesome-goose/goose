package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awesome-goose/goose/config"
	test "github.com/awesome-goose/goose/testing"
)

func TestConfig(t *testing.T) {
	test.NewSuiteRunner(t, &ConfigSuite{}).Run()
}

type ConfigSuite struct {
	test.Suite
	tmpDir  string
	cfg     *config.Config
	cleanup func()
}

func (s *ConfigSuite) SetupTest() {
	var err error
	s.tmpDir, err = os.MkdirTemp("", "config_test")
	s.T.Expect(err).ToBeNil()

	s.cleanup = func() {
		os.RemoveAll(s.tmpDir)
	}

	// Create a test YAML file
	appYaml := `
name: "test-app"
version: "1.0.0"
debug: "true"
port: "8080"
rate: "3.14"
nested:
  key: "nested-value"
  deep:
    key: "deep-value"
`
	err = os.WriteFile(filepath.Join(s.tmpDir, "app.yaml"), []byte(appYaml), 0644)
	s.T.Expect(err).ToBeNil()

	s.cfg, err = config.NewConfig(config.AppConfigPath(s.tmpDir))
	s.T.Expect(err).ToBeNil()
}

func (s *ConfigSuite) TeardownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *ConfigSuite) TestGet_SimpleValue() {
	value := s.cfg.Get("app.name")
	s.T.Expect(value).ToEqual("test-app")
}

func (s *ConfigSuite) TestGet_NestedValue() {
	value := s.cfg.Get("app.nested.key")
	s.T.Expect(value).ToEqual("nested-value")
}

func (s *ConfigSuite) TestGet_DeepNestedValue() {
	value := s.cfg.Get("app.nested.deep.key")
	s.T.Expect(value).ToEqual("deep-value")
}

func (s *ConfigSuite) TestGet_NonExistentKey() {
	value := s.cfg.Get("app.nonexistent")
	s.T.Expect(value).ToEqual("")
}

func (s *ConfigSuite) TestGet_NonExistentNamespace() {
	value := s.cfg.Get("nonexistent.key")
	s.T.Expect(value).ToEqual("")
}

func (s *ConfigSuite) TestGetWithDefault_ExistingKey() {
	value := s.cfg.GetWithDefault("app.name", "default")
	s.T.Expect(value).ToEqual("test-app")
}

func (s *ConfigSuite) TestGetWithDefault_NonExistentKey() {
	value := s.cfg.GetWithDefault("app.nonexistent", "default-value")
	s.T.Expect(value).ToEqual("default-value")
}

func (s *ConfigSuite) TestGetInt_ValidInt() {
	value := s.cfg.GetInt("app.port")
	s.T.Expect(value).ToEqual(8080)
}

func (s *ConfigSuite) TestGetInt_NonExistentKey() {
	value := s.cfg.GetInt("app.nonexistent")
	s.T.Expect(value).ToEqual(0)
}

func (s *ConfigSuite) TestGetBool_ValidBool() {
	value := s.cfg.GetBool("app.debug")
	s.T.Expect(value).ToBeTrue()
}

func (s *ConfigSuite) TestGetBool_NonExistentKey() {
	value := s.cfg.GetBool("app.nonexistent")
	s.T.Expect(value).ToBeFalse()
}

func (s *ConfigSuite) TestGetFloat_ValidFloat() {
	value := s.cfg.GetFloat("app.rate")
	s.T.Expect(value).ToEqual(3.14)
}

func (s *ConfigSuite) TestGetFloat_NonExistentKey() {
	value := s.cfg.GetFloat("app.nonexistent")
	s.T.Expect(value).ToEqual(0.0)
}

func (s *ConfigSuite) TestSet_NewKey() {
	s.cfg.Set("app.newkey", "newvalue")
	value := s.cfg.Get("app.newkey")
	s.T.Expect(value).ToEqual("newvalue")
}

func (s *ConfigSuite) TestSet_OverwriteKey() {
	s.cfg.Set("app.name", "new-name")
	value := s.cfg.Get("app.name")
	s.T.Expect(value).ToEqual("new-name")
}

func (s *ConfigSuite) TestSet_CreateNestedPath() {
	s.cfg.Set("app.created.nested.key", "created-value")
	value := s.cfg.Get("app.created.nested.key")
	s.T.Expect(value).ToEqual("created-value")
}

func (s *ConfigSuite) TestDir_ReturnsConfigDir() {
	dir := s.cfg.Dir()
	s.T.Expect(dir).ToEqual(s.tmpDir)
}

func (s *ConfigSuite) TestTree_ReturnsTree() {
	tree := s.cfg.Tree()
	s.T.Expect(tree).Not().ToBeNil()
	s.T.Expect(tree["app"]).Not().ToBeNil()
}

func (s *ConfigSuite) TestImport_ValidStruct() {
	type TestConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	testCfg := TestConfig{Host: "localhost", Port: 9000}
	err := s.cfg.Import("database", testCfg)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(s.cfg.Get("database.host")).ToEqual("localhost")
}

func (s *ConfigSuite) TestImport_EmptyNamespace() {
	type TestConfig struct {
		Host string `json:"host"`
	}

	err := s.cfg.Import("", TestConfig{})
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ConfigSuite) TestExport_ValidNamespace() {
	type AppConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	var appCfg AppConfig
	err := s.cfg.Export("app", &appCfg)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(appCfg.Name).ToEqual("test-app")
	s.T.Expect(appCfg.Version).ToEqual("1.0.0")
}

func (s *ConfigSuite) TestExport_NonExistentNamespace() {
	type TestConfig struct {
		Key string `yaml:"key"`
	}

	var cfg TestConfig
	err := s.cfg.Export("nonexistent", &cfg)
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ConfigSuite) TestNewConfig_EmptyDir() {
	emptyDir, _ := os.MkdirTemp("", "empty_config_test")
	defer os.RemoveAll(emptyDir)

	cfg, err := config.NewConfig(config.AppConfigPath(emptyDir))

	s.T.Expect(err).ToBeNil()
	s.T.Expect(cfg).Not().ToBeNil()
}

func (s *ConfigSuite) TestNewConfig_YmlExtension() {
	ymlDir, _ := os.MkdirTemp("", "yml_config_test")
	defer os.RemoveAll(ymlDir)

	ymlContent := `key: "yml-value"`
	os.WriteFile(filepath.Join(ymlDir, "test.yml"), []byte(ymlContent), 0644)

	cfg, err := config.NewConfig(config.AppConfigPath(ymlDir))

	s.T.Expect(err).ToBeNil()
	s.T.Expect(cfg.Get("test.key")).ToEqual("yml-value")
}
