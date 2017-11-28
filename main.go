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

const (
	hockeyAppDeployStatusKey     = "HOCKEYAPP_DEPLOY_STATUS"
	hockeyAppDeployStatusSuccess = "success"
	hockeyAppDeployStatusFailed  = "failed"

	hockeyAppDeployPublicURLKey = "HOCKEYAPP_DEPLOY_PUBLIC_URL"
	hockeyAppDeployBuildURLKey  = "HOCKEYAPP_DEPLOY_BUILD_URL"
	hockeyAppDeployConfigURLKey = "HOCKEYAPP_DEPLOY_CONFIG_URL"

	hockeyAppDeployPublicURLKeyList = "HOCKEYAPP_DEPLOY_PUBLIC_URL_LIST"
	hockeyAppDeployBuildURLKeyList  = "HOCKEYAPP_DEPLOY_BUILD_URL_LIST"
	hockeyAppDeployConfigURLKeyList = "HOCKEYAPP_DEPLOY_CONFIG_URL_LIST"
)

var configs ConfigsModel

// ConfigsModel ...
type ConfigsModel struct {
	ApkPath        []string
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

	mandatory := os.Getenv("mandatory")
	if mandatory == "1" || mandatory == "true" {
		mandatory = "1"
	} else {
		mandatory = "0"
	}

	return ConfigsModel{
		ApkPath:        strings.Split(os.Getenv("apk_path"), "|"),
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
		Mandatory:      mandatory,
	}
}

func (configs ConfigsModel) print() {
	fmt.Println()
	log.Infof("Configs:")
	log.Printf(" - ApkPath: %s", configs.ApkPath)
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

func (configs ConfigsModel) validate() error {
	if len(configs.ApkPath) == 0 {
		return errors.New("no ApkPath parameter specified")
	}

	for _, apkPath := range configs.ApkPath {
		if exist, err := pathutil.IsPathExists(apkPath); err != nil {
			return fmt.Errorf("failed to check if ApkPath exist at: %s, error: %v", apkPath, err)
		} else if !exist {
			return fmt.Errorf("apkPath not exist at: %s", apkPath)
		}
	}

	required := []string{configs.APIToken, configs.NotesType, configs.Notify, configs.Status, configs.Mandatory}
	for _, config := range required {
		if config == "" {
			return fmt.Errorf("no %s parameter specified", config)
		}
	}

	if configs.MappingPath != "" {
		if exist, err := pathutil.IsPathExists(configs.MappingPath); err != nil {
			return fmt.Errorf("failed to check if MappingPath exist at: %s, error: %v", configs.MappingPath, err)
		} else if !exist {
			return fmt.Errorf("mappingPath not exist at: %s", configs.MappingPath)
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
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func createRequest(url string, fields, files map[string]string) (*http.Request, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return nil, err
		}
	}

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

	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}

func deploy(apkPath string) (ResponseModel, error) {
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
	if configs.MappingPath != "" {
		files["dsym"] = configs.MappingPath
	}

	request, err := createRequest(requestURL, fields, files)
	if err != nil {
		return ResponseModel{}, fmt.Errorf("Failed to create request, error: %v", err)
	}

	request.Header.Add("X-HockeyAppToken", configs.APIToken)
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return ResponseModel{}, fmt.Errorf("Performing request failed, error: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Warnf("Failed to close response body, error: %v", err)
		}
	}()

	contents, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return ResponseModel{}, fmt.Errorf("Failed to read response body, error: %v", readErr)
	} else if response.StatusCode < 200 || response.StatusCode > 300 {
		return ResponseModel{}, fmt.Errorf("Performing request failed, status code: %d", response.StatusCode)
	}

	log.Donef("Request succeeded")
	fmt.Println()
	log.Infof("Response:")
	log.Printf(" status code: %d", response.StatusCode)
	log.Printf(" body: %s", contents)

	responseModel := ResponseModel{}
	if err := json.Unmarshal([]byte(contents), &responseModel); err != nil {
		return ResponseModel{}, fmt.Errorf("Failed to parse response body, error: %v", err)
	}
	return responseModel, nil
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

func main() {
	configs = createConfigsModelFromEnvs()
	configs.print()
	if err := configs.validate(); err != nil {
		log.Errorf("Issue with input: %s", err)
		os.Exit(1)
	}

	configURLs := []string{}
	buildURLs := []string{}
	publicURLs := []string{}

	for _, apkPath := range configs.ApkPath {
		responseModel, err := deploy(apkPath)
		if err != nil {
			log.Errorf("Hockeyapp deploy failed: %v", err)
			if err := exportEnvironmentWithEnvman(hockeyAppDeployStatusKey, hockeyAppDeployStatusFailed); err != nil {
				log.Warnf("Failed to export %s, error: %v", hockeyAppDeployStatusKey, err)
			}
			os.Exit(1)
		}
		if responseModel.ConfigURL != "" && !contains(configURLs, responseModel.ConfigURL) {
			configURLs = append(configURLs, responseModel.ConfigURL)
			log.Donef("Config URL: %s", responseModel.ConfigURL)
		}
		if responseModel.BuildURL != "" && !contains(buildURLs, responseModel.BuildURL) {
			buildURLs = append(buildURLs, responseModel.BuildURL)
			log.Donef("Build (direct download) URL: %s", responseModel.BuildURL)
		}
		if responseModel.PublicURL != "" && !contains(publicURLs, responseModel.PublicURL) {
			publicURLs = append(publicURLs, responseModel.PublicURL)
			log.Donef("Public URL: %s", responseModel.PublicURL)
		}
	}

	outputs := map[string]string{
		hockeyAppDeployStatusKey:        hockeyAppDeployStatusSuccess,
		hockeyAppDeployConfigURLKeyList: strings.Join(configURLs, "|"),
		hockeyAppDeployBuildURLKeyList:  strings.Join(buildURLs, "|"),
		hockeyAppDeployPublicURLKeyList: strings.Join(publicURLs, "|"),
	}
	if len(configURLs) > 0 {
		outputs[hockeyAppDeployConfigURLKey] = configURLs[len(configURLs)-1]
	}
	if len(buildURLs) > 0 {
		outputs[hockeyAppDeployBuildURLKey] = buildURLs[len(buildURLs)-1]
	}
	if len(publicURLs) > 0 {
		outputs[hockeyAppDeployPublicURLKey] = publicURLs[len(publicURLs)-1]
	}

	for k, v := range outputs {
		if err := exportEnvironmentWithEnvman(k, v); err != nil {
			log.Warnf("Failed to export %s, error: %v", k, err)
		}
	}
}
