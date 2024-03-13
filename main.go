package main

import (
	"ezoneMavenUpload/cmd"
	"ezoneMavenUpload/services"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"log/slog"
	"sync"
)

func main() {
	cmd.Flag()
	fmt.Println("开始查找本地maven仓库内的依赖")

	var err error
	var mip []services.MavenInfoPath
	if mip, err = services.FindMavenRepos(cmd.RepoPath); err != nil {
		slog.Error(fmt.Sprintf("查找本地maven仓库失败: %s", err))
	}
	fmt.Println(fmt.Sprintf("在本地maven仓库: %v; 查找到依赖: %v个", cmd.RepoPath, len(mip)))

	var ezOne services.EzOne
	if ezOne, err = services.NewEzOne(
		cmd.EzOneBaseUrl,
		cmd.EzOneKey,
		cmd.EzOneRepoId,
		services.EzOneWithEnterprise(cmd.EzOneEnterprise),
	); err != nil {
		slog.Error(fmt.Sprintf("连接EzOne失败: %s", err))
		return
	}

	fmt.Println("开始上传本地maven仓库依赖")
	var wg sync.WaitGroup
	ch := make(chan struct{}, cmd.WorksNum)
	bar := pb.StartNew(len(mip))
	for i := range mip {
		wg.Add(1)
		ch <- struct{}{}
		go func(mip services.MavenInfoPath) {
			ezOne.UploadPkgToRepo(mip.GroupId, mip.ArtifactId, mip.Version, mip.Path)
			<-ch
			wg.Done()
			bar.Increment()
		}(mip[i])
	}
	wg.Wait()
	close(ch)
	bar.Finish()
	fmt.Println("完成本地maven仓库上传!")
}
