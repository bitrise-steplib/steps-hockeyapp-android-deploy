package main

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestValidateConfigApkPath(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPath = "/dev/null"
	configs.normalize()
	require.NoError(t, configs.validate())
}

func TestValidateConfigApkList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"/dev/null", "/dev/zero"}
	configs.normalize()
	require.NoError(t, configs.validate())
}

func TestValidateConfigInvalidApkPath(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPath = "invalid"
	configs.normalize()
	require.Error(t, configs.validate())
}

func TestValidateConfigInvalidApkPathInList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"invalid", "/dev/zero"}
	configs.normalize()
	require.Error(t, configs.validate())
}

func TestValidateConfigApkAndMappingPath(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPath = "/dev/null"
	configs.MappingPath = "/dev/null"
	configs.normalize()
	require.NoError(t, configs.validate())
}

func TestValidateConfigApkListAndMapping(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"/dev/null", "/dev/zero"}
	configs.MappingPath = "/dev/null"
	configs.normalize()
	require.NoError(t, configs.validate())
}

func TestValidateConfigInvalidMappingPath(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPath = "/dev/null"
	configs.MappingPath = "invalid"
	configs.normalize()
	require.Error(t, configs.validate())
}

func TestValidateConfigInvalidMappingPathInList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"/dev/null", "/dev/zero"}
	configs.MappingPathList = []string{"/dev/null", "invalid"}
	configs.normalize()
	require.Error(t, configs.validate())
}

func TestMappingPathListExtendedIfTooShort(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1", "apk2"}
	configs.MappingPathList = []string{"mapping1"}
	configs.extendMappingPathList()
	require.Len(t, configs.MappingPathList, 2)
	require.Equal(t, []string{"mapping1", ""}, configs.MappingPathList)
}

func TestMappingPathListNotModifiedIfLargeEnough(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1", "apk2"}
	configs.MappingPathList = []string{"mapping1", "mapping2"}
	configs.extendMappingPathList()
	require.Len(t, configs.MappingPathList, 2)
	require.Equal(t, []string{"mapping1", "mapping2"}, configs.MappingPathList)
}

func TestMappingPathListNotModifiedIfLargerThanNeeded(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1", "apk2"}
	configs.MappingPathList = []string{"mapping1", "mapping2", "mapping3"}
	configs.extendMappingPathList()
	require.Len(t, configs.MappingPathList, 3)
	require.Equal(t, []string{"mapping1", "mapping2", "mapping3"}, configs.MappingPathList)
}

func TestApkPathAppendedToListIfNotPresent(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1"}
	configs.ApkPath = "apk2"
	configs.MappingPathList = []string{"mapping1"}
	configs.normalize()
	require.Len(t, configs.MappingPathList, 2)
	require.Equal(t, []string{"apk1", "apk2"}, configs.ApkPathList)
	require.Equal(t, []string{"mapping1", ""}, configs.MappingPathList)
}

func TestApkPathAppendedToListWithMappingIfNotPresent(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1"}
	configs.ApkPath = "apk2"
	configs.MappingPathList = []string{"mapping1"}
	configs.MappingPath = "mapping2"
	configs.normalize()
	require.Len(t, configs.MappingPathList, 2)
	require.Equal(t, []string{"apk1", "apk2"}, configs.ApkPathList)
	require.Equal(t, []string{"mapping1", "mapping2"}, configs.MappingPathList)
}

func TestApkPathNotAppendedToListWithMappingIfAlreadyPresent(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1"}
	configs.ApkPath = "apk1"
	configs.normalize()
	require.Len(t, configs.MappingPathList, 1)
	require.Equal(t, []string{"apk1"}, configs.ApkPathList)
}

func TestSingleMappingOverriddenByValueFromList(t *testing.T) {
	configs := createDummyConfigs()
	configs.ApkPathList = []string{"apk1"}
	configs.ApkPath = "apk1"
	configs.MappingPathList = []string{"mapping1"}
	configs.MappingPath = "mapping2"
	configs.normalize()
	require.Len(t, configs.MappingPathList, 1)
	require.Equal(t, []string{"apk1"}, configs.ApkPathList)
	require.Equal(t, []string{"mapping1"}, configs.MappingPathList)
}

func createDummyConfigs() ConfigsModel {
	return ConfigsModel{
		APIToken:"token",
		NotesType:"0",
		Notify:"2",
		Status:"2",
		Mandatory:"false",
	}
}
