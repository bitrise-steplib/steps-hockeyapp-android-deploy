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
	"github.com/bitrise-io/go-utils/cmdex"
	log "github.com/bitrise-steplib/steps-hockeyapp-android-deploy/logger"
)

// -----------------------
// --- Constants
// -----------------------

const (
	hockeyAppDeployStatusKey     = "HOCKEYAPP_DEPLOY_STATUS"
	hockeyAppDeployStatusSuccess = "success"
	hockeyAppDeployStatusFailed  = "failed"
	hockeyAppDeployPublicURLKey  = "HOCKEYAPP_DEPLOY_PUBLIC_URL"
	hockeyAppDeployBuildURLKey   = "HOCKEYAPP_DEPLOY_BUILD_URL"
	hockeyAppDeployConfigURLKey  = "HOCKEYAPP_DEPLOY_CONFIG_URL"
)

// -----------------------
// --- Models
// -----------------------

// ConfigsModel ...
type ConfigsModel struct {
	ApkPath        string
	MappingPath    string
	APIToken       string
	AppID          string
	Notes          string
	NotesType      string
	Notify         string
	Status         string
	Tags           string
	CommitSHA      string
	BuildServerURL string
	RepositoryURL  string
	Mandatory      string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		ApkPath:        os.Getenv("apk_path"),
		MappingPath:    os.Getenv("mapping_path"),
		APIToken:       os.Getenv("api_token"),
		AppID:          os.Getenv("app_id"),
		Notes:          os.Getenv("notes"),
		NotesType:      os.Getenv("notes_type"),
		Notify:         os.Getenv("notify"),
		Status:         os.Getenv("status"),
		Tags:           os.Getenv("tags"),
		CommitSHA:      os.Getenv("commit_sha"),
		BuildServerURL: os.Getenv("build_server_url"),
		RepositoryURL:  os.Getenv("repository_url"),
		Mandatory:      os.Getenv("mandatory"),
	}
}

func (configs ConfigsModel) print() {
	log.Info("Configs:")
	log.Detail("- ApkPath: %s", configs.ApkPath)
	log.Detail("- MappingPath: %s", configs.MappingPath)
	log.Detail("- APIToken: %s", configs.APIToken)
	log.Detail("- AppID: %s", configs.AppID)
	log.Detail("- Notes: %s", configs.Notes)
	log.Detail("- NotesType: %s", configs.NotesType)
	log.Detail("- Notify: %s", configs.Notify)
	log.Detail("- Status: %s", configs.Status)
	log.Detail("- Tags: %s", configs.Tags)
	log.Detail("- CommitSHA: %s", configs.CommitSHA)
	log.Detail("- BuildServerURL: %s", configs.BuildServerURL)
	log.Detail("- RepositoryURL: %s", configs.RepositoryURL)
	log.Detail("- Mandatory: %s", configs.Mandatory)
}

func (configs ConfigsModel) validate() error {
	// required
	if configs.ApkPath == "" {
		return errors.New("No ApkPath parameter specified!")
	}
	if exist, err := pathutil.IsPathExists(configs.ApkPath); err != nil {
		return fmt.Errorf("Failed to check if ApkPath exist at: %s, error: %s", configs.ApkPath, err)
	} else if !exist {
		return fmt.Errorf("ApkPath not exist at: %s", configs.ApkPath)
	}

	if configs.APIToken == "" {
		return errors.New("No APIToken parameter specified!")
	}

	if configs.NotesType == "" {
		return errors.New("No NotesType parameter specified!")
	}

	if configs.Notify == "" {
		return errors.New("No Notify parameter specified!")
	}

	if configs.Status == "" {
		return errors.New("No Status parameter specified!")
	}

	if configs.Mandatory == "" {
		return errors.New("No Mandatory parameter specified!")
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

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := cmdex.NewCommand("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
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

	w.Close()

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
		log.Fail("Issue with input: %s", err)
	}

	if configs.Mandatory == "1" || configs.Mandatory == "true" {
		configs.Mandatory = "1"
	} else {
		configs.Mandatory = "0"
	}

	//
	// Create request
	log.Info("Performing request")

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
		"ipa": configs.ApkPath,
	}
	if configs.MappingPath != "" {
		files["dsym"] = configs.MappingPath
	}

	request, err := createRequest(requestURL, fields, files)
	if err != nil {
		log.Fail("Failed to create request, error: %#v", err)
	}
	request.Header.Add("X-HockeyAppToken", configs.APIToken)

	client := http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Fail("Performing request failed, error: %#v", err)
	}

	defer response.Body.Close()

	contents, readErr := ioutil.ReadAll(response.Body)

	if response.StatusCode < 200 || response.StatusCode > 300 {
		if readErr != nil {
			log.Warn("Failed to read response body, error: %#v", readErr)
		} else {
			log.Info("Response:")
			log.Detail("status code: %d", response.StatusCode)
			log.Detail("body: %s", string(contents))
		}
		log.Fail("Performing request failed, status code: %d", response.StatusCode)
	}

	// Success
	log.Done("Request succed")

	log.Info("Response:")
	log.Detail("status code: %d", response.StatusCode)
	log.Detail("body: %s", contents)

	if readErr != nil {
		log.Fail("Failed to read response body, error: %#v", readErr)
	}

	var responseModel ResponseModel
	if err := json.Unmarshal([]byte(contents), &responseModel); err != nil {
		log.Fail("Failed to parse response body, error: %#v", err)
	}

	fmt.Println()
	if responseModel.PublicURL != "" {
		log.Done("Public URL: %s", responseModel.PublicURL)
	}
	if responseModel.BuildURL != "" {
		log.Done("Build (direct download) URL: %s", responseModel.BuildURL)
	}
	if responseModel.ConfigURL != "" {
		log.Done("Config URL: %s", responseModel.ConfigURL)
	}

	if err := exportEnvironmentWithEnvman(hockeyAppDeployStatusKey, hockeyAppDeployStatusSuccess); err != nil {
		log.Fail("Failed to export %s, error: %#v", hockeyAppDeployStatusKey, err)
	}

	if err := exportEnvironmentWithEnvman(hockeyAppDeployPublicURLKey, responseModel.PublicURL); err != nil {
		log.Fail("Failed to export %s, error: %#v", hockeyAppDeployPublicURLKey, err)
	}

	if err := exportEnvironmentWithEnvman(hockeyAppDeployBuildURLKey, responseModel.BuildURL); err != nil {
		log.Fail("Failed to export %s, error: %#v", hockeyAppDeployBuildURLKey, err)
	}

	if err := exportEnvironmentWithEnvman(hockeyAppDeployConfigURLKey, responseModel.ConfigURL); err != nil {
		log.Fail("Failed to export %s, error: %#v", hockeyAppDeployConfigURLKey, err)
	}
}
