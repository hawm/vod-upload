package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"

	"github.com/volcengine/volc-sdk-golang/base"
	"github.com/volcengine/volc-sdk-golang/service/vod"
	"github.com/volcengine/volc-sdk-golang/service/vod/models/business"
	"github.com/volcengine/volc-sdk-golang/service/vod/models/request"
	"github.com/volcengine/volc-sdk-golang/service/vod/upload/functions"
)

func publishVideo(client *vod.Vod, vid string) (bool, error) {
	query := &request.VodUpdateMediaPublishStatusRequest{
		Vid:    vid,
		Status: "Published",
	}

	resp, _, err := client.UpdateMediaPublishStatus(query)

	if err != nil {
		return false, err
	}

	respErr := resp.GetResponseMetadata().GetError()

	if respErr != nil {
		return false, errors.New(respErr.Code)
	}

	return true, nil
}

func uploadMedia(client *vod.Vod, spaceName, filePath, title, uploadPath string) (string, string, error) {
	optionFunc := functions.AddOptionInfoFunc(business.VodUploadFunctionInput{
		Title: title,
	})
	vodFunctions := []business.VodUploadFunction{optionFunc}
	fbts, _ := json.Marshal(vodFunctions)

	vodUploadMediaRequest := &request.VodUploadMediaRequest{
		SpaceName: spaceName,
		FilePath:  filePath,
		Functions: string(fbts),
		FileName:  uploadPath,
	}

	resp, _, err := client.UploadMediaWithCallback(vodUploadMediaRequest)

	if err != nil {
		return "", "", err
	}

	respErr := resp.GetResponseMetadata().GetError()
	if respErr != nil {
		return "", "", errors.New(respErr.Code)
	}

	fileName := resp.GetResult().GetData().GetSourceInfo().GetFileName()
	vid := resp.GetResult().GetData().GetVid()

	return fileName, vid, nil
}

func upload(client *vod.Vod, spaceName, filePath, title, uploadPath string) (string, string, bool, bool, error) {
	uploadPath, vid, err := uploadMedia(client, spaceName, filePath, title, uploadPath)

	if err != nil {
		return title, uploadPath, false, false, err
	}

	published, err := publishVideo(client, vid)

	return title, uploadPath, true, published, err
}

func cliParams() (string, string, string, string) {
	var spaceName, filePath, title, uploadPath string

	flag.StringVar(&spaceName, "spacename", "", "Space name")
	flag.StringVar(&filePath, "filepath", "", "Filepath being upload")
	flag.StringVar(&title, "title", "", "Title of media")
	flag.StringVar(&uploadPath, "uploadpath", "", "Filepath of media on remote")

	flag.Parse()

	if title == "" {
		title = filepath.Base(filePath)
	}

	return spaceName, filePath, title, uploadPath
}

func readConfigINI(path string) (string, string, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return "", "", fmt.Errorf("error loading ini file: %v", err)
	}

	section := cfg.Section("default")
	if section == nil {
		return "", "", fmt.Errorf("default section not found")
	}

	ak := section.Key("VOLC_ACCESSKEY").String()
	sk := section.Key("VOLC_SECRETKEY").String()

	return ak, sk, nil
}

func formatError(err error) string {
	if err != nil {
		return err.Error()
	}

	return ""
}

func main() {
	spaceName, filePath, title, uploadPath := cliParams()

	currentDir, err := os.Getwd()

	if err != nil {
		fmt.Printf("%s, %s, %s, %t, %t, %s\n", filePath, title, "", false, false, formatError(err))
		os.Exit(1)
	}

	configPath := filepath.Join(currentDir, "config.ini")
	ak, sk, err := readConfigINI(configPath)

	if err != nil {
		fmt.Printf("%s, %s, %s, %t, %t, %s\n", filePath, title, "", false, false, formatError(err))
		os.Exit(1)
	}

	client := vod.NewInstanceWithRegion(base.RegionCnNorth1)
	client.SetCredential(base.Credentials{
		AccessKeyID:     ak,
		SecretAccessKey: sk,
	})

	title, uploadPath, uploaded, published, err := upload(client, spaceName, filePath, title, uploadPath)

	fmt.Printf("%s, %s, %s, %t, %t, %s\n", filePath, title, uploadPath, uploaded, published, formatError(err))
}
