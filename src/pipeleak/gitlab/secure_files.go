package gitlab

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/CompassSecurity/pipeleak/helper"
	"github.com/tidwall/gjson"
)

func GetSecureFiles(projectId int, base string, token string) (error, []int64) {
	u, err := url.Parse(base)
	if err != nil {
		return err, []int64{}
	}

	client := helper.GetNonVerifyingHTTPClient()
	// https://docs.gitlab.com/ee/api/secure_files.html#download-secure-file
	// pagination does not exist here
	u.Path = "/api/v4/projects/" + strconv.Itoa(projectId) + "/secure_files"
	s := u.String()
	req, err := http.NewRequest("GET", s, nil)
	if err != nil {
		return err, []int64{}
	}
	req.Header.Add("PRIVATE-TOKEN", token)
	res, err := client.Do(req)
	if err != nil {
		return err, []int64{}
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err, []int64{}
	}

	fileIds := []int64{}
	if res.StatusCode == 200 {
		result := gjson.Get(string(body), "@this")
		result.ForEach(func(key, value gjson.Result) bool {
			id := value.Get("id").Int()
			fileIds = append(fileIds, id)
			return true
		})

		return nil, fileIds
	}

	return errors.New("unable to fetch secure files"), []int64{}
}

func DownloadSecureFile(projectId int, fileId int64, base string, token string) (error, []byte, string) {
	u, err := url.Parse(base)
	if err != nil {
		return err, []byte{}, ""
	}

	client := helper.GetNonVerifyingHTTPClient()
	// https://docs.gitlab.com/ee/api/secure_files.html#download-secure-file
	u.Path = "/api/v4/projects/" + strconv.Itoa(projectId) + "/secure_files/" + strconv.Itoa(int(fileId)) + "/download"
	s := u.String()
	req, err := http.NewRequest("GET", s, nil)
	if err != nil {
		return err, []byte{}, ""
	}
	req.Header.Add("PRIVATE-TOKEN", token)
	res, err := client.Do(req)
	if err != nil {
		return err, []byte{}, ""
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err, []byte{}, ""
	}

	return nil, body, s
}
