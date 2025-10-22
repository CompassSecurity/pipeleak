package securefiles

import (
	"errors"
	"io"
	"net/url"
	"strconv"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/tidwall/gjson"
)

func GetSecureFiles(projectId int, base string, token string) ([]int64, error) {
	u, err := url.Parse(base)
	if err != nil {
		return []int64{}, err
	}

	client := helper.GetPipeleakHTTPClient("", nil, nil)
	// https://docs.gitlab.com/ee/api/secure_files.html#download-secure-file
	// pagination does not exist here
	u.Path = "/api/v4/projects/" + strconv.Itoa(projectId) + "/secure_files"
	s := u.String()
	req, err := retryablehttp.NewRequest("GET", s, nil)
	if err != nil {
		return []int64{}, err
	}
	req.Header.Add("PRIVATE-TOKEN", token)
	res, err := client.Do(req)
	if err != nil {
		return []int64{}, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []int64{}, err
	}

	fileIds := []int64{}
	if res.StatusCode == 200 {
		result := gjson.Get(string(body), "@this")
		result.ForEach(func(key, value gjson.Result) bool {
			id := value.Get("id").Int()
			fileIds = append(fileIds, id)
			return true
		})

		return fileIds, nil
	}

	return []int64{}, errors.New("unable to fetch secure files")
}

func DownloadSecureFile(projectId int, fileId int64, base string, token string) ([]byte, string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return []byte{}, "", err
	}

	client := helper.GetPipeleakHTTPClient("", nil, nil)
	// https://docs.gitlab.com/ee/api/secure_files.html#download-secure-file
	u.Path = "/api/v4/projects/" + strconv.Itoa(projectId) + "/secure_files/" + strconv.Itoa(int(fileId)) + "/download"
	s := u.String()
	req, err := retryablehttp.NewRequest("GET", s, nil)
	if err != nil {
		return []byte{}, "", err
	}
	req.Header.Add("PRIVATE-TOKEN", token)
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, "", err
	}

	return body, s, nil
}
