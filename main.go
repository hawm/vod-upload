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

func UploadMediaWithCallback(client *vod.Vod, spaceName, filePath, fileName string) (string, error) {
	optionFunc := functions.AddOptionInfoFunc(business.VodUploadFunctionInput{
		Title: fileName,
	})
	vodFunctions := []business.VodUploadFunction{optionFunc}
	fbts, _ := json.Marshal(vodFunctions)

	vodUploadMediaRequest := &request.VodUploadMediaRequest{
		SpaceName: spaceName,
		FilePath:  filePath,
		FileName:  fileName,
		Functions: string(fbts),
	}

	resp, _, err := client.UploadMediaWithCallback(vodUploadMediaRequest)

	if err != nil {
		return "", err
	}

	respErr := resp.GetResponseMetadata().GetError()
	if respErr != nil {
		return "", errors.New(respErr.Code)
	}

	vid := resp.GetResult().GetData().GetVid()

	return vid, nil
}

func isVideoFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	return extension == ".mp4" || extension == ".avi" || extension == ".mov" || extension == ".mkv"
}

type VideoUploadResult struct {
	FileName  string
	Vid       string
	Published bool
	Error     error
}

func moveFile(path string, directoryPath string) error {
	newPath := directoryPath + "/" + filepath.Base(path)
	err := os.Rename(path, newPath)
	if err != nil {
		fmt.Println("Error:", err)
	}
	return err
}

func uploadVideosInDirectory(client *vod.Vod, spaceName, inputDirectoryPath, outputDirectoryPath string) ([]VideoUploadResult, error) {
	var results []VideoUploadResult

	err := filepath.Walk(inputDirectoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isVideoFile(path) {
			fileName := filepath.Base(path)
			fmt.Printf("Uploading video: %s\n", fileName)
			vid, uploadErr := UploadMediaWithCallback(client, spaceName, path, fileName)

			if uploadErr != nil {
				if uploadErr.Error() == "SignatureDoesNotMatch" || uploadErr.Error() == "InvalidCredential" {
					return uploadErr
				}
				results = append(results, VideoUploadResult{
					FileName: fileName, Vid: "", Error: uploadErr, Published: false,
				})
			} else {
				published, publishErr := publishVideo(client, vid)
				results = append(results, VideoUploadResult{
					FileName: fileName, Vid: vid, Error: publishErr, Published: published,
				})
				err = moveFile(path, outputDirectoryPath)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return results, err
}

func createResultsINI(results []VideoUploadResult, outputPath string) error {
	fmt.Printf("Creating %s\n", outputPath)

	cfg, err := ini.Load([]byte{})

	if err != nil {
		return fmt.Errorf("error creating ini file: %v", err)
	}

	for _, result := range results {
		section := cfg.Section(result.FileName)

		section.Key("vid").SetValue(result.Vid)
		section.Key("published").SetValue(fmt.Sprintf("%t", result.Published))

		if result.Error != nil {
			section.Key("error").SetValue(result.Error.Error())
		}
	}

	err = cfg.SaveTo(outputPath)
	if err != nil {
		return fmt.Errorf("error saving ini file: %v", err)
	}

	return nil
}

func isDirectoryExists(directoryPath string) bool {

	absolutePath, err := filepath.Abs(directoryPath)

	if err != nil {
		return false
	}

	if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
		fmt.Printf("Directory %s does not exist\n", directoryPath)
		return false
	}

	return true
}

func cliParams() (string, string, string) {
	var spaceName, inputDir, outputDir string

	flag.StringVar(&spaceName, "spacename", "", "Space name")
	flag.StringVar(&inputDir, "inputdir", "", "Input directory")
	flag.StringVar(&outputDir, "outputdir", "", "Output directory")

	flag.Parse()

	return spaceName, inputDir, outputDir
}

func main() {
	spaceName, inputDirectoryPath, outputDirectoryPath := cliParams()

	resultPath := filepath.Join(outputDirectoryPath, "results.ini")

	if !isDirectoryExists(inputDirectoryPath) || !isDirectoryExists(outputDirectoryPath) {
		os.Exit(1)
	}

	// read VOLC_ACCESSKEY & VOLC_SECRETKEY from env automatically
	client := vod.NewInstanceWithRegion(base.RegionCnNorth1)

	results, err := uploadVideosInDirectory(client, spaceName, inputDirectoryPath, outputDirectoryPath)

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
	createResultsINI(results, resultPath)
}
