package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/bitrise-io/depman/pathutil"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// -----------------------
// --- Constants
// -----------------------

const (
	hockeyAppDeployStatusKey = "HOCKEYAPP_DEPLOY_STATUS"
	hockeyAppDeployStatusSuccess = "success"
	hockeyAppDeployStatusFailed = "failed"

	hockeyAppDeployPublicURLKey = "HOCKEYAPP_DEPLOY_PUBLIC_URL"
	hockeyAppDeployBuildURLKey = "HOCKEYAPP_DEPLOY_BUILD_URL"
	hockeyAppDeployConfigURLKey = "HOCKEYAPP_DEPLOY_CONFIG_URL"

	hockeyAppDeployPublicURLListKey = "HOCKEYAPP_DEPLOY_PUBLIC_URL_LIST"
	hockeyAppDeployBuildURLListKey = "HOCKEYAPP_DEPLOY_BUILD_URL_LIST"
	hockeyAppDeployConfigURLListKey = "HOCKEYAPP_DEPLOY_CONFIG_URL_LIST"
)

// -----------------------
// --- Models
// -----------------------

// ConfigsModel ...
type ConfigsModel struct {
	IsUseApkPathList string
	ApkPath          string
	ApkPathList      []string
	MappingPath      string
	APIToken         string
	AppID            string
	Notes            string
	NotesType        string
	Notify           string
	Status           string
	Tags             string
	CommitSHA        string
	BuildServerURL   string
	RepositoryURL    string
	Mandatory        string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		IsUseApkPathList: os.Getenv("is_use_apk_path_list"),
		ApkPath:          os.Getenv("apk_path"),
		ApkPathList:      strings.Split(os.Getenv("apk_path_list"), "|"),
		MappingPath:      os.Getenv("mapping_path"),
		APIToken:         os.Getenv("api_token"),
		AppID:            os.Getenv("app_id"),
		Notes:            os.Getenv("notes"),
		NotesType:        os.Getenv("notes_type"),
		Notify:           os.Getenv("notify"),
		Status:           os.Getenv("status"),
		Tags:             os.Getenv("tags"),
		CommitSHA:        os.Getenv("commit_sha"),
		BuildServerURL:   os.Getenv("build_server_url"),
		RepositoryURL:    os.Getenv("repository_url"),
		Mandatory:        os.Getenv("mandatory"),
	}
}

func (configs ConfigsModel) print() {
	fmt.Println()
	log.Infof("Configs:")
	log.Printf(" - ApkPath: %s", configs.ApkPath)
	log.Printf(" - ApkPathList: %v", configs.ApkPathList)
	log.Printf(" - MappingPath: %s", configs.MappingPath)
	log.Printf(" - APIToken: %s", configs.APIToken)
	log.Printf(" - AppID: %s", configs.AppID)
	log.Printf(" - Notes: %s", configs.Notes)
	log.Printf(" - NotesType: %s", configs.NotesType)
	log.Printf(" - Notify: %s", configs.Notify)
	log.Printf(" - Status: %s", configs.Status)
	log.Printf(" - Tags: %s", configs.Tags)
	log.Printf(" - CommitSHA: %s", configs.CommitSHA)
	log.Printf(" - BuildServerURL: %s", configs.BuildServerURL)
	log.Printf(" - RepositoryURL: %s", configs.RepositoryURL)
	log.Printf(" - Mandatory: %s", configs.Mandatory)
}

func (configs ConfigsModel) getActualApkPathList() []string {
	if configs.IsUseApkPathList == "true" {
		return configs.ApkPathList
	}
	return []string{configs.ApkPath}
}

func (configs ConfigsModel) validate() error {
	// required
	for _, apkPath := range configs.getActualApkPathList() {
		if apkPath == "" {
			return errors.New("empty APK path specified")
		}
		if exist, err := pathutil.IsPathExists(apkPath); err != nil {
			return fmt.Errorf("Failed to check if APK file exist at: %s, error: %s", apkPath, err)
		} else if !exist {
			return fmt.Errorf("APK path not exist at: %s", apkPath)
		}
	}

	if configs.APIToken == "" {
		return errors.New("no APIToken parameter specified")
	}

	if configs.NotesType == "" {
		return errors.New("no NotesType parameter specified")
	}

	if configs.Notify == "" {
		return errors.New("no Notify parameter specified")
	}

	if configs.Status == "" {
		return errors.New("no Status parameter specified")
	}

	if configs.Mandatory == "" {
		return errors.New("no Mandatory parameter specified")
	}

	// optional
	if configs.MappingPath != "" {
		if exist, err := pathutil.IsPathExists(configs.MappingPath); err != nil {
			return fmt.Errorf("Failed to check if MappingPath exist at: %s, error: %s", configs.MappingPath, err)
		} else if !exist {
			return fmt.Errorf("MappingPath not exist at: %s", configs.MappingPath)
		}
	}

	return nil
}

// ResponseModel ...
type ResponseModel struct {
	ConfigURL string `json:"config_url"`
	PublicURL string `json:"public_url"`
	BuildURL  string `json:"build_url"`
}

func createRequest(url string, fields, files map[string]string) (*http.Request, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add fields
	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	// Add files
	for key, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		fw, err := w.CreateFormFile(key, file)
		if err != nil {
			return nil, err
		}
		if _, err = io.Copy(fw, f); err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}

// -----------------------
// --- Main
// -----------------------

func main() {
	configs := createConfigsModelFromEnvs()
	configs.print()
	if err := configs.validate(); err != nil {
		log.Errorf("Issue with input: %s", err)
		os.Exit(1)
	}

	if configs.Mandatory == "1" || configs.Mandatory == "true" {
		configs.Mandatory = "1"
	} else {
		configs.Mandatory = "0"
	}

	configURLs := []string{}
	buildURLs := []string{}
	publicURLs := []string{}

	for _, apkPath := range configs.getActualApkPathList() {
		responseModel, err := uploadBuild(configs, apkPath, configs.MappingPath)
		if err != nil {
			exportSingleEnvironmentOrLogWarn(hockeyAppDeployStatusKey, hockeyAppDeployStatusFailed)
			os.Exit(1)
		}
		configURLs = append(configURLs, responseModel.ConfigURL)
		buildURLs = append(buildURLs, responseModel.BuildURL)
		publicURLs = append(publicURLs, responseModel.PublicURL)
	}

	exportSingleEnvironmentOrLogWarn(hockeyAppDeployStatusKey, hockeyAppDeployStatusSuccess)

	exportSingleEnvironmentOrLogWarn(hockeyAppDeployBuildURLKey, buildURLs[0])
	exportSingleEnvironmentOrLogWarn(hockeyAppDeployConfigURLKey, configURLs[0])
	exportSingleEnvironmentOrLogWarn(hockeyAppDeployPublicURLKey, publicURLs[0])

	exportEnvironmentSliceOrLogWarn(hockeyAppDeployBuildURLListKey, buildURLs)
	exportEnvironmentSliceOrLogWarn(hockeyAppDeployConfigURLListKey, configURLs)
	exportEnvironmentSliceOrLogWarn(hockeyAppDeployPublicURLListKey, publicURLs)
}

func exportSingleEnvironmentOrLogWarn(key string, value string) {
	if err := exportEnvironmentWithEnvman(key, value); err != nil {
		log.Warnf("Failed to export %s, error: %s", key, err)
	}
}

func exportEnvironmentSliceOrLogWarn(key string, values []string) {
	if err := exportEnvironmentWithEnvman(key, strings.Join(values[:], "|")); err != nil {
		log.Warnf("Failed to export %s, error: %s", key, err)
	}
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func uploadBuild(configs ConfigsModel, apkPath string, mappingPath string) (ResponseModel, error) {
	//
	// Create request
	fmt.Println()
	log.Infof("Performing request")

	requestURL := "https://rink.hockeyapp.net/api/2/apps/upload"
	if configs.AppID != "" {
		requestURL = fmt.Sprintf("https://rink.hockeyapp.net/api/2/apps/%s/app_versions/upload", configs.AppID)
	}

	fields := map[string]string{
		"notes":            configs.Notes,
		"notes_type":       configs.NotesType,
		"notify":           configs.Notify,
		"status":           configs.Status,
		"mandatory":        configs.Mandatory,
		"tags":             configs.Tags,
		"commit_sha":       configs.CommitSHA,
		"build_server_url": configs.BuildServerURL,
		"repository_url":   configs.RepositoryURL,
	}

	files := map[string]string{
		"ipa": apkPath,
	}
	if mappingPath != "" {
		files["dsym"] = mappingPath
	}

	var responseModel ResponseModel
	request, err := createRequest(requestURL, fields, files)
	if err != nil {
		log.Errorf("Failed to create request, error: %s", err)
		return responseModel, err
	}
	request.Header.Add("X-HockeyAppToken", configs.APIToken)

	client := http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Errorf("Performing request failed, error: %s", err)
		return responseModel, err
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Warnf("Failed to close response body, error: %s", err)
		}
	}()

	contents, readErr := ioutil.ReadAll(response.Body)

	if response.StatusCode != http.StatusCreated {
		if readErr != nil {
			log.Warnf("Failed to read response body, error: %s", readErr)
		} else {
			fmt.Println()
			log.Infof("Response:")
			log.Printft(" status code: %d", response.StatusCode)
			log.Printft(" body: %s", string(contents))
		}

		log.Errorf("Performing request failed, status code: %d", response.StatusCode)
		return responseModel, readErr
	}

	// Success
	log.Donef("Request succeeded")

	fmt.Println()
	log.Infof("Response:")
	log.Printf(" status code: %d", response.StatusCode)
	log.Printf(" body: %s", contents)

	if readErr != nil {
		log.Errorf("Failed to read response body, error: %s", readErr)
		return responseModel, readErr
	}

	if err := json.Unmarshal([]byte(contents), &responseModel); err != nil {
		log.Errorf("Failed to parse response body, error: %s", err)
		return responseModel, err
	}

	fmt.Println()
	if responseModel.PublicURL != "" {
		log.Donef("Public URL: %s", responseModel.PublicURL)
	}
	if responseModel.BuildURL != "" {
		log.Donef("Build (direct download) URL: %s", responseModel.BuildURL)
	}
	if responseModel.ConfigURL != "" {
		log.Donef("Config URL: %s", responseModel.ConfigURL)
	}
	return responseModel, nil
}