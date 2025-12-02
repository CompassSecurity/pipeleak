package securefiles

import (
	"errors"
	"io"
	"net/url"
	"strconv"

	"github.com/CompassSecurity/pipeleek/pkg/httpclient"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/tidwall/gjson"
)

func GetSecureFiles(projectId int64, base string, token string) ([]int64, error) {
	u, err := url.Parse(base)
	if err != nil {
		return []int64{}, err
	}

	client := httpclient.GetPipeleekHTTPClient("", nil, nil)
	// https://docs.gitlab.com/ee/api/secure_files.html#download-secure-file
	// pagination does not exist here
	u.Path = "/api/v4/projects/" + strconv.FormatInt(projectId, 10) + "/secure_files"
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

func DownloadSecureFile(projectId int64, fileId int64, base string, token string) ([]byte, string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return []byte{}, "", err
	}

	client := httpclient.GetPipeleekHTTPClient("", nil, nil)
	// https://docs.gitlab.com/ee/api/secure_files.html#download-secure-file
	u.Path = "/api/v4/projects/" + strconv.FormatInt(projectId, 10) + "/secure_files/" + strconv.FormatInt(fileId, 10) + "/download"
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
