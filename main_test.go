package main

import (
	"testing"
	"github.com/stretchr/testify/require"
	"os"
	"github.com/bitrise-io/go-utils/command"
	"io/ioutil"
	"github.com/bitrise-io/go-utils/log"
)

func TestValidateConfigApkPathDontUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPath = "/dev/null"
	require.NoError(t, configs.validate())
}

func TestValidateConfigApkPathUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.IsUseApkPathList = "true"
	configs.ApkPath = "/dev/null"
	require.Error(t, configs.validate())
}

func TestValidateConfigApkPathListUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.IsUseApkPathList = "true"
	configs.ApkPathList = []string{"/dev/null"}
	require.NoError(t, configs.validate())
}

func TestValidateConfigApkPathListDontUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"/dev/null"}
	require.Error(t, configs.validate())
}

func TestValidateConfigApkPathAndListListUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.IsUseApkPathList = "true"
	configs.ApkPath = "/dev/null"
	configs.ApkPathList = []string{"/dev/null"}
	require.NoError(t, configs.validate())
}

func TestActualApkPathListUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.IsUseApkPathList = "true"
	configs.ApkPath = "single apk"
	configs.ApkPathList = []string{"apk from the list"}
	require.Equal(t, configs.ApkPathList, configs.getActualApkPathList())
}

func TestActualApkPathListDontUseList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPath = "single apk"
	configs.ApkPathList = []string{"apk from the list"}
	require.Equal(t, []string{configs.ApkPath}, configs.getActualApkPathList())
}

func TestExportSingleValue(t *testing.T) {
	envstore, err := setupTempEnvstore()
	require.NoError(t, err)
	defer func() {
		err := os.Remove(envstore)
		if err != nil {
			log.Errorf("Could not delete temporary file %s, error %s", envstore, err)
		}
	}()

	require.NoError(t, os.Setenv("ENVMAN_ENVSTORE_PATH", envstore))
	exportSingleEnvironmentOrLogWarn("SAMPLE_KEY", "sample value")

	environment, err := command.New("envman", "print").RunAndReturnTrimmedOutput()
	require.NoError(t, err)

	require.Contains(t, environment, "SAMPLE_KEY")
	require.Contains(t, environment, "sample value")
	require.NoError(t, os.Remove(envstore))
}

func TestExportMultipleValues(t *testing.T) {
	envstore, err := setupTempEnvstore()
	require.NoError(t, err)
	defer func() {
		err := os.Remove(envstore)
		if err != nil {
			log.Warnf("Could not delete temporary file %s, error %s", envstore, err)
		}
	}()

	require.NoError(t, os.Setenv("ENVMAN_ENVSTORE_PATH", envstore))
	exportEnvironmentSliceOrLogWarn("SAMPLE_KEY", []string{"sample value", "sample value 2"})

	environment, err := command.New("envman", "print").RunAndReturnTrimmedOutput()
	require.NoError(t, err)

	require.Contains(t, environment, "SAMPLE_KEY")
	require.Contains(t, environment, "sample value|sample value 2")
}

func createDummyConfigs() ConfigsModel {
	return ConfigsModel{
		IsUseApkPathList:"false",
		APIToken:"token",
		NotesType:"0",
		Notify:"2",
		Status:"2",
		Mandatory:"false",
		ApkPath:"",
		MappingPath:"",
		ApkPathList: []string{""},
	}
}

func setupTempEnvstore() (string, error) {
	file, err := ioutil.TempFile("", "envstore")
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}
