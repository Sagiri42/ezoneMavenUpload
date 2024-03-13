package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type EzOne struct {
	BaseUrl    *url.URL
	Enterprise string
	Token      string
	RepoId     string
}

type EzOneOption func(*EzOne)

func EzOneWithEnterprise(ezEnterprise string) EzOneOption {
	return func(e *EzOne) {
		e.Enterprise = ezEnterprise
	}
}

type EzOneResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type EzLibraryInfo struct {
	RepoId   string `json:"repoId"`
	RepoName string `json:"repoName"`
	RepoType string `json:"repoType"`
}

type EzPackageInfo struct {
	PkgName string `json:"pkgName"`
	PkgId   string `json:"pkgId"`
}

type EzCreatePackage struct {
	RepoId     string `json:"repoId"`
	PkgName    string `json:"pkgName"`
	PkgNameSec string `json:"pkgNameSec"`
}

// NewEzOne 创建EzOne服务
func NewEzOne(ezBaseurl, ezToken, ezRepoId string, options ...EzOneOption) (e EzOne, err error) {
	slog.Debug("开始创建EzOne结构体")
	if e.BaseUrl, err = url.Parse(ezBaseurl); err != nil {
		slog.Error(fmt.Sprint("url格式错误:", e.BaseUrl))
		return
	}
	e.Token = ezToken
	e.RepoId = ezRepoId

	for i := range options {
		options[i](&e)
	}
	if err = e.QueryRepository(); err != nil {
		slog.Error(fmt.Sprint("不存在制品库:", e.RepoId))
	}
	return
}

// splicingBaseUrl 拼接url
func (e EzOne) splicingBaseUrl(apiUrl string, params map[string]string) string {
	queryParams := url.Values{}
	queryParams.Set("access_token", e.Token)
	for i := range params {
		queryParams.Set(i, params[i])
	}

	if urls := strings.Split(apiUrl, "/"); e.Enterprise != "" && len(urls) >= 2 {
		apiUrl = strings.Join(append([]string{urls[1], e.Enterprise}, urls[2:]...), "/")
		slog.Debug(fmt.Sprint("已启用多组织, 添加组织至接口:", apiUrl))
	}

	apiPath, _ := url.Parse(apiUrl)
	fullPath := e.BaseUrl.ResolveReference(apiPath)
	fullPath.RawQuery = queryParams.Encode()

	slog.Debug(fmt.Sprint("已拼接url:", fullPath.String()))
	return fullPath.String()
}

// QueryRepository 查询制品库是否存在
func (e EzOne) QueryRepository() (err error) {
	apiUrl := e.splicingBaseUrl("/v1/package/api/repository/simpleInfo", map[string]string{
		"repoId": e.RepoId,
	})
	slog.Debug(fmt.Sprint("查询制品库是否存在:", apiUrl))

	var response *http.Response
	if response, err = http.Get(apiUrl); err != nil {
		return
	}
	defer response.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(response.Body); err != nil {
		return
	}

	var bodyData EzOneResponse[EzLibraryInfo]
	if err = json.Unmarshal(bodyBytes, &bodyData); err != nil {
		return
	}

	if bodyData.Code != 0 {
		return errors.New(bodyData.Message)
	}
	slog.Info(fmt.Sprint("制品库存在, 当前制品库名:", bodyData.Data.RepoName))
	return
}

// QueryPackage 查询制品包是否存在
func (e EzOne) QueryPackage(pkgName string) (pkgId string, err error) {
	apiUrl := e.splicingBaseUrl("/v1/package/api/package/simpleList", map[string]string{
		"repoId":   e.RepoId,
		"pkgName":  pkgName,
		"pageSize": "0",
	})
	slog.Debug("查询制品包是否存在:", apiUrl)

	var response *http.Response
	if response, err = http.Get(apiUrl); err != nil {
		return
	}
	defer response.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(response.Body); err != nil {
		return
	}

	var bodyData EzOneResponse[[]EzPackageInfo]
	if err = json.Unmarshal(bodyBytes, &bodyData); err != nil {
		return
	}
	if bodyData.Code != 0 {
		return "", errors.New(bodyData.Message)
	}
	for i := range bodyData.Data {
		if bodyData.Data[i].PkgName == pkgName {
			return bodyData.Data[i].PkgId, nil
		}
	}
	return "", nil
}

// QueryVersion 查询制品版本是否存在
func (e EzOne) QueryVersion(pkgId, version string) (isExist bool, err error) {
	apiUrl := e.splicingBaseUrl("/v1/package/api/artifact/isExist", map[string]string{
		"pkgId":   pkgId,
		"version": version,
		"format":  "maven",
	})
	slog.Debug(fmt.Sprint("查询制品库版本是否存在:", apiUrl))

	var response *http.Response
	if response, err = http.Get(apiUrl); err != nil {
		return
	}
	defer response.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(response.Body); err != nil {
		return
	}
	var bodyData EzOneResponse[bool]
	if err = json.Unmarshal(bodyBytes, &bodyData); err != nil {
		return
	}

	if bodyData.Code != 0 {
		return false, errors.New(bodyData.Message)
	}

	return bodyData.Data, nil
}

// CreatePackage 新建制品包
func (e EzOne) CreatePackage(groupName, pkgName string) (packageId string, err error) {
	apiUrl := e.splicingBaseUrl("/v1/package/api/package", nil)
	slog.Debug(fmt.Sprint("创建制品包:", apiUrl))

	var requestBody []byte
	if requestBody, err = json.Marshal(EzCreatePackage{
		RepoId:     e.RepoId,
		PkgName:    groupName,
		PkgNameSec: pkgName,
	}); err != nil {
		return
	}

	var response *http.Response
	if response, err = http.Post(apiUrl, "application/json", bytes.NewReader(requestBody)); err != nil {
		return
	}
	defer response.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(response.Body); err != nil {
		return
	}
	var bodyData EzOneResponse[EzPackageInfo]
	if err = json.Unmarshal(bodyBytes, &bodyData); err != nil {
		return
	}

	if bodyData.Code != 0 {
		return "", errors.New(bodyData.Message)
	}

	return bodyData.Data.PkgId, nil
}

// UploadPackage 上传制品到指定的制品包版本
func (e EzOne) UploadPackage(pkgId, version, filePath string) (err error) {
	apiUrl := e.splicingBaseUrl("/v1/package/api/artifact/upload", map[string]string{})
	slog.Debug(fmt.Sprint("上传制品包版本:", apiUrl))

	var fileOpen *os.File
	if fileOpen, err = os.Open(filePath); err != nil {
		return
	}
	defer fileOpen.Close()

	requestBody := bytes.Buffer{}
	multipartWriter := multipart.NewWriter(&requestBody)

	var fileWriter io.Writer
	if fileWriter, err = multipartWriter.CreateFormFile("file", filepath.Base(filePath)); err != nil {
		return
	}
	if _, err = io.Copy(fileWriter, fileOpen); err != nil {
		return
	}
	_ = multipartWriter.WriteField("classifier", "")
	_ = multipartWriter.WriteField("pkgId", pkgId)
	_ = multipartWriter.WriteField("version", version)
	if err = multipartWriter.Close(); err != nil {
		return
	}

	var response *http.Response
	if response, err = http.Post(apiUrl, multipartWriter.FormDataContentType(), &requestBody); err != nil {
		return
	}
	defer response.Body.Close()

	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(response.Body); err != nil {
		return
	}
	var bodyData EzOneResponse[types.Nil]
	if err = json.Unmarshal(bodyBytes, &bodyData); err != nil {
		return
	}

	if bodyData.Code != 0 {
		return errors.New(bodyData.Message)
	}
	return nil
}

func (e EzOne) UploadPkgToRepo(groupId, artifactId, version, path string) {
	var err error
	var pkgId string
	pkgName := fmt.Sprintf("%s:%s", groupId, artifactId)
	if pkgId, err = e.QueryPackage(pkgName); err != nil {
		slog.Error(fmt.Sprintf("制品包 %s 查询失败:", pkgName), err)
		return
	} else if pkgId == "" {
		if pkgId, err = e.CreatePackage(groupId, artifactId); err != nil || pkgId == "" {
			slog.Error(fmt.Sprintf("制品包 %s 创建失败:", pkgName), err)
			return
		}
		slog.Info(fmt.Sprintf("已创建制品包: %s; 制品包ID: %s", pkgName, pkgId))
	}
	slog.Info(fmt.Sprintf("已存在制品包: %s; 制品ID: %s", pkgName, pkgId))
	if err = e.UploadPackage(pkgId, version, path); err != nil {
		slog.Error(fmt.Sprintf("上传制品 %s 版本 %s 创建失败; 错误: %s", pkgName, version, err))
	} else {
		slog.Info(fmt.Sprintf("已上传制品: %s; 版本: %s", pkgName, version))
	}
}
