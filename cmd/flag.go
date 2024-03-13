package cmd

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"runtime"
)

const Version = "1.3.0"

var (
	showVersion     bool
	RepoPath        string
	EzOneBaseUrl    string
	EzOneKey        string
	EzOneEnterprise string
	EzOneRepoId     string
	WorksNum        int
)

func Flag() {
	pflag.BoolVarP(&showVersion, "version", "v", false, "版本号")
	pflag.StringVarP(&RepoPath, "repoPath", "p", "~/.m2/repositor", "本地仓库路径")
	pflag.StringVarP(&EzOneBaseUrl, "ezOneBaseUrl", "u", "", "简单云BaseURL")
	pflag.StringVarP(&EzOneEnterprise, "ezOneEnterprise", "e", "", "简单云企业名")
	pflag.StringVarP(&EzOneKey, "ezOneKey", "k", "", "简单云个人令牌")
	pflag.StringVarP(&EzOneRepoId, "ezOneRepoId", "r", "", "简单云制品库ID")
	pflag.IntVar(&WorksNum, "worksNum", runtime.NumCPU(), "运行协程数")

	pflag.Parse()

	version()

	if EzOneBaseUrl == "" || EzOneKey == "" || EzOneRepoId == "" {
		fmt.Println("参数缺失, 请完善参数: 简单云BaseURL, 简单云个人令牌, 简单云制品库ID")
		os.Exit(1)
	}
}

func version() {
	if showVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
}
